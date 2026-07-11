package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
)

// contextKey là kiểu RIÊNG để làm key trong context. Dùng kiểu riêng (không phải
// string trần) để không đụng key của package/thư viện khác — đây là idiom Go chuẩn.
type contextKey string

const userIDKey contextKey = "user_id"

// SessionResolver là interface mà middleware Authenticate CẦN để đổi token -> userID.
//
// Middleware KHÔNG biết session được lưu ở đâu (DB, cache...) — nó chỉ cần một thứ
// biết cách "resolve". Ở Giai đoạn 1, module auth sẽ implement interface này bằng
// cách hash token rồi tra bảng sessions. Đây là "accept interfaces" — dễ thay/test.
type SessionResolver interface {
	// ResolveSession trả userID nếu token hợp lệ & còn hạn; ngược lại trả lỗi.
	ResolveSession(ctx context.Context, rawToken string) (int64, error)
}

// Authenticate là middleware cho các route CẦN ĐĂNG NHẬP.
//
// Nhiệm vụ: đọc "Authorization: Bearer <token>", resolve ra userID, rồi nhét userID
// vào context. Từ đó handler lấy userID QUA CONTEXT — tuyệt đối không lấy từ body/param
// (nguyên tắc số 1: không tin client).
func Authenticate(resolver SessionResolver) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r.Header.Get("Authorization"))
			if token == "" {
				httpx.Error(w, apperr.Unauthorized("thiếu session token"))
				return
			}

			userID, err := resolver.ResolveSession(r.Context(), token)
			if err != nil {
				httpx.Error(w, apperr.Unauthorized("phiên đăng nhập không hợp lệ"))
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID lấy userID mà Authenticate đã nhét vào context.
// ok = false nghĩa là request KHÔNG đi qua Authenticate (route công khai) hoặc chưa
// đăng nhập — handler dựa vào ok để phân biệt guest / user.
func UserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

// extractBearerToken cắt tiền tố "Bearer " khỏi header Authorization.
func extractBearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
