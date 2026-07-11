-- Bật các extension Postgres dùng xuyên suốt dự án.
-- unaccent: bỏ dấu tiếng Việt cho full-text search ("pho" khớp "phở").
-- pg_trgm : so khớp gần đúng (similarity) để gợi ý place trùng khi admin gộp.
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
