-- name: CreateRequest :one
INSERT INTO requests (id, created_at, updated_at, provider,req_type,from_lang,to_lang,user_id)
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