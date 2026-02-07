CREATE TABLE organization (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    singleton   BOOLEAN NOT NULL DEFAULT TRUE UNIQUE
);

CREATE TABLE users (
    id             SERIAL PRIMARY KEY,
    email          TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL,
    password_hash  TEXT NOT NULL,
    role           TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner','admin','member')),
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at  TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE connections (
     id          SERIAL PRIMARY KEY,
     name        TEXT NOT NULL UNIQUE,
     kind        TEXT NOT NULL CHECK (kind IN ('postgres','mysql','cql')),
     dsn         TEXT,
     is_enabled  BOOLEAN NOT NULL DEFAULT TRUE,
     user_id     INT NOT NULL REFERENCES users(id),
     created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE connection_access (
   user_id       INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
   connection_id INT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
   can_query     BOOLEAN NOT NULL DEFAULT TRUE,
   allow_writes  BOOLEAN NOT NULL DEFAULT FALSE,
   can_manage    BOOLEAN NOT NULL DEFAULT FALSE,
   granted_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (user_id, connection_id)
);