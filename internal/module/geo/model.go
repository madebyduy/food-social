package geo

import "time"

// model.go — ENTITY địa lý, ánh xạ 1-1 với các bảng provinces/districts/wards.
//
// Phân cấp: Province (tỉnh/thành) → District (quận/huyện) → Ward (phường/xã).
// Các bảng này ít thay đổi (thường seed sẵn), nên không có cột soft-delete.

// Province — tỉnh/thành. Ví dụ: "Hải Phòng".
type Province struct {
	ID        int64
	Name      string
	Slug      string // không dấu, duy nhất toàn hệ thống; dùng cho URL/feed theo tỉnh
	CreatedAt time.Time
}

// District — quận/huyện thuộc một Province.
type District struct {
	ID         int64
	ProvinceID int64
	Name       string
	Slug       string // duy nhất TRONG một tỉnh (UNIQUE(province_id, slug))
	CreatedAt  time.Time
}

// Ward — phường/xã thuộc một District.
type Ward struct {
	ID         int64
	DistrictID int64
	Name       string
	Slug       string // duy nhất TRONG một quận/huyện
	CreatedAt  time.Time
}
