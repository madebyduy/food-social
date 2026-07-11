package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/madebyduy/food-social/internal/apperr"
)

// kindToStatus là NƠI DUY NHẤT map Kind (nghiệp vụ) -> HTTP status.
// Muốn đổi cách trả status cho một loại lỗi, chỉ sửa ở đây.
var kindToStatus = map[apperr.Kind]int{
	apperr.KindBadRequest:   http.StatusBadRequest,
	apperr.KindUnauthorized: http.StatusUnauthorized,
	apperr.KindForbidden:    http.StatusForbidden,
	apperr.KindNotFound:     http.StatusNotFound,
	apperr.KindConflict:     http.StatusConflict,
	apperr.KindTooMany:      http.StatusTooManyRequests,
	apperr.KindInternal:     http.StatusInternalServerError,
}

// Error là hàm xử lý lỗi tập trung mà MỌI handler gọi.
//
// Cách hoạt động:
//  1. Dùng errors.As để bóc *apperr.AppError ra khỏi chuỗi lỗi (%w) — kể cả khi
//     lỗi đã bị wrap nhiều lớp.
//  2. Nếu không phải AppError (lỗi lạ, ngoài dự tính) -> coi là Internal 500 và
//     KHÔNG để lộ chi tiết ra client.
//  3. Lỗi 500 thì log kèm lỗi gốc để dev điều tra; lỗi nghiệp vụ (4xx) không cần log ồn.
func Error(w http.ResponseWriter, err error) {
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		appErr = apperr.Internal(err)
	}

	status, ok := kindToStatus[appErr.Kind]
	if !ok {
		status = http.StatusInternalServerError
	}

	if appErr.Kind == apperr.KindInternal {
		// Chỉ log chi tiết ở tầng này, client chỉ thấy "lỗi hệ thống".
		slog.Error("internal error", "error", appErr.Err, "message", appErr.Message)
	}

	WriteJSON(w, status, Envelope{
		Error: &apiError{
			Code:    appErr.Kind.String(),
			Message: appErr.Message,
		},
	})
}
