package post

import (
	"encoding/json"
	"net/http"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/middleware"
)

type Handler struct {
	// svc *Service
}

func (h *Handler) Create(w http.ResponseWriter,r *http.Request){
	var req CreatePost
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "body không hợp lệ", http.StatusBadRequest)
		return
	}
 current_user,ok := middleware.UserID(r.Context())
 if !ok {
	httpx.Error(w, apperr.Unauthorized("Bạn cần đăng nhập để thực hiện hành động này"))
	return
 }

 


}