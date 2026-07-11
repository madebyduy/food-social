-- Bảng users: tài khoản người dùng.
-- Quy ước: TIMESTAMPTZ (lưu UTC), soft delete qua deleted_at, enum bằng VARCHAR + CHECK.
CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    username        VARCHAR(50)  NOT NULL,
    email           VARCHAR(255) NOT NULL,
    phone           VARCHAR(20),
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(100) NOT NULL,
    avatar_url      TEXT,
    bio             TEXT,
    role            VARCHAR(20)  NOT NULL DEFAULT 'USER',
    status          VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    follower_count  INTEGER      NOT NULL DEFAULT 0, -- denormalized: đọc nhanh trang profile
    following_count INTEGER      NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    CHECK (role   IN ('USER','ADMIN')),
    CHECK (status IN ('ACTIVE','SUSPENDED','BANNED','DELETED'))
);

-- Unique CASE-INSENSITIVE: 'Duy' và 'duy' là một. Chỉ tính các user chưa xóa mềm.
CREATE UNIQUE INDEX ux_users_username_lower ON users (lower(username)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX ux_users_email_lower    ON users (lower(email))    WHERE deleted_at IS NULL;
