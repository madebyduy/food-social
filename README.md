# AnNgon / FoodSocial API

Backend Go cho mạng xã hội review món ăn địa phương tại Việt Nam.

## Chạy local

Yêu cầu: Go 1.22+, Docker và PostgreSQL 15+.

```bash
docker compose -f internal/deployments/docker-compose.yml up -d
copy .env.example .env       # PowerShell: Copy-Item .env.example .env
go test ./...
go run ./cmd/api
```

Sau khi cài `golang-migrate`, chạy migration bằng `make migrate-up`. Seed địa lý mẫu bằng `make seed` khi đã có `psql`.

Mặc định API dùng UTF-8, lưu thời gian UTC; client Việt Nam nên hiển thị theo `Asia/Ho_Chi_Minh`, tiền tệ dùng VND và số điện thoại chuẩn E.164 (`+84`). Dữ liệu địa lý được tách tỉnh/thành, quận/huyện và phường/xã để hỗ trợ tìm kiếm địa phương.

## Đã triển khai

- Auth register/login/logout, session token băm, bcrypt, rate limit và health/readiness.
- Hồ sơ user, CRUD bài viết, soft delete, optimistic locking, cursor pagination.
- Ảnh theo luồng presign/confirm, hashtag, địa lý Việt Nam và places.
- Bình luận tối đa 2 cấp, like/unlike, save/unsave, follow/unfollow; counter cập nhật trong transaction và thao tác lặp lại an toàn.

## Còn thiếu trước production

Theo tài liệu kỹ thuật v2, các mảng lớn tiếp theo là xác minh OTP email/SMS và reset mật khẩu, quản lý thiết bị/xóa tài khoản, notification/outbox, report + admin audit, vote độ tin cậy/auto-hide, search nâng cao, job dọn media và integration test với PostgreSQL. Cần bổ sung kiểm tra MIME/kích thước ảnh thật ở storage, chống spam nội dung và các văn bản pháp lý Việt Nam trước khi mở công khai.

## API interaction chính

- `GET/POST /api/v1/posts/{id}/comments`
- `POST/DELETE /api/v1/posts/{id}/like`
- `POST/DELETE /api/v1/posts/{id}/save`
- `POST/DELETE /api/v1/users/{id}/follow`

- `POST /api/v1/auth/password-reset/request`
- `POST /api/v1/auth/password-reset/confirm`
- `POST /api/v1/reports`
- `POST/DELETE /api/v1/posts/{id}/vote`
- `GET /api/v1/notifications`
- `POST /api/v1/notifications/{id}/read`
- `POST /api/v1/admin/posts/{id}/hide` và `/restore`
- `GET /api/v1/search?q=pho&type=all`

Các endpoint cần đăng nhập dùng `Authorization: Bearer <token>` và mọi response đều có envelope `{ "data": ... }`.

Luồng reset mật khẩu hiện đã lưu token băm, hết hạn sau 30 phút và thu hồi toàn bộ session sau khi đổi mật khẩu. Cần gắn email/SMS provider trước khi mở công khai; API không trả token reset và không tiết lộ email có tồn tại hay không.
