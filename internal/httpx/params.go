package httpx

import (
	"net/http"
	"strconv"

	"github.com/madebyduy/food-social/internal/apperr"
)

// PathInt64 đọc một path param dạng số nguyên (vd: "id" trong "/users/{id}").
//
// Go 1.22+ hỗ trợ pattern routing sẵn trong net/http, lấy giá trị qua r.PathValue.
// Nếu param không phải số hợp lệ -> trả apperr.BadRequest (400).
func PathInt64(r *http.Request, name string) (int64, error) {
	raw := r.PathValue(name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.BadRequest(name + " không hợp lệ")
	}
	return id, nil
}
