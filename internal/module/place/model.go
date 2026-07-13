package place

import (
	"database/sql"
	"time"
)

// model.go — ENTITY địa điểm ăn uống, ánh xạ 1-1 với bảng `places` (và place_merge_history).
//
// Nhiều bài review gom về một place. Place trùng lặp được GỘP bằng canonical_place_id
// (chỉ đổi một con trỏ) thay vì sửa place_id trên hàng nghìn bài.

// Status — trạng thái place (khớp CHECK constraint trong migration).
type Status string

const (
	StatusActive Status = "ACTIVE" // place bình thường
	StatusMerged Status = "MERGED" // đã gộp vào canonical khác
	StatusHidden Status = "HIDDEN" // bị ẩn (hậu kiểm)
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusMerged, StatusHidden:
		return true
	default:
		return false
	}
}

// Place là entity ánh xạ đúng các cột của bảng `places`.
//
// CanonicalPlaceID NULL nghĩa là place này TỰ nó là canonical (place gốc).
// Cột generated search_vector KHÔNG có ở đây — DB tự tính, repository không đọc/ghi.
type Place struct {
	ID               int64
	CanonicalPlaceID sql.NullInt64  // NULL = tự nó là canonical
	GooglePlaceID    sql.NullString // NULL = place nhập tay chưa có Google ID
	Name             string
	Address          sql.NullString
	ProvinceID       sql.NullInt64
	DistrictID       sql.NullInt64
	WardID           sql.NullInt64
	Latitude         sql.NullFloat64 // DECIMAL(10,7) trong DB
	Longitude        sql.NullFloat64
	PostCount        int // denormalized cho trang place
	Status           Status
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PlaceMergeHistory — một dòng audit cho mỗi lần admin gộp place B vào canonical A.
type PlaceMergeHistory struct {
	ID               int64
	MergedPlaceID    int64 // place bị gộp (B)
	CanonicalPlaceID int64 // place giữ lại (A)
	MergedBy         int64 // admin thực hiện
	Reason           sql.NullString
	CreatedAt        time.Time
}
