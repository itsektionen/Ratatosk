-- name: ClaimSite :execrows
INSERT INTO sites (name, owner_id, created_at)
VALUES (sqlc.arg(name), sqlc.arg(owner_id), sqlc.arg(created_at))
ON CONFLICT (name) DO NOTHING;

-- name: UnownSite :exec
UPDATE sites SET owner_id = NULL WHERE name = sqlc.arg(name);

-- name: DeleteSite :exec
DELETE FROM sites WHERE name = sqlc.arg(name);

-- name: AdoptSite :exec
INSERT OR IGNORE INTO sites (name, created_at)
VALUES (sqlc.arg(name), sqlc.arg(created_at));

-- name: GetSite :one
SELECT s.name,
       CAST(COALESCE(s.owner_id, 0) AS INTEGER) AS owner_id,
       CAST(COALESCE(u.name, '') AS TEXT) AS owner_name,
       CAST(COALESCE(u.role, '') AS TEXT) AS owner_role
FROM sites s LEFT JOIN users u ON u.id = s.owner_id
WHERE s.name = sqlc.arg(name);

-- name: ListSites :many
SELECT s.name,
       CAST(COALESCE(s.owner_id, 0) AS INTEGER) AS owner_id,
       CAST(COALESCE(u.name, '') AS TEXT) AS owner_name,
       CAST(COALESCE(u.role, '') AS TEXT) AS owner_role
FROM sites s LEFT JOIN users u ON u.id = s.owner_id
ORDER BY s.name;
