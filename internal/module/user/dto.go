package user

import (
	"database/sql"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/madebyduy/food-social/internal/apperr"
)

// dto.go chứa các DTO (Data Transfer Object) — struct dùng để GIAO TIẾP với client.
//
//   - Request DTO: hình dạng JSON client GỬI LÊN (đầu vào của handler).
//   - Response DTO: hình dạng JSON server TRẢ VỀ (đầu ra của handler).
//
// Tách DTO khỏi entity (model.go) để: (1) không lộ password_hash; (2) đổi shape API
// mà không phải đụng schema DB, và ngược lại.

// ProfileResponse là hồ sơ CÔNG KHAI của một user (ai cũng xem được).
// Cố ý KHÔNG có email/phone/role/status — đó là thông tin riêng tư/nội bộ.
type ProfileResponse struct {
	ID             int64     `json:"id"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"display_name"`
	AvatarURL      *string   `json:"avatar_url"` // null nếu chưa đặt
	Bio            *string   `json:"bio"`        // null nếu chưa đặt
	FollowerCount  int       `json:"follower_count"`
	FollowingCount int       `json:"following_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// toProfileResponse map entity User -> ProfileResponse (ẩn các field nhạy cảm).
func toProfileResponse(u *User) ProfileResponse {
	return ProfileResponse{
		ID:             u.ID,
		Username:       u.Username,
		DisplayName:    u.DisplayName,
		AvatarURL:      nullStringToPtr(u.AvatarURL),
		Bio:            nullStringToPtr(u.Bio),
		FollowerCount:  u.FollowerCount,
		FollowingCount: u.FollowingCount,
		CreatedAt:      u.CreatedAt,
	}
}

// UpdateProfileRequest là body của PATCH /users/{id}.
//
// Dùng CON TRỎ cho từng field để phân biệt 3 trạng thái (semantic của PATCH):
//   - field = nil        -> client KHÔNG gửi field này  -> giữ nguyên giá trị cũ.
//   - field = &"..."     -> client muốn ĐẶT giá trị mới.
//   - field = &""        -> client muốn XÓA (đặt về rỗng), vd xóa bio.
//
// Cố ý CHỈ cho sửa 3 field này. role/status/email... không nằm ở đây -> client không
// thể đổi (đó là việc của admin / luồng khác).
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	Bio         *string `json:"bio"`
	AvatarURL   *string `json:"avatar_url"`
}

// Validate kiểm tra CÚ PHÁP đầu vào (độ dài...). Trả apperr.BadRequest nếu sai.
// Lưu ý: đây là validate ở tầng nghiệp vụ nhẹ; ràng buộc thật sự vẫn có ở DB.
func (r UpdateProfileRequest) Validate() error {
	if r.DisplayName != nil {
		name := strings.TrimSpace(*r.DisplayName)
		if name == "" {
			return apperr.BadRequest("display_name không được để trống")
		}
		if utf8.RuneCountInString(name) > 100 {
			return apperr.BadRequest("display_name tối đa 100 ký tự")
		}
	}
	if r.Bio != nil && utf8.RuneCountInString(*r.Bio) > 500 {
		return apperr.BadRequest("bio tối đa 500 ký tự")
	}
	if r.AvatarURL != nil && len(*r.AvatarURL) > 1000 {
		return apperr.BadRequest("avatar_url quá dài")
	}
	return nil
}

// applyTo áp các thay đổi từ request lên entity đã tải từ DB.
// Chỉ field nào != nil mới được ghi đè (đúng semantic PATCH ở trên).
func (r UpdateProfileRequest) applyTo(u *User) {
	if r.DisplayName != nil {
		u.DisplayName = strings.TrimSpace(*r.DisplayName)
	}
	if r.Bio != nil {
		u.Bio = ptrToNullString(r.Bio)
	}
	if r.AvatarURL != nil {
		u.AvatarURL = ptrToNullString(r.AvatarURL)
	}
}

// --- Helper chuyển đổi giữa sql.NullString (DB) và *string (JSON) ---

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func ptrToNullString(p *string) sql.NullString {
	if p == nil || *p == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *p, Valid: true}
}
