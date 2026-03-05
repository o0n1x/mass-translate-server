-- +goose Up
CREATE TABLE documents (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    external_id TEXT NOT NULL,
    external_key TEXT,
    filename TEXT,
    status TEXT NOT NULL,
    request_id UUID NOT NULL,

    FOREIGN KEY(request_id) REFERENCES requests(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE documents;