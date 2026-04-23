-- +goose Up
ALTER TABLE documents
ADD COLUMN err_message TEXT DEFAULT null;

-- +goose Down
ALTER TABLE documents
DROP COLUMN err_message;