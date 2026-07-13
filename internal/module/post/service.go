package post

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
	"github.com/madebyduy/food-social/internal/httpx"
)

// Service chứa NGHIỆP VỤ của module post. KHÔNG đụng http.*, KHÔNG viết SQL.
type Service struct {
	db   *sql.DB
	repo Repository
	log  *slog.Logger
}

// NewService trả về *Service (struct cụ thể) — "return structs".
func NewService(db *sql.DB, repo Repository, log *slog.Logger) *Service {
	return &Service{db: db, repo: repo, log: log}
}

// Create tạo một bài ĐẦY ĐỦ cho user đang đăng nhập: nội dung + ảnh + vị trí + hashtag.
//
// Vì thao tác đụng NHIỀU bảng nên TẤT CẢ nằm trong MỘT transaction: thành công trọn vẹn,
// hoặc rollback sạch. authorID lấy từ context ở handler (KHÔNG từ body) -> chống giả mạo.
func (s *Service) Create(ctx context.Context, authorID int64, req CreatePostRequest) (*PostWithRelations, error) {
	// 1) Dựng entity từ request (đã Validate + chuẩn hóa ở handler).
	p := &Post{
		UserID:      authorID,
		Content:     strings.TrimSpace(req.Content),
		IsSponsored: req.IsSponsored,
	}

	// 2) Suy ra trạng thái vị trí: có place -> CONFIRMED; chỉ province -> UNKNOWN; không có -> UNKNOWN.
	switch {
	case req.PlaceID != nil:
		p.PlaceID = sql.NullInt64{Int64: *req.PlaceID, Valid: true}
		p.LocationStatus = LocationStatusConfirmed
	case req.ProvinceID != nil:
		p.ProvinceID = sql.NullInt64{Int64: *req.ProvinceID, Valid: true}
		p.LocationStatus = LocationStatusUnknown
	default:
		p.LocationStatus = LocationStatusUnknown
	}

	// 3) Chạy toàn bộ trong transaction.
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		// 3a) Kiểm địa điểm (nếu client gửi). Sai -> 400.
		if req.PlaceID != nil {
			ok, err := s.repo.PlaceExists(ctx, tx, *req.PlaceID)
			if err != nil {
				return err
			}
			if !ok {
				return apperr.BadRequest("place_id không tồn tại hoặc không khả dụng")
			}
		} else if req.ProvinceID != nil {
			ok, err := s.repo.ProvinceExists(ctx, tx, *req.ProvinceID)
			if err != nil {
				return err
			}
			if !ok {
				return apperr.BadRequest("province_id không tồn tại")
			}
		}

		// 3b) Kiểm ảnh: mọi media_id phải USABLE + thuộc author + chưa gắn bài khác.
		if len(req.MediaIDs) > 0 {
			n, err := s.repo.CountAttachableMedia(ctx, tx, authorID, req.MediaIDs)
			if err != nil {
				return err
			}
			if n != len(req.MediaIDs) {
				return apperr.BadRequest("một số ảnh không hợp lệ (không tồn tại, chưa USABLE, không thuộc về bạn, hoặc đã gắn bài khác)")
			}
		}

		// 3c) INSERT bài -> có p.ID.
		if err := s.repo.Create(ctx, tx, p); err != nil {
			return err
		}

		// 3d) Gắn ảnh theo đúng thứ tự client gửi.
		for i, mediaID := range req.MediaIDs {
			if err := s.repo.AttachImage(ctx, tx, p.ID, mediaID, i); err != nil {
				return err
			}
		}

		// 3e) Hashtag: upsert lấy id -> nối vào bài.
		for _, tag := range req.Hashtags {
			hashtagID, err := s.repo.UpsertHashtag(ctx, tx, tag)
			if err != nil {
				return err
			}
			if err := s.repo.LinkHashtag(ctx, tx, p.ID, hashtagID); err != nil {
				return err
			}
		}

		// 3f) Bài gắn place -> tăng bộ đếm bài của place.
		if req.PlaceID != nil {
			if err := s.repo.IncrementPlacePostCount(ctx, tx, *req.PlaceID); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sau khi commit, nạp ảnh + hashtag để trả response đầy đủ (giống endpoint đọc).
	return s.hydrateOne(ctx, p)
}

// Get lấy chi tiết 1 bài KÈM ảnh + hashtag (GET /posts/{id}).
func (s *Service) Get(ctx context.Context, id int64) (*PostWithRelations, error) {
	p, err := s.repo.GetByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return s.hydrateOne(ctx, p)
}

// hydrateOne nạp quan hệ cho một bài.
func (s *Service) hydrateOne(ctx context.Context, p *Post) (*PostWithRelations, error) {
	items, err := s.hydrate(ctx, []*Post{p})
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

// hydrate nạp ảnh + hashtag cho một loạt bài trong 2 câu query (không N+1), rồi zip lại.
func (s *Service) hydrate(ctx context.Context, posts []*Post) ([]*PostWithRelations, error) {
	out := make([]*PostWithRelations, 0, len(posts))
	if len(posts) == 0 {
		return out, nil
	}

	ids := make([]int64, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}

	imagesByPost, err := s.repo.ImagesByPostIDs(ctx, s.db, ids)
	if err != nil {
		return nil, err
	}
	tagsByPost, err := s.repo.HashtagsByPostIDs(ctx, s.db, ids)
	if err != nil {
		return nil, err
	}

	for _, p := range posts {
		out = append(out, &PostWithRelations{
			Post:     p,
			Images:   imagesByPost[p.ID],
			Hashtags: tagsByPost[p.ID],
		})
	}
	return out, nil
}

// Feed trả trang feed các bài VISIBLE mới nhất.
//
// MẸO limit+1: xin repo lấy dư 1 bài để biết CÓ CÒN trang sau không mà khỏi COUNT.
// Nếu lấy được > limit -> còn trang sau; cắt bớt về đúng limit trước khi trả.
func (s *Service) Feed(ctx context.Context, cursor *httpx.Cursor, limit int) ([]*PostWithRelations, bool, error) {
	posts, err := s.repo.ListVisible(ctx, s.db, cursor, limit+1)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}
	items, err := s.hydrate(ctx, posts)
	if err != nil {
		return nil, false, err
	}
	return items, hasMore, nil
}

// ListByUser trả các bài VISIBLE của một user (cùng kiểu phân trang như Feed).
func (s *Service) ListByUser(ctx context.Context, userID int64, cursor *httpx.Cursor, limit int) ([]*PostWithRelations, bool, error) {
	posts, err := s.repo.ListByUser(ctx, s.db, userID, cursor, limit+1)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}
	items, err := s.hydrate(ctx, posts)
	if err != nil {
		return nil, false, err
	}
	return items, hasMore, nil
}

// CountByUser đếm số bài đang hiển thị của một user (cho trang hồ sơ).
func (s *Service) CountByUser(ctx context.Context, userID int64) (int, error) {
	return s.repo.CountVisibleByUser(ctx, s.db, userID)
}

// Update sửa nội dung bài. Hai luật nghiệp vụ:
//   - Chỉ CHỦ bài được sửa (actorID phải trùng tác giả) -> nếu không: 403.
//   - Optimistic lock: version client gửi phải khớp version hiện tại -> nếu lệch: 409.
//
// Ta đọc bài trước để phân biệt rõ 404 (không có) vs 403 (không phải chủ); sau đó câu UPDATE
// có điều kiện version mới là "chốt" chống lost-update khi hai người sửa cùng lúc.
func (s *Service) Update(ctx context.Context, actorID, postID int64, req UpdatePostRequest) (*PostWithRelations, error) {
	existing, err := s.repo.GetByID(ctx, s.db, postID)
	if err != nil {
		return nil, err // ErrNotFound -> 404
	}
	if existing.UserID != actorID {
		return nil, apperr.Forbidden("bạn chỉ có thể sửa bài của chính mình")
	}

	updated, err := s.repo.UpdateContent(ctx, s.db, postID, actorID, strings.TrimSpace(req.Content), req.Version)
	if err != nil {
		// Bài vẫn tồn tại và đúng chủ (vừa kiểm ở trên), nên "không dòng khớp" ở đây
		// nghĩa là version đã lệch -> có người sửa xen giữa -> 409 Conflict.
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, apperr.Conflict("bài đã được cập nhật ở nơi khác, hãy tải lại rồi thử lại")
		}
		return nil, err
	}
	return s.hydrateOne(ctx, updated)
}

// Delete xóa mềm bài. Chỉ chủ bài được xóa.
//
// Chạy trong transaction vì phải BÙ TRỪ các bộ đếm denormalized đã tăng lúc tạo bài:
// giảm hashtags.post_count cho mọi tag của bài, và giảm places.post_count nếu bài gắn place.
// Nếu không bù, đếm sẽ phồng dần theo số bài bị xóa.
func (s *Service) Delete(ctx context.Context, actorID, postID int64) error {
	existing, err := s.repo.GetByID(ctx, s.db, postID)
	if err != nil {
		return err // ErrNotFound -> 404 (đã xóa trước đó cũng rơi vào đây)
	}
	if existing.UserID != actorID {
		return apperr.Forbidden("bạn chỉ có thể xóa bài của chính mình")
	}

	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.repo.SoftDelete(ctx, tx, postID, actorID); err != nil {
			return err
		}
		if err := s.repo.DecrementHashtagCountsByPost(ctx, tx, postID); err != nil {
			return err
		}
		if existing.PlaceID.Valid {
			if err := s.repo.DecrementPlacePostCount(ctx, tx, existing.PlaceID.Int64); err != nil {
				return err
			}
		}
		return nil
	})
}
