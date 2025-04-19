-- name: InsertShare :one
INSERT INTO shares (shared_by, shared_for, file_id, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (shared_by, shared_for, file_id) DO UPDATE
SET expires_at = EXCLUDED.expires_at
RETURNING *;


-- name: GetFilesSharedWithUser :many
SELECT f.*
FROM shares s
JOIN files f ON s.file_id = f.id
WHERE s.shared_for = $1;
