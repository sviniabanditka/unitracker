-- +goose Up
-- +goose StatementBegin
CREATE TABLE snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  filename TEXT NOT NULL UNIQUE,
  size_bytes INTEGER NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('auto','manual','pre-restore')),
  note TEXT,
  created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX idx_snapshots_type_created ON snapshots(type, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE snapshots;
-- +goose StatementEnd
