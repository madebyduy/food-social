package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder bọc http.ResponseWriter để "nghe lén" status code đã ghi.
// Mặc định net/http không cho đọc lại status sau khi WriteHeader, nên ta tự nhớ.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *statusRecorder) Write(b []byte) (int, error) {
	// Nếu handler gọi Write mà chưa gọi WriteHeader thì mặc định là 200.
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
	n, err := rec.ResponseWriter.Write(b)
	rec.bytes += n
	return n, err
}

// Logging ghi một dòng log cho mỗi request: method, path, status, thời gian xử lý,
// và request_id để nối với các log khác cùng request.
func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFrom(r.Context()),
			)
		})
	}
}
