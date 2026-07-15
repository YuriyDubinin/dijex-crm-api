DROP TRIGGER IF EXISTS trg_revoke_tokens_on_user_disable ON customer_users;
DROP FUNCTION IF EXISTS trigger_revoke_tokens_on_user_disable();
DROP TABLE IF EXISTS auth_tokens;
DROP TYPE IF EXISTS auth_device_type;
DROP TYPE IF EXISTS auth_token_type;
