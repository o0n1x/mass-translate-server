-- name: GetDocument :one
SELECT *
FROM documents
WHERE id=$1;