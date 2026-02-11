-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER user_profile_updated_at BEFORE
UPDATE ON user_profile FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER bots_updated_at BEFORE
UPDATE ON bots FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER stocks_updated_at BEFORE
UPDATE ON stocks FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER orders_updated_at BEFORE
UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER positions_updated_at BEFORE
UPDATE ON positions FOR EACH ROW EXECUTE FUNCTION update_updated_at();
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS user_profile_updated_at ON user_profile;
DROP TRIGGER IF EXISTS bots_updated_at ON bots;
DROP TRIGGER IF EXISTS stocks_updated_at ON stocks;
DROP TRIGGER IF EXISTS orders_updated_at ON orders;
DROP TRIGGER IF EXISTS positions_updated_at ON positions;
DROP FUNCTION IF EXISTS update_updated_at() CASCADE;
-- +goose StatementEnd