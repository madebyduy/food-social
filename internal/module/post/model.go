package post

import (
	"database/sql"
	"time"
)

type Status string

const (
	StatusVisible           Status = "VISIBLE"
	StatusHiddenByCommunity Status = "HIDDEN_BY_COMMUNITY"
	StatusHiddenByAdmin     Status = "HIDDEN_BY_ADMIN"
	StatusDeletedByAuthor   Status = "DELETED_BY_AUTHOR"
)

func (s Status) Valid() bool {
	switch s {
	case StatusVisible,
		StatusHiddenByCommunity,
		StatusHiddenByAdmin,
		StatusDeletedByAuthor:
		return true
	default:
		return false
	}
}

type LocationStatus string

const (
	LocationStatusUnknown   LocationStatus = "UNKNOWN"
	LocationStatusSuggested LocationStatus = "SUGGESTED"
	LocationStatusConfirmed LocationStatus = "CONFIRMED"
)

func (s LocationStatus) Valid() bool {
	switch s {
	case LocationStatusUnknown,
		LocationStatusSuggested,
		LocationStatusConfirmed:
		return true
	default:
		return false
	}
}

type Post struct {
	ID         int64
	UserID     int64
	PlaceID    sql.NullInt64
	ProvinceID sql.NullInt64

	Content        string
	Status         Status
	LocationStatus LocationStatus
	Version        int
	IsSponsored    bool

	TrustedWeight   float64
	UntrustedWeight float64
	TotalVoteCount  int
	UntrustedRatio  float64

	LikeCount    int
	CommentCount int
	SaveCount    int

	CreatedAt time.Time
	UpdatedAt time.Time
	HiddenAt  sql.NullTime
	DeletedAt sql.NullTime
}

// PostImage — liên kết ảnh với bài, ánh xạ 1-1 với bảng `post_images`.
//
// post_images KHÔNG lưu URL. Nó trỏ tới media_assets qua MediaID; ảnh phải đã ở trạng thái
// USABLE và thuộc đúng chủ bài trước khi được gắn (kiểm ở tầng service khi tạo/sửa post).
type PostImage struct {
	ID        int64
	PostID    int64
	MediaID   int64 // tham chiếu media_assets.id (một ảnh chỉ gắn một bài — UNIQUE)
	SortOrder int   // thứ tự hiển thị ảnh trong bài
	CreatedAt time.Time
}

// Hashtag — thẻ hashtag, ánh xạ 1-1 với bảng `hashtags`.
//
// Quan hệ nhiều-nhiều với post qua PostHashtag: một bài nhiều tag, một tag nhiều bài.
// Tag lưu lowercase, KHÔNG kèm dấu '#'. Chuẩn hóa chuỗi tag (bỏ '#', lower, trim) làm ở service.
type Hashtag struct {
	ID        int64
	Tag       string // lowercase, không dấu '#', UNIQUE
	PostCount int    // denormalized: số bài dùng tag này
	CreatedAt time.Time
}

// PostHashtag — bảng nối `post_hashtags` (khóa chính kép post_id + hashtag_id).
type PostHashtag struct {
	PostID    int64
	HashtagID int64
}
