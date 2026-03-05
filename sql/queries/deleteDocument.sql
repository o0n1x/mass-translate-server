-- name: DeleteDocument :exec
DELETE FROM  documents
WHERE id=$1;