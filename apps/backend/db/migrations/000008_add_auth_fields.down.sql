DROP TABLE IF EXISTS refresh_tokens;

ALTER TABLE users
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS must_change_password;
