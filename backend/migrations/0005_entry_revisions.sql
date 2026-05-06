-- +goose Up
-- +goose StatementBegin
CREATE TABLE entry_revisions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  entry_id INTEGER NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
  data_json TEXT NOT NULL,
  occurred_at TEXT NOT NULL,
  child_id INTEGER,
  is_deleted INTEGER NOT NULL DEFAULT 0,
  change_type TEXT NOT NULL CHECK (change_type IN ('create','update','delete','restore')),
  changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_entry_revisions_entry ON entry_revisions(entry_id, changed_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO entry_revisions (entry_id, data_json, occurred_at, child_id, is_deleted, change_type, changed_by, changed_at)
SELECT id, data_json, occurred_at, child_id, is_deleted, 'create', created_by, created_at FROM entries;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE entry_revisions;
-- +goose StatementEnd
