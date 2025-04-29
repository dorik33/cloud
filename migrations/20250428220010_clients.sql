-- +goose Up
-- +goose StatementBegin
CREATE TABLE clients (
    client_id VARCHAR(255) PRIMARY KEY,
    capacity INTEGER NOT NULL CHECK (capacity > 0),
    rate_per_sec INTEGER NOT NULL CHECK (rate_per_sec > 0),
    tokens INTEGER NOT NULL CHECK (tokens >= 0 AND tokens <= capacity),
    last_refill TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER update_clients_updated_at
BEFORE UPDATE ON clients
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_clients_updated_at ON clients; 
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION IF EXISTS update_updated_at;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS clients;
-- +goose StatementEnd