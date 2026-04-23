-- name: GetRequest :one
SELECT *
FROM requests
WHERE id=$1;