package post

import (
	"net/http"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/middleware"
)

// Handler là tầng HTTP của module post: đọc request -> gọi Service -> trả JSON.
// KHÔNG chứa nghiệp vụ, KHÔNG viết SQL.
type Handler struct {
	svc *Service
}

// NewHandler nhận *Service (được wire ở main.go).
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create xử lý: POST /api/v1/posts   (🔵 cần đăng nhập)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	authorID, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, apperr.Unauthorized("Bạn cần đăng nhập để thực hiện hành động này"))
		return
	}

	var req CreatePostRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.Error(w, err)
		return
	}

	p, err := h.svc.Create(r.Context(), authorID, req)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.Created(w, toPostResponse(p))
}

// GetByID xử lý: GET /api/v1/posts/{id}   (🟢 công khai)
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}

	p, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, err) // apperr.ErrNotFound -> 404
		return
	}

	httpx.OK(w, toPostResponse(p))
}

// Feed xử lý: GET /api/v1/posts?cursor=&limit=   (🟢 công khai, cursor pagination)
func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	limit := httpx.ParseLimit(r.URL.Query().Get("limit"))
	cursor, err := parseCursor(r)
	if err != nil {
		httpx.Error(w, err) // cursor hỏng -> 400
		return
	}

	posts, hasMore, err := h.svc.Feed(r.Context(), cursor, limit)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.List(w, toPostResponses(posts), len(posts), nextCursor(posts, hasMore))
}

// ListByUser xử lý: GET /api/v1/users/{id}/posts?cursor=&limit=   (🟢 công khai)
func (h *Handler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}

	limit := httpx.ParseLimit(r.URL.Query().Get("limit"))
	cursor, err := parseCursor(r)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	posts, hasMore, err := h.svc.ListByUser(r.Context(), userID, cursor, limit)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.List(w, toPostResponses(posts), len(posts), nextCursor(posts, hasMore))
}

// CountVisibleByUser xử lý: GET /api/v1/users/{id}/posts/count   (🟢 công khai)
//
// {id} là id của USER cần đếm (đối tượng bị xem) — lấy từ URL, KHÔNG phải người đăng nhập.
func (h *Handler) CountVisibleByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}

	n, err := h.svc.CountByUser(r.Context(), userID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.OK(w, map[string]int{"count": n})
}

// Update xử lý: PATCH /api/v1/posts/{id}   (🔵 chỉ tác giả; optimistic lock qua version)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, apperr.Unauthorized("Bạn cần đăng nhập để thực hiện hành động này"))
		return
	}

	id, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}

	var req UpdatePostRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.Error(w, err)
		return
	}

	p, err := h.svc.Update(r.Context(), actorID, id, req)
	if err != nil {
		httpx.Error(w, err) // Forbidden -> 403, NotFound -> 404, Conflict -> 409
		return
	}

	httpx.OK(w, toPostResponse(p))
}

// Delete xử lý: DELETE /api/v1/posts/{id}   (🔵 chỉ tác giả; xóa MỀM)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, apperr.Unauthorized("Bạn cần đăng nhập để thực hiện hành động này"))
		return
	}

	id, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}

	if err := h.svc.Delete(r.Context(), actorID, id); err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.NoContent(w) // 204
}

// --- Helper phân trang (tầng transport) ---

// parseCursor đọc ?cursor= từ query. Rỗng -> nil (trang đầu). Hỏng -> lỗi 400.
func parseCursor(r *http.Request) (*httpx.Cursor, error) {
	raw := r.URL.Query().Get("cursor")
	if raw == "" {
		return nil, nil
	}
	c, err := httpx.DecodeCursor(raw)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// nextCursor tạo cursor cho trang sau từ bản ghi CUỐI trang hiện tại. Hết trang -> rỗng.
func nextCursor(items []*PostWithRelations, hasMore bool) string {
	if !hasMore || len(items) == 0 {
		return ""
	}
	last := items[len(items)-1].Post
	return httpx.EncodeCursor(httpx.Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
}
