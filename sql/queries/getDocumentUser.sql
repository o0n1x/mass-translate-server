-- name: GetDocumentUser :one
SELECT users.id
FROM users 
JOIN requests ON requests.user_id = users.id 
JOIN documents ON documents.request_id = requests.id
WHERE documents.id = $1;