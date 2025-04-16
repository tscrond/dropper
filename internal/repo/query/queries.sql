-- name: CreateUser :exec
INSERT INTO users (google_id, user_name, user_email)
VALUES ($1, $2, $3)
ON CONFLICT (google_id) DO NOTHING;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE user_email = $1;

-- name: InsertFile :one
INSERT INTO files (owner_google_id, file_name, file_type, size, md5_checksum)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetFilesByOwner :many
SELECT * FROM files WHERE owner_google_id = $1;

-- name: ShareFile :one
INSERT INTO shares (shared_by, shared_for, file_id, sharing_link)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetFilesSharedWithUser :many
SELECT f.*
FROM shares s
JOIN files f ON s.file_id = f.id
WHERE s.shared_for = $1;
