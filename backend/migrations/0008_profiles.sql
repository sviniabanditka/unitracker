-- +goose Up
-- +goose StatementBegin
ALTER TABLE children RENAME TO profiles;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE profiles DROP COLUMN birthday;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE profiles ADD COLUMN description TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM trackers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE entries RENAME COLUMN child_id TO profile_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_entries_child_occurred;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_entries_profile_occurred ON entries(profile_id, occurred_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE entry_revisions RENAME COLUMN child_id TO profile_id;
-- +goose StatementEnd

-- +goose StatementBegin
-- profile_id is functionally NOT NULL (enforced in Go on every insert), but
-- SQLite ALTER TABLE ADD COLUMN cannot add a NOT NULL column without a DEFAULT
-- value, so we declare it nullable at the schema level and rely on the app
-- layer + the trackers.profile_id NOT NULL contract.
ALTER TABLE trackers ADD COLUMN profile_id INTEGER REFERENCES profiles(id) ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_trackers_profile ON trackers(profile_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE user_profile_access (
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
  granted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  granted_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  PRIMARY KEY (user_id, profile_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_upa_user ON user_profile_access(user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_upa_profile ON user_profile_access(profile_id);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO user_profile_access (user_id, profile_id)
SELECT u.id, p.id FROM users u CROSS JOIN profiles p WHERE u.role = 'user';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE settings SET value='Tracker', updated_at=CURRENT_TIMESTAMP
 WHERE key='app_name' AND value='Baby Tracker';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_profile_access;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_trackers_profile;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE trackers DROP COLUMN profile_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE entry_revisions RENAME COLUMN profile_id TO child_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_entries_profile_occurred;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE entries RENAME COLUMN profile_id TO child_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_entries_child_occurred ON entries(child_id, occurred_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE profiles DROP COLUMN description;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE profiles ADD COLUMN birthday TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE profiles RENAME TO children;
-- +goose StatementEnd
