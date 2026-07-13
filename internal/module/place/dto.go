package place

import (
	"database/sql"
	"strings"
	"time"

	"github.com/madebyduy/food-social/internal/apperr"
)

const maxNameLen = 255

// CreatePlaceRequest — body POST /api/v1/places. Chỉ name bắt buộc; phần còn lại tùy chọn.
// Con trỏ (*T) để phân biệt "không gửi" (nil) với giá trị 0/"".
type CreatePlaceRequest struct {
	Name          string   `json:"name"`
	Address       *string  `json:"address"`
	ProvinceID    *int64   `json:"province_id"`
	DistrictID    *int64   `json:"district_id"`
	WardID        *int64   `json:"ward_id"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	GooglePlaceID *string  `json:"google_place_id"`
}

type PlaceResponse struct {
	ID            int64     `json:"id"`
	GooglePlaceID *string   `json:"google_place_id"`
	Name          string    `json:"name"`
	Address       *string   `json:"address"`
	ProvinceID    *int64    `json:"province_id"`
	DistrictID    *int64    `json:"district_id"`
	WardID        *int64    `json:"ward_id"`
	Latitude      *float64  `json:"latitude"`
	Longitude     *float64  `json:"longitude"`
	PostCount     int       `json:"post_count"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (r *CreatePlaceRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return apperr.BadRequest("name không được để trống")
	}
	if len([]rune(r.Name)) > maxNameLen {
		return apperr.BadRequest("name tối đa 255 ký tự")
	}
	if r.Latitude != nil && (*r.Latitude < -90 || *r.Latitude > 90) {
		return apperr.BadRequest("latitude phải trong khoảng [-90, 90]")
	}
	if r.Longitude != nil && (*r.Longitude < -180 || *r.Longitude > 180) {
		return apperr.BadRequest("longitude phải trong khoảng [-180, 180]")
	}
	return nil
}

func toPlaceResponse(p *Place) PlaceResponse {
	return PlaceResponse{
		ID:            p.ID,
		GooglePlaceID: nullStringPtr(p.GooglePlaceID),
		Name:          p.Name,
		Address:       nullStringPtr(p.Address),
		ProvinceID:    nullInt64Ptr(p.ProvinceID),
		DistrictID:    nullInt64Ptr(p.DistrictID),
		WardID:        nullInt64Ptr(p.WardID),
		Latitude:      nullFloat64Ptr(p.Latitude),
		Longitude:     nullFloat64Ptr(p.Longitude),
		PostCount:     p.PostCount,
		Status:        string(p.Status),
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func toPlaceResponses(items []*Place) []PlaceResponse {
	out := make([]PlaceResponse, 0, len(items))
	for _, p := range items {
		out = append(out, toPlaceResponse(p))
	}
	return out
}

func nullStringPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	v := n.String
	return &v
}

func nullInt64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func nullFloat64Ptr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}
