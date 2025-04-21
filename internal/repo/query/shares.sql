-- name: InsertShare :one
INSERT INTO shares (shared_by, shared_for, file_id, expires_at, sharing_token)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (shared_by, shared_for, file_id) DO UPDATE
SET expires_at = EXCLUDED.expires_at
RETURNING *;

-- name: GetSharedFileIdFromToken :one
SELECT file_id FROM shares WHERE sharing_token = $1;

-- name: GetFilesSharedWithUser :many
SELECT
    f.*,
    s.*
FROM shares s
JOIN files f ON s.file_id = f.id
WHERE s.shared_for = $1;

-- name: GetBucketAndObjectFromToken :one
SELECT
u.user_bucket,
f.file_name
FROM shares s
JOIN files f ON s.file_id = f.id
JOIN users u ON f.owner_google_id = u.google_id
WHERE s.sharing_token = $1;

-- name: GetTokenExpirationTime :one
SELECT expires_at FROM shares WHERE sharing_token = $1;
