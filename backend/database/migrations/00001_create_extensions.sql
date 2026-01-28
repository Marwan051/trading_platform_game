-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS timescaledb;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP EXTENSION timescaledb;
-- +goose StatementEnd