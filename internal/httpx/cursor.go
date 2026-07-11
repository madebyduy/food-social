package httpx

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/madebyduy/food-social/internal/apperr"
)

// Cursor dùng cho phân trang kiểu keyset (không dùng OFFSET).
//
// Ý tưởng: thay vì "bỏ qua N dòng" (OFFSET — chậm ở trang sâu và bị nhảy dòng khi có
// bản ghi mới chèn vào), ta nhớ (created_at, id) của bản ghi CUỐI trang trước, rồi
// query WHERE (created_at, id) < (cursor) ORDER BY created_at DESC, id DESC.
//
// Cặp (created_at, id) đảm bảo thứ tự tuyệt đối kể cả khi nhiều bản ghi cùng created_at.
type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

// EncodeCursor biến Cursor thành chuỗi base64 "mờ" (opaque) để trả cho client.
// Client không cần hiểu bên trong — chỉ gửi lại nguyên văn ở lần gọi sau.
func EncodeCursor(c Cursor) string {
	raw := fmt.Sprintf("%d|%d", c.CreatedAt.UnixNano(), c.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parse ngược chuỗi cursor. Cursor hỏng -> lỗi 400 (do client gửi sai).
func DecodeCursor(s string) (Cursor, error) {
	rawBytes, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, apperr.BadRequest("cursor không hợp lệ")
	}

	var nanos, id int64
	if _, err := fmt.Sscanf(string(rawBytes), "%d|%d", &nanos, &id); err != nil {
		return Cursor{}, apperr.BadRequest("cursor không hợp lệ")
	}

	return Cursor{
		CreatedAt: time.Unix(0, nanos).UTC(),
		ID:        id,
	}, nil
}

// --- Tham số phân trang từ query string ---

const (
	defaultLimit = 20
	maxLimit     = 50
)

// ParseLimit đọc ?limit=... với mặc định 20, tối đa 50. limit sai/âm -> dùng mặc định.
func ParseLimit(raw string) int {
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}
