package post

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/madebyduy/food-social/internal/apperr"
)
const maxContentLen = 5000
type CreatePostRequets struct {
	Content     string `json:"content"`
	IsSponsored bool   `json:"is_sponsored"`
}

type PostResponse struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	Content        string    `json:"content"`
	Status         string    `json:"status"`
	LocationStatus string    `json:"location_status"`
	IsSponsored    bool      `json:"is_sponsored"`
	LikeCount      int       `json:"like_count"`
	CommentCount   int       `json:"comment_count"`
	SaveCount      int       `json:"save_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (r CreatePostRequets) Validate() error {
	r.Content = strings.TrimSpace(r.Content)
	if r.Content == "" {
		return apperr.BadRequest("content không được để trống")
	}
	if utf8.RuneCountInString(r.Content)>maxContentLen {
		return apperr.BadRequest("content tối đa 5000 ký tự")
	}
	return nil
}

func toPostResponse(p *Post) PostResponse {
	return PostResponse{
		ID:             p.ID,
		UserID:         p.UserID,
		Content:        p.Content,
		Status:         string(p.Status),
		LocationStatus: string(p.LocationStatus),
		IsSponsored:    p.IsSponsored,
		LikeCount:      p.LikeCount,
		CommentCount:   p.CommentCount,
		SaveCount:      p.SaveCount,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}