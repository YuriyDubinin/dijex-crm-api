DROP FUNCTION IF EXISTS trigger_set_updated_at();
-- Расширения pgcrypto/btree_gist намеренно НЕ удаляем: они безвредны и могут
-- использоваться другими объектами БД.
