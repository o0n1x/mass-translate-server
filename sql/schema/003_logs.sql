-- +goose Up
CREATE TABLE logs (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    is_successful BOOLEAN NOT NULL,
    cached BOOLEAN NOT NULL,
    error TEXT,
    request_id UUID NOT NULL,
    
    FOREIGN KEY(request_id) REFERENCES requests(id)
    
    
);

-- +goose Down
DROP TABLE logs;