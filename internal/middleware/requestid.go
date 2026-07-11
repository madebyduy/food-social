package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// requestIDKey là key riêng để lưu request ID trong context.
// Dùng kiểu contextKey riêng (khai báo ở auth.go) để tránh đụng key của package khác.
const requestIDKey contextKey = "request_id"

// requestIDHeader là tên header trả về cho client (giúp client báo lỗi kèm ID để trace).
const requestIDHeader = "X-Request-ID"

// RequestID gắn cho mỗi request một ID ngẫu nhiên duy nhất.
//
// Nếu client đã gửi sẵn X-Request-ID (vd: từ gateway) thì tôn trọng giá trị đó;
// nếu không thì tự sinh. ID được đưa vào context (để Logging đọc) và trả trong header.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(requestIDHeader)
			if id == "" {
				id = newRequestID()
			}

			w.Header().Set(requestIDHeader, id)
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestIDFrom lấy request ID ra khỏi context (rỗng nếu chưa gắn).
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// newRequestID sinh 16 byte ngẫu nhiên -> chuỗi hex 32 ký tự.
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand gần như không bao giờ lỗi; nếu có thì trả chuỗi rỗng cũng không sao.
		return ""
	}
	return hex.EncodeToString(b)
}
