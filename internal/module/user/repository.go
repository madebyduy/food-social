package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
)

// Repository là INTERFACE mô tả các thao tác DB mà module user cần.
//
// Vì sao khai báo interface (thay vì chỉ struct)?
//   - Service phụ thuộc vào interface này -> khi test Service ta thay bằng mock
//     in-memory, không cần DB thật (xem service_test.go).
//   - "Accept interfaces, return structs": Service nhận Repository (interface),
//     còn NewRepository trả về struct cụ thể.
//
// Mọi method nhận database.Querier (không giữ *sql.DB bên trong) để Service quyết định
// chạy trong transaction hay không.
type Repository interface {
	// GetByID lấy 1 user chưa bị xóa mềm. Không thấy -> apperr.ErrNotFound.
	GetByID(ctx context.Context, q database.Querier, id int64) (*User, error)

	// UpdateProfile ghi 3 field hồ sơ (display_name, bio, avatar_url) của user.
	// Không thấy user (đã xóa) -> apperr.ErrNotFound.
	UpdateProfile(ctx context.Context, q database.Querier, u *User) error
}

// repository là implement cụ thể, viết chữ thường (private) — bên ngoài chỉ dùng qua
// interface Repository. Struct rỗng vì nó stateless (nhận Querier từ ngoài mỗi lần gọi).
type repository struct{}

// NewRepository trả về Repository. Tham số *sql.DB không được giữ lại (đánh dấu _)
// nhưng vẫn nhận để đồng bộ chữ ký với các module khác và dễ mở rộng sau này.
func NewRepository(_ *sql.DB) Repository {
	return &repository{}
}

func (r *repository) GetByID(ctx context.Context, q database.Querier, id int64) (*User, error) {
	const query = `
		SELECT
			id, username, email, phone, password_hash, display_name,
			avatar_url, bio, role, status,
			follower_count, following_count,
			created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	var u User
	err := q.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.Phone, &u.PasswordHash, &u.DisplayName,
		&u.AvatarURL, &u.Bio, &u.Role, &u.Status,
		&u.FollowerCount, &u.FollowingCount,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		// Phân biệt "không có dòng nào" (404) với lỗi DB thật (500).
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("user.GetByID scan: %w", err)
	}

	return &u, nil
}

func (r *repository) UpdateProfile(ctx context.Context, q database.Querier, u *User) error {
	const query = `
		UPDATE users
		SET display_name = $1,
		    bio          = $2,
		    avatar_url   = $3,
		    updated_at   = now()
		WHERE id = $4 AND deleted_at IS NULL`

	result, err := q.ExecContext(ctx, query,
		u.DisplayName, u.Bio, u.AvatarURL, u.ID,
	)
	if err != nil {
		return fmt.Errorf("user.UpdateProfile exec: %w", err)
	}

	// Không có dòng nào bị đổi -> user không tồn tại hoặc đã bị xóa.
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("user.UpdateProfile rows affected: %w", err)
	}
	if rows == 0 {
		return apperr.ErrNotFound
	}

	return nil
}
