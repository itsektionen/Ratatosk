-- +goose Up
CREATE TABLE users (
	id         INTEGER PRIMARY KEY,
	name       TEXT NOT NULL,
	role       TEXT NOT NULL,
	oidc_sub   TEXT UNIQUE,
	invited_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
	created_at INTEGER NOT NULL
);
CREATE TABLE sessions (
	token_hash BLOB PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	expires_at INTEGER NOT NULL
);
CREATE TABLE invites (
	token_hash BLOB PRIMARY KEY,
	user_id    INTEGER REFERENCES users(id) ON DELETE CASCADE,
	created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
	expires_at INTEGER NOT NULL
);
CREATE TABLE sites (
	name       TEXT PRIMARY KEY,
	owner_id   INTEGER REFERENCES users(id) ON DELETE SET NULL,
	created_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE sites;
DROP TABLE invites;
DROP TABLE sessions;
DROP TABLE users;
