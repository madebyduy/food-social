// Package apperr định nghĩa MỘT kiểu lỗi ứng dụng duy nhất cho toàn hệ thống.
//
// Ý tưởng cốt lõi (rất quan trọng cho người mới):
//   - Repository (tầng SQL) trả về lỗi kỹ thuật (vd: sql.ErrNoRows) hoặc sentinel
//     như ErrNotFound.
//   - Service (tầng nghiệp vụ) diễn giải lỗi đó thành *AppError mang sẵn "muốn trả
//     HTTP status nào" (Kind) và "thông điệp an toàn để hiện cho user" (Message).
//   - Handler (tầng HTTP) KHÔNG tự map lỗi — chỉ gọi httpx.Error(w, err), nơi đó
//     đọc Kind để chọn status code.
//
// Nhờ đó chi tiết lỗi nội bộ (câu SQL, lỗi driver...) không bao giờ rò ra client.
package apperr

import "fmt"

// Kind phân loại lỗi theo NGHIỆP VỤ, độc lập với HTTP.
// Việc map Kind -> HTTP status nằm ở tầng httpx (xem internal/httpx/errors.go),
// để package apperr không phụ thuộc net/http.
type Kind int

const (
	KindInternal     Kind = iota // lỗi hệ thống ngoài dự tính -> 500
	KindBadRequest               // input sai cú pháp / validate fail -> 400
	KindUnauthorized             // thiếu/sai/hết hạn token -> 401
	KindForbidden                // đã đăng nhập nhưng không có quyền -> 403
	KindNotFound                 // tài nguyên không tồn tại -> 404
	KindConflict                 // trùng, version lệch, trạng thái không cho phép -> 409
	KindTooMany                  // vượt rate limit -> 429
)

// String trả về "code" ổn định để đưa vào JSON (client dựa vào code này, không dựa
// vào message vì message có thể đổi/đa ngôn ngữ).
func (k Kind) String() string {
	switch k {
	case KindBadRequest:
		return "BAD_REQUEST"
	case KindUnauthorized:
		return "UNAUTHORIZED"
	case KindForbidden:
		return "FORBIDDEN"
	case KindNotFound:
		return "NOT_FOUND"
	case KindConflict:
		return "CONFLICT"
	case KindTooMany:
		return "TOO_MANY_REQUESTS"
	default:
		return "INTERNAL"
	}
}

// AppError là kiểu lỗi ứng dụng duy nhất được truyền qua các tầng.
//
// Nó thỏa interface error nên có thể dùng với fmt.Errorf("...: %w", appErr),
// errors.Is, errors.As như mọi lỗi Go bình thường.
type AppError struct {
	Kind    Kind   // quyết định HTTP status
	Message string // AN TOÀN để hiển thị cho user
	Err     error  // lỗi gốc — chỉ dùng để LOG, KHÔNG trả về client
}

// Error để *AppError thỏa interface error.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap cho phép errors.Is / errors.As bóc tới lỗi gốc bên trong.
func (e *AppError) Unwrap() error { return e.Err }

// --- Constructor tiện dụng: dùng ở tầng service ---

func BadRequest(msg string) *AppError {
	return &AppError{Kind: KindBadRequest, Message: msg}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Kind: KindUnauthorized, Message: msg}
}

func Forbidden(msg string) *AppError {
	return &AppError{Kind: KindForbidden, Message: msg}
}

func NotFound(msg string) *AppError {
	return &AppError{Kind: KindNotFound, Message: msg}
}

func Conflict(msg string) *AppError {
	return &AppError{Kind: KindConflict, Message: msg}
}

func TooMany(msg string) *AppError {
	return &AppError{Kind: KindTooMany, Message: msg}
}

// Internal bọc một lỗi hạ tầng bất kỳ thành lỗi 500 với message chung chung,
// đồng thời GIỮ lỗi gốc trong Err để còn log được.
func Internal(err error) *AppError {
	return &AppError{Kind: KindInternal, Message: "lỗi hệ thống", Err: err}
}

// --- Sentinel dùng ở tầng repository ---
//
// Repository trả ErrNotFound khi query không có dòng nào (sql.ErrNoRows).
// Service có thể errors.Is(err, apperr.ErrNotFound) để xử lý, hoặc trả thẳng.
var ErrNotFound = NotFound("không tìm thấy")
