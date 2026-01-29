-- +goose Up
CREATE TABLE requests (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    provider TEXT NOT NULL,
    req_type TEXT NOT NULL,
    from_lang TEXT NOT NULL,
    to_lang TEXT NOT NULL,
    user_id UUID NOT NULL,
    
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE requests;