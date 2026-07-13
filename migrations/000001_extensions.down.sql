-- Gỡ extension (chỉ chạy khi rollback toàn bộ). Thường để lại cũng vô hại.
DROP FUNCTION IF EXISTS immutable_unaccent(text);
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS unaccent;
