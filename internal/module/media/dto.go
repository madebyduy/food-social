package media

import "database/sql"

// SignResponse — trả cho client sau /media/sign. Client dùng bộ này để PUT ảnh thẳng lên
// Cloudinary (multipart: file + api_key + timestamp + folder + public_id + signature).
type SignResponse struct {
	MediaID   int64  `json:"media_id"`
	UploadURL string `json:"upload_url"`
	CloudName string `json:"cloud_name"`
	APIKey    string `json:"api_key"`
	Timestamp int64  `json:"timestamp"`
	Folder    string `json:"folder"`
	PublicID  string `json:"public_id"`
	Signature string `json:"signature"`
}

// AssetResponse — thông tin một ảnh trả về sau confirm / khi cần.
type AssetResponse struct {
	ID         int64   `json:"id"`
	StorageKey string  `json:"storage_key"`
	URL        string  `json:"url"`
	MimeType   *string `json:"mime_type"`
	Width      *int64  `json:"width"`
	Height     *int64  `json:"height"`
	Status     string  `json:"status"`
}

func toSignResponse(mediaID int64, p UploadParams) SignResponse {
	return SignResponse{
		MediaID:   mediaID,
		UploadURL: p.UploadURL,
		CloudName: p.CloudName,
		APIKey:    p.APIKey,
		Timestamp: p.Timestamp,
		Folder:    p.Folder,
		PublicID:  p.PublicID,
		Signature: p.Signature,
	}
}

func toAssetResponse(a *Asset, url string) AssetResponse {
	return AssetResponse{
		ID:         a.ID,
		StorageKey: a.StorageKey,
		URL:        url,
		MimeType:   nullStringPtr(a.MimeType),
		Width:      nullInt64Ptr(a.Width),
		Height:     nullInt64Ptr(a.Height),
		Status:     string(a.Status),
	}
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
