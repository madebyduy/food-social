package user

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/madebyduy/food-social/internal/apperr"
)

// Service chứa NGHIỆP VỤ của module user. Nó KHÔNG biết gì về HTTP (không đụng
// http.Request/ResponseWriter, không biết JSON) và KHÔNG viết SQL.
//
// Service giữ:
//   - db   *sql.DB   : để mở transaction khi cần (module user hiện chưa cần, nhưng
//     cũng dùng db như một Querier để truyền cho repo ở thao tác đơn).
//   - repo Repository: interface -> test được bằng mock.
//   - log            : ghi log nghiệp vụ khi cần.
type Service struct {
	db   *sql.DB
	repo Repository
	log  *slog.Logger
}

// NewService trả về *Service (struct cụ thể) — "return structs".
func NewService(db *sql.DB, repo Repository, log *slog.Logger) *Service {
	return &Service{db: db, repo: repo, log: log}
}

// GetProfile lấy hồ sơ công khai của một user theo id.
// Trả về entity *User; việc "giấu" field nhạy cảm là do Handler map sang DTO.
func (s *Service) GetProfile(ctx context.Context, id int64) (*User, error) {
	// Thao tác đọc đơn -> truyền thẳng s.db (là một Querier), không cần transaction.
	return s.repo.GetByID(ctx, s.db, id)
}

// UpdateProfile sửa hồ sơ của user.
//
// Tham số:
//   - actorID  : user ĐANG ĐĂNG NHẬP (lấy từ session/context, KHÔNG từ body).
//   - targetID : user bị sửa (lấy từ URL /users/{id}).
//
// Quy tắc nghiệp vụ cốt lõi: chỉ được sửa hồ sơ CỦA CHÍNH MÌNH.
func (s *Service) UpdateProfile(
	ctx context.Context,
	actorID, targetID int64,
	req UpdateProfileRequest,
) (*User, error) {
	// Phân quyền ở tầng SERVICE, không tin client: dù URL là ai thì cũng phải trùng
	// người đang đăng nhập.
	if actorID != targetID {
		return nil, apperr.Forbidden("bạn chỉ có thể sửa hồ sơ của chính mình")
	}

	// Đây là thao tác 1 bảng, 2 câu lệnh (đọc rồi ghi). Với sửa hồ sơ, "last write wins"
	// là chấp nhận được nên không cần transaction/optimistic lock (khác với sửa post).
	u, err := s.repo.GetByID(ctx, s.db, targetID)
	if err != nil {
		return nil, err // đã là apperr.ErrNotFound (404) nếu không tồn tại
	}

	req.applyTo(u)

	if err := s.repo.UpdateProfile(ctx, s.db, u); err != nil {
		return nil, err
	}

	return u, nil
}
