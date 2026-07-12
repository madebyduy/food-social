package media

import (
	"database/sql"
	"time"
)

// model.go — ENTITY ảnh upload, ánh xạ 1-1 với bảng `media_assets`.
//
// Luồng an toàn: client presign → PUT thẳng lên storage → confirm. Server không nhận URL
// tự do từ client; post chỉ tham chiếu media_id đã ở trạng thái USABLE (xem post.PostImage).

// Status — vòng đời của một media asset (khớp CHECK constraint trong migration).
type Status string

const (
	StatusPending  Status = "PENDING"  // vừa presign, chờ client PUT + confirm
	StatusUsable   Status = "USABLE"   // đã kiểm MIME/size, bỏ EXIF/GPS → gắn được vào bài
	StatusRejected Status = "REJECTED" // không hợp lệ
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusUsable, StatusRejected:
		return true
	default:
		return false
	}
}

// Asset là entity ánh xạ đúng các cột của bảng `media_assets`.
//
// Đặt tên là Asset (không phải MediaAsset) để tránh stutter: dùng ngoài package là media.Asset.
type Asset struct {
	ID          int64
	OwnerID     int64  // chỉ chủ sở hữu mới gắn được ảnh vào bài của mình
	StorageKey  string // key object trên S3/R2, UNIQUE
	MimeType    sql.NullString
	SizeBytes   sql.NullInt64 // cột size_bytes là INTEGER trong DB
	Width       sql.NullInt64
	Height      sql.NullInt64
	HasGPS      bool
	Status      Status
	CreatedAt   time.Time
	ConfirmedAt sql.NullTime
}
