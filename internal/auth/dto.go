package auth

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/madebyduy/food-social/internal/apperr"
)

type RegisterRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

func (r RegisterRequest) Validate() error {
	if err := validateUsername(r.Username); err != nil {
		return err
	}
	if err := validateEmail(r.Email); err != nil {
		return err
	}
	if err := validatePassword(r.Password); err != nil {
		return err
	}
	if utf8.RuneCountInString(strings.TrimSpace(r.DisplayName)) > 100 {
		return apperr.BadRequest("display_name tối đa 100 ký tự")
	}
	return nil
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (r LoginRequest) Validate() error {
	if strings.TrimSpace(r.Login) == "" {
		return apperr.BadRequest("login không được để trống")
	}
	return validatePassword(r.Password)
}

type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

type UserResponse struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        Role      `json:"role"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func toUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt,
	}
}
