// Package middleware chứa các lớp "bọc" quanh handler, chạy TRƯỚC/SAU mỗi request.
//
// Thứ tự một request đi qua (từ ngoài vào trong):
//
//	Client
//	  -> Recovery      (bắt panic, đảm bảo luôn trả response)
//	  -> RequestID     (gắn ID để trace log)
//	  -> Logging       (đo thời gian, ghi log request)
//	  -> [Authenticate] (chỉ route cần đăng nhập: đọc token -> nhét userID vào context)
//	  -> Handler       (parse -> gọi Service -> map kết quả ra JSON)
//	  -> Service       (nghiệp vụ + transaction)
//	  -> Repository    (SQL)
//	  -> PostgreSQL
package middleware

import "net/http"

// Middleware là một hàm nhận handler và trả về handler đã được "bọc".
type Middleware func(http.Handler) http.Handler

// Chain gắn nhiều middleware quanh một handler.
//
// Middleware được áp theo thứ tự KHAI BÁO: cái liệt kê ĐẦU TIÊN nằm NGOÀI CÙNG
// (chạy trước hết khi vào, sau hết khi ra). Ví dụ:
//
//	Chain(mux, Recovery, RequestID, Logging)
//
// => request đi qua Recovery -> RequestID -> Logging -> mux.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	// Áp từ cuối lên đầu để cái đầu tiên thành lớp ngoài cùng.
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
