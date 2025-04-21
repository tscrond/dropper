-- name: InsertFile :one
INSERT INTO files (owner_google_id, file_name, file_type, size, md5_checksum)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (owner_google_id, md5_checksum) DO NOTHING
RETURNING *;

-- name: GetFilesByOwner :many
SELECT * FROM files WHERE owner_google_id = $1;

-- name: GetFileByOwnerAndName :one
SELECT id, md5_checksum
FROM files
WHERE owner_google_id = $1 AND file_name = $2;

-- name: GetFileById :one
SELECT * FROM files WHERE id = $1;
