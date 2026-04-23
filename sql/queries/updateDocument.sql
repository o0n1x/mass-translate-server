-- name: UpdateDocument :one
UPDATE documents
SET status = $2 ,err_message = $3 , updated_at = NOW()
WHERE id = $1
RETURNING *;