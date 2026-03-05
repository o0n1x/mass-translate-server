-- name: UpdateDocument :one
UPDATE documents
SET status = $2 , updated_at = NOW()
WHERE id = $1
RETURNING *;