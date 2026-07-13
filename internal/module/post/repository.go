package post

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
	"github.com/madebyduy/food-social/internal/httpx"
)

// postColumns là danh sách cột đọc ra một Post ĐẦY ĐỦ. Gom vào một const để GetByID,
// feed, list-by-user, update... dùng chung, tránh viết lại 20 cột ở nhiều nơi (lệch cột
// là nguồn bug scan kinh điển).
const postColumns = `
	id, user_id, place_id, province_id,
	content, status, location_status, version, is_sponsored,
	trusted_weight, untrusted_weight, total_vote_count, untrusted_ratio,
	like_count, comment_count, save_count,
	created_at, updated_at, hidden_at, deleted_at`

// Repository là INTERFACE mô tả các thao tác DB mà module post cần.
//
// Mọi method nhận database.Querier (KHÔNG giữ *sql.DB) để Service quyết định chạy trong
// transaction hay không: tạo bài (nhiều bảng) chạy trong *sql.Tx; đọc/đếm đơn thì truyền
// thẳng *sql.DB.
type Repository interface {
	Create(ctx context.Context, q database.Querier, p *Post) error
	GetByID(ctx context.Context, q database.Querier, id int64) (*Post, error)

	// ListVisible trả feed các bài VISIBLE, sắp created_at DESC, id DESC (keyset).
	// cursor == nil -> trang đầu; ngược lại lấy các bài "cũ hơn" cursor.
	ListVisible(ctx context.Context, q database.Querier, cursor *httpx.Cursor, limit int) ([]*Post, error)

	// ListByUser trả các bài VISIBLE của một user (cùng kiểu phân trang keyset).
	ListByUser(ctx context.Context, q database.Querier, userID int64, cursor *httpx.Cursor, limit int) ([]*Post, error)

	// CountVisibleByUser đếm số bài đang hiển thị của một user (cho trang hồ sơ).
	CountVisibleByUser(ctx context.Context, q database.Querier, userID int64) (int, error)

	// UpdateContent sửa nội dung bài với OPTIMISTIC LOCK: chỉ ghi khi version còn khớp
	// expectedVersion và bài thuộc actorID. Không có dòng khớp -> apperr.ErrNotFound
	// (Service diễn giải: version lệch -> 409, hoặc không phải chủ/không tồn tại).
	UpdateContent(ctx context.Context, q database.Querier, postID, actorID int64, content string, expectedVersion int) (*Post, error)

	// SoftDelete xóa mềm bài của actorID. Không có dòng khớp -> apperr.ErrNotFound.
	SoftDelete(ctx context.Context, q database.Querier, postID, actorID int64) error

	// ImagesByPostIDs nạp ảnh cho NHIỀU bài trong MỘT câu (chống N+1), gom theo post_id.
	ImagesByPostIDs(ctx context.Context, q database.Querier, postIDs []int64) (map[int64][]PostImageView, error)

	// HashtagsByPostIDs nạp hashtag cho NHIỀU bài trong MỘT câu, gom theo post_id.
	HashtagsByPostIDs(ctx context.Context, q database.Querier, postIDs []int64) (map[int64][]string, error)

	// DecrementHashtagCountsByPost giảm post_count của mọi hashtag mà bài đang dùng (khi xóa bài).
	DecrementHashtagCountsByPost(ctx context.Context, q database.Querier, postID int64) error

	// DecrementPlacePostCount giảm bộ đếm bài của một place (khi xóa bài có gắn place).
	DecrementPlacePostCount(ctx context.Context, q database.Querier, placeID int64) error

	// --- Dùng khi tạo bài đầy đủ (ảnh + hashtag + place) ---
	CountAttachableMedia(ctx context.Context, q database.Querier, ownerID int64, mediaIDs []int64) (int, error)
	AttachImage(ctx context.Context, q database.Querier, postID, mediaID int64, sortOrder int) error
	UpsertHashtag(ctx context.Context, q database.Querier, tag string) (int64, error)
	LinkHashtag(ctx context.Context, q database.Querier, postID, hashtagID int64) error
	PlaceExists(ctx context.Context, q database.Querier, placeID int64) (bool, error)
	ProvinceExists(ctx context.Context, q database.Querier, provinceID int64) (bool, error)
	IncrementPlacePostCount(ctx context.Context, q database.Querier, placeID int64) error
}

// repository là implement cụ thể (private). Stateless -> struct rỗng.
type repository struct{}

// NewRepository trả về Repository. *sql.DB không được giữ lại (đánh dấu _).
func NewRepository(_ *sql.DB) Repository {
	return &repository{}
}

// rowScanner là phần chung của *sql.Row và *sql.Rows — nhờ vậy scanPost dùng được cho cả
// QueryRow (một dòng) lẫn vòng lặp rows.Next() (nhiều dòng).
type rowScanner interface {
	Scan(dest ...any) error
}

// scanPost đọc một dòng theo đúng thứ tự postColumns thành *Post.
func scanPost(sc rowScanner) (*Post, error) {
	var p Post
	err := sc.Scan(
		&p.ID, &p.UserID, &p.PlaceID, &p.ProvinceID,
		&p.Content, &p.Status, &p.LocationStatus, &p.Version, &p.IsSponsored,
		&p.TrustedWeight, &p.UntrustedWeight, &p.TotalVoteCount, &p.UntrustedRatio,
		&p.LikeCount, &p.CommentCount, &p.SaveCount,
		&p.CreatedAt, &p.UpdatedAt, &p.HiddenAt, &p.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// queryPosts chạy một câu SELECT trả NHIỀU bài và scan hết thành slice.
func queryPosts(ctx context.Context, q database.Querier, query string, args ...any) ([]*Post, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("post list query: %w", err)
	}
	defer rows.Close()

	var out []*Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("post list scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("post list rows: %w", err)
	}
	return out, nil
}

// Create — INSERT bài mới. place_id/province_id/location_status do Service tính sẵn.
func (r *repository) Create(ctx context.Context, q database.Querier, p *Post) error {
	const query = `
		INSERT INTO posts (user_id, content, is_sponsored, place_id, province_id, location_status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, status, version, created_at, updated_at`

	err := q.QueryRowContext(ctx, query,
		p.UserID, p.Content, p.IsSponsored, p.PlaceID, p.ProvinceID, p.LocationStatus,
	).Scan(
		&p.ID, &p.Status, &p.Version, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("post.Create insert: %w", err)
	}
	return nil
}

// GetByID — đọc 1 bài theo id, bỏ qua bài đã xóa mềm.
func (r *repository) GetByID(ctx context.Context, q database.Querier, id int64) (*Post, error) {
	query := `SELECT ` + postColumns + ` FROM posts WHERE id = $1 AND deleted_at IS NULL`

	p, err := scanPost(q.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("post.GetByID scan: %w", err)
	}
	return p, nil
}

// ListVisible — feed keyset. Khớp partial index ix_posts_feed(created_at DESC, id DESC)
// WHERE status='VISIBLE'. So sánh bộ (created_at, id) đảm bảo thứ tự tuyệt đối.
func (r *repository) ListVisible(ctx context.Context, q database.Querier, cursor *httpx.Cursor, limit int) ([]*Post, error) {
	if cursor == nil {
		query := `SELECT ` + postColumns + `
			FROM posts
			WHERE status = 'VISIBLE'
			ORDER BY created_at DESC, id DESC
			LIMIT $1`
		return queryPosts(ctx, q, query, limit)
	}

	query := `SELECT ` + postColumns + `
		FROM posts
		WHERE status = 'VISIBLE' AND (created_at, id) < ($2, $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $1`
	return queryPosts(ctx, q, query, limit, cursor.CreatedAt, cursor.ID)
}

// ListByUser — bài VISIBLE của một user, phân trang keyset.
func (r *repository) ListByUser(ctx context.Context, q database.Querier, userID int64, cursor *httpx.Cursor, limit int) ([]*Post, error) {
	if cursor == nil {
		query := `SELECT ` + postColumns + `
			FROM posts
			WHERE user_id = $1 AND status = 'VISIBLE' AND deleted_at IS NULL
			ORDER BY created_at DESC, id DESC
			LIMIT $2`
		return queryPosts(ctx, q, query, userID, limit)
	}

	query := `SELECT ` + postColumns + `
		FROM posts
		WHERE user_id = $1 AND status = 'VISIBLE' AND deleted_at IS NULL
		  AND (created_at, id) < ($3, $4)
		ORDER BY created_at DESC, id DESC
		LIMIT $2`
	return queryPosts(ctx, q, query, userID, limit, cursor.CreatedAt, cursor.ID)
}

// CountVisibleByUser — đếm bài đang hiển thị của một user.
func (r *repository) CountVisibleByUser(ctx context.Context, q database.Querier, userID int64) (int, error) {
	const query = `
		SELECT count(*) FROM posts
		WHERE user_id = $1 AND status = 'VISIBLE' AND deleted_at IS NULL`

	var n int
	if err := q.QueryRowContext(ctx, query, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("post.CountVisibleByUser: %w", err)
	}
	return n, nil
}

// UpdateContent — sửa nội dung với optimistic lock. Điều kiện version = expectedVersion
// là "khóa lạc quan": nếu ai đó đã sửa bài giữa lúc client đọc và lúc gửi PATCH thì version
// đã tăng, WHERE không khớp -> 0 dòng -> ErrNoRows -> ErrNotFound. version tự +1 mỗi lần sửa.
func (r *repository) UpdateContent(ctx context.Context, q database.Querier, postID, actorID int64, content string, expectedVersion int) (*Post, error) {
	query := `
		UPDATE posts
		SET content = $1, version = version + 1, updated_at = now()
		WHERE id = $2 AND user_id = $3 AND version = $4 AND deleted_at IS NULL
		RETURNING ` + postColumns

	p, err := scanPost(q.QueryRowContext(ctx, query, content, postID, actorID, expectedVersion))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("post.UpdateContent: %w", err)
	}
	return p, nil
}

// SoftDelete — đánh dấu xóa mềm. RowsAffected == 0 nghĩa là bài không tồn tại, đã xóa,
// hoặc không thuộc actorID -> ErrNotFound.
func (r *repository) SoftDelete(ctx context.Context, q database.Querier, postID, actorID int64) error {
	const query = `
		UPDATE posts
		SET status = 'DELETED_BY_AUTHOR', deleted_at = now(), updated_at = now()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	res, err := q.ExecContext(ctx, query, postID, actorID)
	if err != nil {
		return fmt.Errorf("post.SoftDelete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("post.SoftDelete rows: %w", err)
	}
	if n == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// ImagesByPostIDs — nạp ảnh cho một loạt bài (một câu JOIN), gom vào map[post_id][]ảnh.
// Sắp theo (post_id, sort_order) để ảnh trong mỗi bài đúng thứ tự hiển thị.
func (r *repository) ImagesByPostIDs(ctx context.Context, q database.Querier, postIDs []int64) (map[int64][]PostImageView, error) {
	result := make(map[int64][]PostImageView)
	if len(postIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT pi.post_id, pi.media_id, m.storage_key, m.width, m.height, pi.sort_order
		FROM post_images pi
		JOIN media_assets m ON m.id = pi.media_id
		WHERE pi.post_id = ANY($1)
		ORDER BY pi.post_id, pi.sort_order`

	rows, err := q.QueryContext(ctx, query, postIDs)
	if err != nil {
		return nil, fmt.Errorf("post.ImagesByPostIDs query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var postID int64
		var v PostImageView
		if err := rows.Scan(&postID, &v.MediaID, &v.StorageKey, &v.Width, &v.Height, &v.SortOrder); err != nil {
			return nil, fmt.Errorf("post.ImagesByPostIDs scan: %w", err)
		}
		result[postID] = append(result[postID], v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("post.ImagesByPostIDs rows: %w", err)
	}
	return result, nil
}

// HashtagsByPostIDs — nạp hashtag cho một loạt bài (một câu JOIN), gom vào map[post_id][]tag.
func (r *repository) HashtagsByPostIDs(ctx context.Context, q database.Querier, postIDs []int64) (map[int64][]string, error) {
	result := make(map[int64][]string)
	if len(postIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT ph.post_id, h.tag
		FROM post_hashtags ph
		JOIN hashtags h ON h.id = ph.hashtag_id
		WHERE ph.post_id = ANY($1)
		ORDER BY ph.post_id, h.tag`

	rows, err := q.QueryContext(ctx, query, postIDs)
	if err != nil {
		return nil, fmt.Errorf("post.HashtagsByPostIDs query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var postID int64
		var tag string
		if err := rows.Scan(&postID, &tag); err != nil {
			return nil, fmt.Errorf("post.HashtagsByPostIDs scan: %w", err)
		}
		result[postID] = append(result[postID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("post.HashtagsByPostIDs rows: %w", err)
	}
	return result, nil
}

// DecrementHashtagCountsByPost — giảm post_count của mọi hashtag bài đang dùng.
// GREATEST(... , 0) chặn đếm âm (phòng dữ liệu lệch trước đó).
func (r *repository) DecrementHashtagCountsByPost(ctx context.Context, q database.Querier, postID int64) error {
	const query = `
		UPDATE hashtags
		SET post_count = GREATEST(post_count - 1, 0)
		WHERE id IN (SELECT hashtag_id FROM post_hashtags WHERE post_id = $1)`

	if _, err := q.ExecContext(ctx, query, postID); err != nil {
		return fmt.Errorf("post.DecrementHashtagCountsByPost: %w", err)
	}
	return nil
}

// DecrementPlacePostCount — giảm bộ đếm bài của place.
func (r *repository) DecrementPlacePostCount(ctx context.Context, q database.Querier, placeID int64) error {
	const query = `UPDATE places SET post_count = GREATEST(post_count - 1, 0), updated_at = now() WHERE id = $1`

	if _, err := q.ExecContext(ctx, query, placeID); err != nil {
		return fmt.Errorf("post.DecrementPlacePostCount: %w", err)
	}
	return nil
}

// CountAttachableMedia — đếm số ảnh hợp lệ (USABLE + đúng chủ + chưa gắn bài khác).
func (r *repository) CountAttachableMedia(ctx context.Context, q database.Querier, ownerID int64, mediaIDs []int64) (int, error) {
	const query = `
		SELECT count(*)
		FROM media_assets m
		WHERE m.owner_id = $1
		  AND m.status = 'USABLE'
		  AND m.id = ANY($2)
		  AND NOT EXISTS (SELECT 1 FROM post_images pi WHERE pi.media_id = m.id)`

	var n int
	if err := q.QueryRowContext(ctx, query, ownerID, mediaIDs).Scan(&n); err != nil {
		return 0, fmt.Errorf("post.CountAttachableMedia: %w", err)
	}
	return n, nil
}

// AttachImage — INSERT một dòng post_images.
func (r *repository) AttachImage(ctx context.Context, q database.Querier, postID, mediaID int64, sortOrder int) error {
	const query = `INSERT INTO post_images (post_id, media_id, sort_order) VALUES ($1, $2, $3)`

	if _, err := q.ExecContext(ctx, query, postID, mediaID, sortOrder); err != nil {
		return fmt.Errorf("post.AttachImage: %w", err)
	}
	return nil
}

// UpsertHashtag — tạo tag nếu chưa có, hoặc tăng post_count nếu đã có, rồi trả về id.
func (r *repository) UpsertHashtag(ctx context.Context, q database.Querier, tag string) (int64, error) {
	const query = `
		INSERT INTO hashtags (tag, post_count)
		VALUES ($1, 1)
		ON CONFLICT (tag) DO UPDATE SET post_count = hashtags.post_count + 1
		RETURNING id`

	var id int64
	if err := q.QueryRowContext(ctx, query, tag).Scan(&id); err != nil {
		return 0, fmt.Errorf("post.UpsertHashtag: %w", err)
	}
	return id, nil
}

// LinkHashtag — nối bài với hashtag.
func (r *repository) LinkHashtag(ctx context.Context, q database.Querier, postID, hashtagID int64) error {
	const query = `INSERT INTO post_hashtags (post_id, hashtag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`

	if _, err := q.ExecContext(ctx, query, postID, hashtagID); err != nil {
		return fmt.Errorf("post.LinkHashtag: %w", err)
	}
	return nil
}

// PlaceExists — true nếu place tồn tại và đang ACTIVE.
func (r *repository) PlaceExists(ctx context.Context, q database.Querier, placeID int64) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM places WHERE id = $1 AND status = 'ACTIVE')`

	var ok bool
	if err := q.QueryRowContext(ctx, query, placeID).Scan(&ok); err != nil {
		return false, fmt.Errorf("post.PlaceExists: %w", err)
	}
	return ok, nil
}

// ProvinceExists — true nếu province tồn tại.
func (r *repository) ProvinceExists(ctx context.Context, q database.Querier, provinceID int64) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM provinces WHERE id = $1)`

	var ok bool
	if err := q.QueryRowContext(ctx, query, provinceID).Scan(&ok); err != nil {
		return false, fmt.Errorf("post.ProvinceExists: %w", err)
	}
	return ok, nil
}

// IncrementPlacePostCount — tăng bộ đếm bài của place.
func (r *repository) IncrementPlacePostCount(ctx context.Context, q database.Querier, placeID int64) error {
	const query = `UPDATE places SET post_count = post_count + 1, updated_at = now() WHERE id = $1`

	if _, err := q.ExecContext(ctx, query, placeID); err != nil {
		return fmt.Errorf("post.IncrementPlacePostCount: %w", err)
	}
	return nil
}
