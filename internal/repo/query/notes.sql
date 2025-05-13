-- name: UpdateNoteForFile :one
INSERT INTO notes (user_id, file_id, content)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, file_id) 
DO UPDATE SET content = EXCLUDED.content
RETURNING *;

-- name: GetNoteForFileById :one
SELECT * FROM notes WHERE user_id = $1 AND file_id = $2;
