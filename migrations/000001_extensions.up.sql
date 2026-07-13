-- Bật các extension Postgres dùng xuyên suốt dự án.
-- unaccent: bỏ dấu tiếng Việt cho full-text search ("pho" khớp "phở").
-- pg_trgm : so khớp gần đúng (similarity) để gợi ý place trùng khi admin gộp.
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- unaccent() do extension cung cấp bị đánh dấu STABLE (không IMMUTABLE), nên KHÔNG dùng được
-- trực tiếp trong cột generated STORED (Postgres báo "generation expression is not immutable").
-- Bọc lại bằng một hàm IMMUTABLE — an toàn vì từ điển 'unaccent' cố định, không đổi theo phiên.
-- Các bảng places/posts dùng immutable_unaccent(...) trong search_vector.
CREATE OR REPLACE FUNCTION immutable_unaccent(text)
RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$ SELECT unaccent('unaccent', $1) $$;
