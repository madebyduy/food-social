package user

import (
	"net/http"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/middleware"
)

// Handler là tầng HTTP: nó CHỈ biết HTTP.
// Nhiệm vụ: đọc path/query/body -> gọi Service -> map kết quả (hoặc lỗi) ra JSON.
// Handler KHÔNG chứa nghiệp vụ và KHÔNG viết SQL.
type Handler struct {
	svc *Service
}

// NewHandler nhận *Service (đã được wire ở main.go).
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetByID xử lý: GET /api/v1/users/{id}   (🟢 công khai — guest xem được)
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}

	u, err := h.svc.GetProfile(r.Context(), id)
	if err != nil {
		httpx.Error(w, err) // apperr.ErrNotFound -> 404, lỗi lạ -> 500
		return
	}

	// Map entity -> DTO công khai TRƯỚC khi trả (giấu email/password_hash...).
	httpx.OK(w, toProfileResponse(u))
}

// Update xử lý: PATCH /api/v1/users/{id}   (🔵 cần đăng nhập; chỉ sửa chính mình)
//
// Route này được bọc bởi middleware.Authenticate, nên userID đã có sẵn trong context.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	// actorID = người ĐANG ĐĂNG NHẬP, lấy từ CONTEXT (không bao giờ lấy từ body).
	actorID, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, apperr.Unauthorized("cần đăng nhập"))
		return
	}

	// targetID = user muốn sửa, lấy từ URL.
	targetID, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var req UpdateProfileRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}

	// Validate CÚ PHÁP ở handler (độ dài...). Kiểm tra QUYỀN nằm trong service.
	if err := req.Validate(); err != nil {
		httpx.Error(w, err)
		return
	}

	u, err := h.svc.UpdateProfile(r.Context(), actorID, targetID, req)
	if err != nil {
		httpx.Error(w, err) // Forbidden -> 403, NotFound -> 404...
		return
	}

	httpx.OK(w, toProfileResponse(u))
}
