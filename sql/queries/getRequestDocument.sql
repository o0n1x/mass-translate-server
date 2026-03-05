-- name: GetDocumentProvider :one
SELECT requests.id,provider
FROM requests
JOIN documents ON documents.request_id = requests.id
WHERE documents.id = $1;