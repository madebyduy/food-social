package place

import (
	"net/http"
	"strconv"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create: POST /api/v1/places   (🔵 cần đăng nhập)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreatePlaceRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.Error(w, err)
		return
	}

	p, err := h.svc.Create(r.Context(), req)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, toPlaceResponse(p))
}

// GetByID: GET /api/v1/places/{id}   (🟢 công khai)
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}

	p, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, err) // ErrNotFound -> 404
		return
	}
	httpx.OK(w, toPlaceResponse(p))
}

// Search: GET /api/v1/places?q=&province_id=&limit=   (🟢 công khai)
//
// Dùng cho ô "chọn địa điểm" khi đăng bài: gõ tên -> gợi ý. q rỗng -> trả place phổ biến.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	provinceID, err := optionalQueryInt64(q.Get("province_id"))
	if err != nil {
		httpx.Error(w, err)
		return
	}

	limit := httpx.ParseLimit(q.Get("limit"))

	places, err := h.svc.Search(r.Context(), q.Get("q"), provinceID, limit)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.List(w, toPlaceResponses(places), len(places), "")
}

// optionalQueryInt64 đọc một query param số nguyên KHÔNG bắt buộc. Rỗng -> nil (không lọc).
func optionalQueryInt64(raw string) (*int64, error) {
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return nil, apperr.BadRequest("province_id không hợp lệ")
	}
	return &v, nil
}
