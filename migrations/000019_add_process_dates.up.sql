-- Плановый интервал процесса (для отображения на календаре многодневной полосой,
-- как проект). Append-only + IF NOT EXISTS: применяется и на «старых» БД при
-- рестарте, и на свежих (как no-op).
ALTER TABLE processes ADD COLUMN IF NOT EXISTS starts_at DATE;
ALTER TABLE processes ADD COLUMN IF NOT EXISTS ends_at   DATE;
