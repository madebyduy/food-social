-- Bảng password_reset_tokens: token đặt lại mật khẩu (xem Phần 00b — Bảo mật).
-- Cùng cơ chế với sessions: DB chỉ lưu sha256(token), token gốc chỉ trả cho client MỘT LẦN.
-- Sau khi reset-password thành công: set used_at, và revoke TOÀN BỘ session hiện có của user.
--
-- (Bảng này thuộc module auth, được tạo ở đây để giữ đúng thứ tự migration trong tài liệu
--  — posts sẽ là 000008. Bạn viết model/repository cho nó khi làm phần auth.)
CREATE TABLE password_reset_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    token_hash  VARCHAR(255) NOT NULL UNIQUE,   -- sha256(token), KHÔNG lưu token gốc
    expires_at  TIMESTAMPTZ NOT NULL,           -- hạn ngắn 15–30 phút
    used_at     TIMESTAMPTZ,                     -- NULL = chưa dùng; token chỉ dùng được 1 lần
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- FK luôn tự đánh index (Postgres không tự tạo index cho FK).
CREATE INDEX ix_password_reset_user ON password_reset_tokens(user_id);
