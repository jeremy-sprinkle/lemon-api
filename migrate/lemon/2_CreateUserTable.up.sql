CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE usertable (
    id VARCHAR(36) PRIMARY KEY UNIQUE,
    username BYTEA UNIQUE NOT NULL,
    hash BYTEA NOT NULL,
    save_state VARCHAR
);

CREATE INDEX user_index ON usertable (id);