-- name: CreateLog :one
INSERT INTO logs (id, created_at, updated_at, is_successful,cached,error,request_id)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    $3,
    $4
)
RETURNING *;