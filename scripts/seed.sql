-- Seed vài user mẫu để test module user ngay (GET/PATCH).
-- password_hash ở đây là chuỗi giả (chưa có luồng login) — Giai đoạn 1 (auth) sẽ tạo
-- user bằng bcrypt thật qua POST /auth/register.
INSERT INTO users (username, email, password_hash, display_name, bio)
VALUES
    ('duy',  'duy@example.com',  'x-not-a-real-hash', 'Duy Trần',  'Mê ăn ốc Hải Phòng'),
    ('lan',  'lan@example.com',  'x-not-a-real-hash', 'Lan Nguyễn', NULL)
ON CONFLICT DO NOTHING;
