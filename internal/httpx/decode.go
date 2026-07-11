package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/madebyduy/food-social/internal/apperr"
)

// maxBodyBytes giới hạn kích thước body JSON (1 MB). Chống client gửi body khổng lồ
// làm cạn RAM. Ảnh KHÔNG đi qua API này (upload thẳng lên storage), nên 1 MB là dư.
const maxBodyBytes = 1 << 20

// DecodeJSON đọc body request thành struct dst một cách AN TOÀN:
//   - Giới hạn kích thước body (chống payload quá lớn).
//   - Từ chối field lạ (DisallowUnknownFields) -> client gõ sai tên field sẽ báo lỗi
//     thay vì bị nuốt âm thầm.
//   - Chỉ cho đúng MỘT JSON object (chống rác phía sau).
//
// Mọi lỗi decode được trả về dưới dạng apperr.BadRequest (400), KHÔNG phải 500 —
// vì đây là lỗi do input của client.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}

	// Đảm bảo không còn JSON thứ hai phía sau (vd: `{...}{...}`).
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperr.BadRequest("body chỉ được chứa một JSON object")
	}

	return nil
}

// decodeError dịch các loại lỗi của encoding/json thành thông báo dễ hiểu.
func decodeError(err error) error {
	var (
		syntaxErr *json.SyntaxError
		typeErr   *json.UnmarshalTypeError
		maxErr    *http.MaxBytesError
	)

	switch {
	case errors.As(err, &syntaxErr):
		return apperr.BadRequest(fmt.Sprintf("JSON sai cú pháp tại vị trí %d", syntaxErr.Offset))
	case errors.As(err, &typeErr):
		return apperr.BadRequest(fmt.Sprintf("field %q sai kiểu dữ liệu", typeErr.Field))
	case errors.Is(err, io.EOF):
		return apperr.BadRequest("body không được rỗng")
	case errors.As(err, &maxErr):
		return apperr.BadRequest("body quá lớn")
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return apperr.BadRequest(fmt.Sprintf("field không hợp lệ: %s", field))
	default:
		return apperr.BadRequest("body không hợp lệ")
	}
}
