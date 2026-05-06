-- +goose Up
-- +goose StatementBegin
CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO settings(key, value) VALUES
  ('backup_interval_hours', '6'),
  ('backup_retention_count', '20'),
  ('app_name', 'Baby Tracker'),
  ('default_locale', 'en');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE settings;
-- +goose StatementEnd
