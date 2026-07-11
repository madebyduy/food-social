package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
)

// Recovery là lớp NGOÀI CÙNG: bắt mọi panic xảy ra ở tầng sâu hơn để server không sập
// và client luôn nhận được một response 500 gọn gàng (thay vì bị ngắt kết nối).
//
// panic chỉ nên xảy ra do lỗi lập trình (nil pointer, index out of range...). Ta log
// kèm stack trace để dev sửa, nhưng KHÔNG để lộ stack ra client.
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"error", rec,
						"path", r.URL.Path,
						"request_id", RequestIDFrom(r.Context()),
						"stack", string(debug.Stack()),
					)
					httpx.Error(w, apperr.Internal(nil))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
