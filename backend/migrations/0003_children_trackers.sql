-- +goose Up
-- +goose StatementBegin
CREATE TABLE children (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  birthday TEXT,
  avatar_url TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE trackers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  icon TEXT,
  color TEXT,
  description TEXT,
  schema_json TEXT NOT NULL,
  is_archived INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_trackers_archived ON trackers(is_archived);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE trackers;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE children;
-- +goose StatementEnd
