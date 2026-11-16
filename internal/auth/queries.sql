-- name: UpsertOIDCUser :one
INSERT INTO users (name, role, oidc_sub, created_at)
VALUES (sqlc.arg(name), sqlc.arg(role), sqlc.arg(oidc_sub), sqlc.arg(created_at))
ON CONFLICT (oidc_sub) DO UPDATE SET name = excluded.name, role = excluded.role
RETURNING id, name, role;

-- name: CreateGuest :one
INSERT INTO users (name, role, invited_by, created_at)
VALUES (sqlc.arg(name), sqlc.arg(role), sqlc.arg(invited_by), sqlc.arg(created_at))
RETURNING id, name, role;

-- name: GetUser :one
SELECT id, name, role FROM users WHERE id = sqlc.arg(id);

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at)
VALUES (sqlc.arg(token_hash), sqlc.arg(user_id), sqlc.arg(expires_at));

-- name: GetSessionUser :one
SELECT u.id, u.name, u.role
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = sqlc.arg(token_hash) AND s.expires_at > sqlc.arg(now);

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = sqlc.arg(token_hash);

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < sqlc.arg(now);

-- name: CreateInvite :exec
INSERT INTO invites (token_hash, user_id, created_by, expires_at)
VALUES (sqlc.arg(token_hash), sqlc.arg(user_id), sqlc.arg(created_by), sqlc.arg(expires_at));

-- name: GetInvite :one
SELECT user_id, created_by FROM invites
WHERE token_hash = sqlc.arg(token_hash) AND expires_at > sqlc.arg(now);

-- name: DeleteExpiredInvites :exec
DELETE FROM invites WHERE expires_at < sqlc.arg(now);
