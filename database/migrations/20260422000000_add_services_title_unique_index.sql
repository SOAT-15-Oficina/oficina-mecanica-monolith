-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS idx_services_title_unique ON services (LOWER(title));
CREATE INDEX IF NOT EXISTS idx_services_active ON services (active);
CREATE INDEX IF NOT EXISTS idx_services_title_search ON services (title);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_services_title_search;
DROP INDEX IF EXISTS idx_services_active;
DROP INDEX IF EXISTS idx_services_title_unique;
-- +goose StatementEnd
