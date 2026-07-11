// Package httpx chứa các tiện ích HTTP dùng CHUNG cho mọi handler:
// envelope response nhất quán, ghi JSON, map lỗi -> status, decode body có giới hạn,
// cursor pagination, và helper đọc path param.
//
// Mục tiêu: handler ở mọi module chỉ cần gọi httpx.OK / httpx.Error... nên toàn bộ
// API có CÙNG một shape JSON. Client (mobile) chỉ phải parse một kiểu duy nhất.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Envelope là "vỏ bọc" JSON duy nhất cho MỌI response.
//
//	Thành công single:  { "data": {...} }
//	Thành công list:    { "data": [...], "meta": { "next_cursor": "...", "count": 20 } }
//	Lỗi:                { "error": { "code": "NOT_FOUND", "message": "..." } }
//
// Nhờ omitempty, field nào nil sẽ bị lược khỏi JSON.
type Envelope struct {
	Data  any       `json:"data,omitempty"`
	Meta  *Meta     `json:"meta,omitempty"`
	Error *apiError `json:"error,omitempty"`
}

// Meta mang thông tin phân trang cho list endpoint.
type Meta struct {
	NextCursor string `json:"next_cursor,omitempty"` // rỗng = hết trang
	Count      int    `json:"count"`
}

// apiError là phần "error" trong envelope. Để chữ thường (unexported) vì client
// không cần import kiểu này — chỉ đọc JSON.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON là hàm ghi cấp thấp: set header, status, encode Envelope.
// Các helper OK/Created/List/Error bên dưới đều gọi về đây.
func WriteJSON(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(env); err != nil {
		// Body có thể đã ghi dở, không thể sửa status nữa — chỉ log để còn biết.
		slog.Error("encode JSON response failed", "error", err)
	}
}

// OK trả 200 với một object đơn.
func OK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, Envelope{Data: data})
}

// Created trả 201 với tài nguyên vừa tạo.
func Created(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusCreated, Envelope{Data: data})
}

// NoContent trả 204 (không body) — dùng cho DELETE thành công chẳng hạn.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// List trả 200 với mảng dữ liệu + meta phân trang.
// nextCursor rỗng nghĩa là đã hết trang.
func List(w http.ResponseWriter, data any, count int, nextCursor string) {
	WriteJSON(w, http.StatusOK, Envelope{
		Data: data,
		Meta: &Meta{NextCursor: nextCursor, Count: count},
	})
}
