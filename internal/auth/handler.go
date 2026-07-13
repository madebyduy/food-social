package auth

import (
	"net/http"
	"strings"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/middleware"
)

type Handler struct {
	svc *Service
}

type passwordResetRequest struct {
	Email string `json:"email"`
}
type passwordResetConfirm struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.RequestPasswordReset(r.Context(), req.Email); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, httpx.Envelope{Data: map[string]string{"message": "nếu email tồn tại, hướng dẫn đặt lại mật khẩu sẽ được gửi"}})
}
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirm
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, apperr.Unauthorized("cần đăng nhập"))
		return
	}
	if err := h.svc.DeleteAccount(r.Context(), id); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}

	res, err := h.svc.Register(r.Context(), req)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}

	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		httpx.Error(w, apperr.Unauthorized("thiếu session token"))
		return
	}

	if err := h.svc.Logout(r.Context(), token); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Error(w, apperr.Unauthorized("cần đăng nhập"))
		return
	}

	res, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, res)
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
