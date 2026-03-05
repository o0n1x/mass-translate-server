-- name: CreateDocument :one
INSERT INTO documents (id, created_at, updated_at, external_id,external_key,filename,status,request_id)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;