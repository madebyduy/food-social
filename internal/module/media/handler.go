package media

import (
	"net/http"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Sign: POST /api/v1/media/sign   (🔵 cần đăng nhập)
//
// Trả tham số + chữ ký để client PUT ảnh THẲNG lên Cloudinary. Không nhận file ở đây.
func (h *Handler) Sign(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, apperr.Unauthorized("Bạn cần đăng nhập để thực hiện hành động này"))
		return
	}

	asset, params, err := h.svc.Sign(r.Context(), ownerID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	httpx.Created(w, toSignResponse(asset.ID, params))
}

// Confirm: POST /api/v1/media/{id}/confirm   (🔵 cần đăng nhập; chỉ chủ ảnh)
//
// Sau khi client upload xong, gọi endpoint này để server xác minh với Cloudinary và
// đánh dấu ảnh USABLE (mới gắn được vào bài).
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, apperr.Unauthorized("Bạn cần đăng nhập để thực hiện hành động này"))
		return
	}

	id, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}

	asset, err := h.svc.Confirm(r.Context(), ownerID, id)
	if err != nil {
		httpx.Error(w, err) // Forbidden -> 403, NotFound -> 404, BadRequest -> 400
		return
	}

	httpx.OK(w, toAssetResponse(asset, h.svc.AssetURL(asset)))
}
