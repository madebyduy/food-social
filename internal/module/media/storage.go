package media

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/madebyduy/food-social/internal/apperr"
)

// storage.go — trừu tượng hóa nơi lưu ảnh + hiện thực Cloudinary.
//
// Luồng an toàn "signed direct upload":
//  1. Client gọi /media/sign  -> server tạo media_assets PENDING + ký (signature) các tham số.
//  2. Client PUT ảnh THẲNG lên Cloudinary kèm signature (không qua server -> nhẹ băng thông server).
//  3. Client gọi /media/{id}/confirm -> server GỌI Admin API Cloudinary xác minh ảnh có thật,
//     lấy width/height/format, rồi đánh dấu USABLE. Server là nguồn sự thật, KHÔNG tin client.
//
// Storage là interface -> muốn đổi sang S3/R2 chỉ cần viết implement khác, không sửa service.

// UploadParams là bộ tham số + chữ ký server trả cho client để upload trực tiếp.
type UploadParams struct {
	UploadURL  string
	CloudName  string
	APIKey     string
	Timestamp  int64
	Folder     string
	PublicID   string // tên ảnh (chưa kèm folder)
	Signature  string
	StorageKey string // public_id ĐẦY ĐỦ (folder/tên) — lưu vào media_assets.storage_key
}

// ResourceInfo là metadata lấy về khi xác minh ảnh trên Cloudinary.
type ResourceInfo struct {
	Format    string
	Bytes     int64
	Width     int
	Height    int
	SecureURL string
}

type Storage interface {
	// SignUpload tạo tham số + chữ ký cho một ảnh mới (publicID do service sinh ngẫu nhiên).
	SignUpload(publicID string) UploadParams
	// FetchResource xác minh ảnh tồn tại trên Cloudinary và trả metadata. Không có -> ErrNotFound.
	FetchResource(ctx context.Context, storageKey string) (*ResourceInfo, error)
	// PublicURL dựng URL hiển thị công khai từ storage_key.
	PublicURL(storageKey string) string
}

type cloudinaryStorage struct {
	cloudName string
	apiKey    string
	apiSecret string
	folder    string
	http      *http.Client
}

// NewCloudinaryStorage tạo Storage dùng Cloudinary.
func NewCloudinaryStorage(cloudName, apiKey, apiSecret, folder string) Storage {
	return &cloudinaryStorage{
		cloudName: cloudName,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		folder:    folder,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *cloudinaryStorage) SignUpload(publicID string) UploadParams {
	ts := time.Now().Unix()

	// Chữ ký Cloudinary: ghép các tham số (ĐÃ sắp xếp theo alphabet) dạng key=value nối bằng '&',
	// nối thêm api_secret ở cuối, rồi SHA1 hex. Ở đây: folder, public_id, timestamp.
	toSign := fmt.Sprintf("folder=%s&public_id=%s&timestamp=%d", c.folder, publicID, ts)

	h := sha1.New()
	_, _ = io.WriteString(h, toSign+c.apiSecret)
	signature := hex.EncodeToString(h.Sum(nil))

	return UploadParams{
		UploadURL:  fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", c.cloudName),
		CloudName:  c.cloudName,
		APIKey:     c.apiKey,
		Timestamp:  ts,
		Folder:     c.folder,
		PublicID:   publicID,
		Signature:  signature,
		StorageKey: c.folder + "/" + publicID,
	}
}

func (c *cloudinaryStorage) FetchResource(ctx context.Context, storageKey string) (*ResourceInfo, error) {
	// Admin API: GET /resources/image/upload/{public_id}, xác thực Basic (api_key:api_secret).
	endpoint := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/resources/image/upload/%s",
		c.cloudName, url.PathEscape(storageKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("cloudinary build request: %w", err))
	}
	req.SetBasicAuth(c.apiKey, c.apiSecret)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("cloudinary call: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Ảnh chưa được upload lên Cloudinary (client chưa PUT hoặc PUT hỏng).
		return nil, apperr.BadRequest("ảnh chưa được upload lên storage, không thể xác nhận")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, apperr.Internal(fmt.Errorf("cloudinary status %d: %s", resp.StatusCode, body))
	}

	var payload struct {
		Format    string `json:"format"`
		Bytes     int64  `json:"bytes"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		SecureURL string `json:"secure_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, apperr.Internal(fmt.Errorf("cloudinary decode: %w", err))
	}

	return &ResourceInfo{
		Format:    payload.Format,
		Bytes:     payload.Bytes,
		Width:     payload.Width,
		Height:    payload.Height,
		SecureURL: payload.SecureURL,
	}, nil
}

func (c *cloudinaryStorage) PublicURL(storageKey string) string {
	return fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/%s", c.cloudName, storageKey)
}
