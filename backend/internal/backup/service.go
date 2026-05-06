package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	appdb "github.com/bublya/baby-tracker/backend/internal/db"
	"github.com/bublya/baby-tracker/backend/internal/settings"
)

// snapshotFilenameRE captures the timestamp + type from "snapshot-YYYYMMDDTHHMMSSZ-{type}.db".
var snapshotTypes = map[string]bool{
	TypeAuto: true, TypeManual: true, TypePreRestore: true,
}

var (
	ErrInvalidType    = errors.New("invalid snapshot type")
	ErrFileMissing    = errors.New("snapshot file missing")
	ErrRestoreTimeout = errors.New("restore drain timed out")
)

type Service struct {
	db        *appdb.Database
	gate      *appdb.Gate
	store     *snapshotStore
	settings  *settings.Store
	backupDir string

	// migrate is invoked after the pool is reopened during Restore so a snapshot
	// taken from an older schema is upgraded transparently.
	migrate func(driver any) error
}

func NewService(database *appdb.Database, gate *appdb.Gate, settingsStore *settings.Store, dataDir string) (*Service, error) {
	backupDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}
	return &Service{
		db:        database,
		gate:      gate,
		store:     newSnapshotStore(database),
		settings:  settingsStore,
		backupDir: backupDir,
	}, nil
}

func (s *Service) BackupDir() string { return s.backupDir }

// Reconcile scans the backup directory and ensures every snapshot file has a
// matching row. Used at startup and after a restore (which wipes the
// snapshots table because it's part of the restored database).
func (s *Service) Reconcile(ctx context.Context) error {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return err
	}
	existing, err := s.store.List(ctx, ListFilter{Limit: 500})
	if err != nil {
		return err
	}
	known := make(map[string]struct{}, len(existing))
	for _, snap := range existing {
		known[snap.Filename] = struct{}{}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "snapshot-") || !strings.HasSuffix(name, ".db") {
			continue
		}
		if _, ok := known[name]; ok {
			continue
		}
		typ, ts := parseSnapshotName(name)
		if !snapshotTypes[typ] {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			slog.Warn("reconcile stat", "file", name, "err", statErr)
			continue
		}
		recovered := "recovered from disk"
		if _, err := s.store.InsertReconciled(ctx, insertInput{
			Filename:  name,
			SizeBytes: info.Size(),
			Type:      typ,
			Note:      &recovered,
		}, ts); err != nil {
			slog.Warn("reconcile insert", "file", name, "err", err)
		}
	}
	return nil
}

func parseSnapshotName(name string) (string, time.Time) {
	// snapshot-YYYYMMDDTHHMMSSZ-{type}.db
	stripped := strings.TrimSuffix(strings.TrimPrefix(name, "snapshot-"), ".db")
	parts := strings.SplitN(stripped, "-", 2)
	if len(parts) != 2 {
		return "", time.Time{}
	}
	ts, err := time.Parse("20060102T150405Z", parts[0])
	if err != nil {
		ts = time.Time{}
	}
	return parts[1], ts
}

// Create produces a new snapshot via VACUUM INTO. byUserID may be nil for scheduled jobs.
func (s *Service) Create(ctx context.Context, snapType string, note *string, byUserID *int64) (*Snapshot, error) {
	if snapType != TypeAuto && snapType != TypeManual && snapType != TypePreRestore {
		return nil, ErrInvalidType
	}
	filename := fmt.Sprintf("snapshot-%s-%s.db", time.Now().UTC().Format("20060102T150405Z"), snapType)
	path := filepath.Join(s.backupDir, filename)
	if _, err := s.db.Conn().ExecContext(ctx, `VACUUM INTO ?`, path); err != nil {
		return nil, fmt.Errorf("vacuum into: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat snapshot: %w", err)
	}
	snap, err := s.store.Insert(ctx, insertInput{
		Filename:  filename,
		SizeBytes: info.Size(),
		Type:      snapType,
		Note:      note,
		CreatedBy: byUserID,
	})
	if err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	if snapType == TypeAuto {
		if err := s.RetentionApply(ctx); err != nil {
			slog.Warn("backup retention apply", "err", err)
		}
	}
	return snap, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]*Snapshot, error) {
	return s.store.List(ctx, filter)
}

func (s *Service) Get(ctx context.Context, id int64) (*Snapshot, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	snap, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	if rmErr := os.Remove(filepath.Join(s.backupDir, snap.Filename)); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
		slog.Warn("snapshot file delete", "id", id, "filename", snap.Filename, "err", rmErr)
	}
	return nil
}

// Open returns a file handle for download streaming. Caller closes.
func (s *Service) Open(ctx context.Context, id int64) (*os.File, *Snapshot, error) {
	snap, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(filepath.Join(s.backupDir, snap.Filename))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, snap, ErrFileMissing
		}
		return nil, snap, err
	}
	return f, snap, nil
}

// RetentionApply trims the auto snapshots beyond backup_retention_count.
func (s *Service) RetentionApply(ctx context.Context) error {
	keep := s.settings.IntOr(ctx, settings.KeyBackupRetentionCount, 20)
	if keep <= 0 {
		keep = 20
	}
	old, err := s.store.OldestAuto(ctx, keep)
	if err != nil {
		return err
	}
	for _, snap := range old {
		if err := s.store.Delete(ctx, snap.ID); err != nil {
			slog.Warn("retention: db delete", "id", snap.ID, "err", err)
			continue
		}
		path := filepath.Join(s.backupDir, snap.Filename)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("retention: file delete", "path", path, "err", err)
		}
	}
	return nil
}

// Restore replaces the live SQLite file with the snapshot's contents. The
// caller must invoke this from an HTTP handler so we know the request itself
// is counted in the gate WaitGroup (we Pause/Resume it around the drain).
func (s *Service) Restore(ctx context.Context, id int64, byUserID *int64, drainTimeout time.Duration) error {
	snap, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	srcPath := filepath.Join(s.backupDir, snap.Filename)
	if _, err := os.Stat(srcPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrFileMissing
		}
		return err
	}

	preNote := fmt.Sprintf("before restoring snapshot #%d (%s)", snap.ID, snap.Filename)
	if _, err := s.Create(ctx, TypePreRestore, &preNote, byUserID); err != nil {
		return fmt.Errorf("pre-restore snapshot: %w", err)
	}

	s.gate.PauseSelf()
	enterCtx, cancel := context.WithTimeout(ctx, drainTimeout)
	if err := s.gate.Enter(enterCtx); err != nil {
		cancel()
		s.gate.ResumeSelf()
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrRestoreTimeout
		}
		return err
	}
	cancel()

	swapErr := s.swap(ctx, srcPath)

	s.gate.Exit()
	s.gate.ResumeSelf()

	if swapErr == nil {
		// The restored DB has the snapshots table state from when the source
		// snapshot was taken. Re-link all on-disk files (incl. the pre-restore
		// we just created) so the admin still sees them.
		if err := s.Reconcile(ctx); err != nil {
			slog.Warn("reconcile after restore", "err", err)
		}
	}
	return swapErr
}

func (s *Service) swap(ctx context.Context, srcPath string) error {
	dstPath := s.db.Path()

	// Force a checkpoint so all WAL contents land in the main file before close.
	if _, err := s.db.Conn().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		slog.Warn("wal_checkpoint before restore", "err", err)
	}

	old := s.db.Replace(nil)
	if old != nil {
		if err := old.Close(); err != nil {
			slog.Warn("close old pool", "err", err)
		}
	}
	// Remove sidecar files so the new pool opens with a clean WAL.
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(dstPath + suffix)
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		// Best-effort recovery: re-open whatever is there.
		if reOpenErr := s.reopen(); reOpenErr != nil {
			slog.Error("reopen after failed copy", "err", reOpenErr)
		}
		return fmt.Errorf("copy snapshot: %w", err)
	}

	return s.reopen()
}

func (s *Service) reopen() error {
	newDB, err := appdb.Open(s.db.Path())
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	if err := appdb.Migrate(newDB); err != nil {
		_ = newDB.Close()
		return fmt.Errorf("migrate after restore: %w", err)
	}
	s.db.Replace(newDB)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".restore.tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// helper for log fields
func filenameOnly(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}

var _ = filenameOnly
