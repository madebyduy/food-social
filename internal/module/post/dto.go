package post

import (
	"database/sql"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/madebyduy/food-social/internal/apperr"
)

const (
	maxContentLen = 5000
	maxImages     = 10 // số ảnh tối đa mỗi bài
	maxHashtags   = 10 // số hashtag tối đa mỗi bài
	maxTagLen     = 100
)

// CreatePostRequest là body của POST /api/v1/posts.
//
// Ảnh KHÔNG upload qua đây: client presign + PUT thẳng lên storage trước, tạo media_assets
// ở trạng thái USABLE, rồi chỉ gửi danh sách media_ids vào đây.
//
// Vị trí là TÙY CHỌN lúc đăng:
//   - Gửi place_id -> bài coi như đã xác nhận địa điểm (location_status=CONFIRMED).
//   - Chưa biết địa điểm: bỏ trống place_id, có thể kèm province_id để hiện trên feed theo tỉnh
//     (location_status=UNKNOWN); người khác đề xuất địa điểm sau.
//
// PlaceID/ProvinceID dùng *int64 (con trỏ) để phân biệt "không gửi" (nil) với "gửi số 0".
type CreatePostRequest struct {
	Content     string   `json:"content"`
	IsSponsored bool     `json:"is_sponsored"`
	MediaIDs    []int64  `json:"media_ids"`
	PlaceID     *int64   `json:"place_id"`
	ProvinceID  *int64   `json:"province_id"`
	Hashtags    []string `json:"hashtags"`
}

// PostResponse là hình dạng JSON trả cho client. Chỉ chứa field CÔNG KHAI
// (không lộ trọng số vote nội bộ, không lộ hidden_at/deleted_at...).
type PostResponse struct {
	ID             int64           `json:"id"`
	UserID         int64           `json:"user_id"`
	Content        string          `json:"content"`
	Status         string          `json:"status"`
	LocationStatus string          `json:"location_status"`
	PlaceID        *int64          `json:"place_id"`
	ProvinceID     *int64          `json:"province_id"`
	IsSponsored    bool            `json:"is_sponsored"`
	LikeCount      int             `json:"like_count"`
	CommentCount   int             `json:"comment_count"`
	SaveCount      int             `json:"save_count"`
	Images         []ImageResponse `json:"images"`
	Hashtags       []string        `json:"hashtags"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// ImageResponse là một ảnh trong PostResponse.
type ImageResponse struct {
	MediaID    int64  `json:"media_id"`
	StorageKey string `json:"storage_key"`
	Width      *int64 `json:"width"`
	Height     *int64 `json:"height"`
	SortOrder  int    `json:"sort_order"`
}

// Validate kiểm CÚ PHÁP đầu vào VÀ chuẩn hóa dữ liệu tại chỗ.
//
// Dùng con trỏ nhận (*CreatePostRequest) — KHÁC value receiver — để việc TrimSpace/lọc
// trùng/chuẩn hóa hashtag GHI ĐÈ được lên chính req mà handler đang giữ. Nhờ đó service
// nhận req đã sạch, khỏi chuẩn hóa lại.
func (r *CreatePostRequest) Validate() error {
	r.Content = strings.TrimSpace(r.Content)
	if r.Content == "" {
		return apperr.BadRequest("content không được để trống")
	}
	if utf8.RuneCountInString(r.Content) > maxContentLen {
		return apperr.BadRequest("content tối đa 5000 ký tự")
	}

	// media_ids: bỏ trùng (tránh gắn 1 ảnh 2 lần), chặn id <= 0, giới hạn số lượng.
	r.MediaIDs = dedupeInt64(r.MediaIDs)
	if len(r.MediaIDs) > maxImages {
		return apperr.BadRequest("tối đa 10 ảnh mỗi bài")
	}
	for _, id := range r.MediaIDs {
		if id <= 0 {
			return apperr.BadRequest("media_ids chứa id không hợp lệ")
		}
	}

	// place_id / province_id nếu có thì phải là số dương.
	if r.PlaceID != nil && *r.PlaceID <= 0 {
		return apperr.BadRequest("place_id không hợp lệ")
	}
	if r.ProvinceID != nil && *r.ProvinceID <= 0 {
		return apperr.BadRequest("province_id không hợp lệ")
	}

	// hashtags: chuẩn hóa (bỏ '#', lowercase, trim), bỏ rỗng, bỏ trùng.
	r.Hashtags = normalizeHashtags(r.Hashtags)
	if len(r.Hashtags) > maxHashtags {
		return apperr.BadRequest("tối đa 10 hashtag mỗi bài")
	}
	for _, t := range r.Hashtags {
		if utf8.RuneCountInString(t) > maxTagLen {
			return apperr.BadRequest("hashtag quá dài (tối đa 100 ký tự)")
		}
	}

	return nil
}

// dedupeInt64 loại phần tử trùng, GIỮ NGUYÊN thứ tự xuất hiện đầu tiên.
func dedupeInt64(in []int64) []int64 {
	if len(in) == 0 {
		return in
	}
	seen := make(map[int64]struct{}, len(in))
	out := make([]int64, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// normalizeHashtags chuẩn hóa từng tag về dạng lưu trong DB: lowercase, không '#', không rỗng,
// không trùng. (DB lưu tag lowercase, KHÔNG kèm '#' — xem model Hashtag.)
func normalizeHashtags(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		t := strings.ToLower(strings.TrimSpace(raw))
		t = strings.TrimPrefix(t, "#")
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// UpdatePostRequest là body của PATCH /api/v1/posts/{id}.
//
// Version là version bài mà client đang cầm (đọc từ lần GET trước) — server so với version
// hiện tại để chống lost-update (optimistic lock). Bản này chỉ sửa nội dung; sửa ảnh/hashtag
// để dành mốc sau.
type UpdatePostRequest struct {
	Content string `json:"content"`
	Version int    `json:"version"`
}

// Validate kiểm cú pháp + chuẩn hóa content.
func (r *UpdatePostRequest) Validate() error {
	r.Content = strings.TrimSpace(r.Content)
	if r.Content == "" {
		return apperr.BadRequest("content không được để trống")
	}
	if utf8.RuneCountInString(r.Content) > maxContentLen {
		return apperr.BadRequest("content tối đa 5000 ký tự")
	}
	if r.Version <= 0 {
		return apperr.BadRequest("version không hợp lệ")
	}
	return nil
}

// toPostResponses map slice bài-kèm-quan-hệ -> slice DTO (cho các endpoint list).
func toPostResponses(items []*PostWithRelations) []PostResponse {
	out := make([]PostResponse, 0, len(items))
	for _, it := range items {
		out = append(out, toPostResponse(it))
	}
	return out
}

// toPostResponse map một bài kèm ảnh + hashtag -> DTO công khai.
//
// Images/Hashtags luôn được khởi tạo non-nil để JSON trả [] (không phải null) khi bài
// chưa có ảnh/tag — client khỏi phải kiểm null.
func toPostResponse(it *PostWithRelations) PostResponse {
	p := it.Post
	resp := PostResponse{
		ID:             p.ID,
		UserID:         p.UserID,
		Content:        p.Content,
		Status:         string(p.Status),
		LocationStatus: string(p.LocationStatus),
		PlaceID:        nullInt64Ptr(p.PlaceID),
		ProvinceID:     nullInt64Ptr(p.ProvinceID),
		IsSponsored:    p.IsSponsored,
		LikeCount:      p.LikeCount,
		CommentCount:   p.CommentCount,
		SaveCount:      p.SaveCount,
		Images:         toImageResponses(it.Images),
		Hashtags:       it.Hashtags,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if resp.Hashtags == nil {
		resp.Hashtags = []string{}
	}
	return resp
}

// toImageResponses map các ảnh của bài -> DTO (luôn non-nil).
func toImageResponses(images []PostImageView) []ImageResponse {
	out := make([]ImageResponse, 0, len(images))
	for _, img := range images {
		out = append(out, ImageResponse{
			MediaID:    img.MediaID,
			StorageKey: img.StorageKey,
			Width:      nullInt64Ptr(img.Width),
			Height:     nullInt64Ptr(img.Height),
			SortOrder:  img.SortOrder,
		})
	}
	return out
}

// nullInt64Ptr đổi sql.NullInt64 -> *int64 để JSON hiện null khi không có giá trị.
func nullInt64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}
