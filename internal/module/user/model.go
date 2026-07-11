package user

import (
	"database/sql"
	"time"
)

// model.go chứa ENTITY — ánh xạ 1-1 với bảng `users` trong DB.
//
// Quy ước quan trọng cho người mới:
//   - Entity (User) là thứ Repository đọc/ghi từ DB. Nó dùng các kiểu sql.Null* cho
//     cột cho phép NULL, và CHỨA cả password_hash.
//   - Entity KHÔNG bao giờ được trả thẳng ra JSON cho client. Handler luôn map entity
//     -> DTO (xem dto.go) để không lộ password_hash và để tách shape API khỏi DB.

// Role — vai trò tài khoản (khớp CHECK constraint trong migration).
type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

// Status — trạng thái tài khoản (khớp CHECK constraint trong migration).
type Status string

const (
	StatusActive    Status = "ACTIVE"
	StatusSuspended Status = "SUSPENDED"
	StatusBanned    Status = "BANNED"
	StatusDeleted   Status = "DELETED"
)

// User là entity ánh xạ đúng các cột của bảng `users`.
//
// Các cột NULL-able dùng sql.NullString / sql.NullTime để phân biệt "chưa có giá trị"
// với "chuỗi rỗng". Khi map ra DTO ta sẽ chuyển các Null* này thành *string (con trỏ).
type User struct {
	ID             int64
	Username       string
	Email          string
	Phone          sql.NullString
	PasswordHash   string // KHÔNG BAO GIỜ đưa vào DTO/JSON
	DisplayName    string
	AvatarURL      sql.NullString
	Bio            sql.NullString
	Role           Role
	Status         Status
	FollowerCount  int
	FollowingCount int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      sql.NullTime
}
