package media

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/madebyduy/food-social/internal/apperr"
)

type Service struct {
	db      *sql.DB
	repo    Repository
	storage Storage
	log     *slog.Logger
}

func NewService(db *sql.DB, repo Repository, storage Storage, log *slog.Logger) *Service {
	return &Service{db: db, repo: repo, storage: storage, log: log}
}

// Sign tạo một ảnh PENDING + bộ tham số ký để client upload thẳng lên Cloudinary.
func (s *Service) Sign(ctx context.Context, ownerID int64) (*Asset, UploadParams, error) {
	name, err := randomName()
	if err != nil {
		return nil, UploadParams{}, apperr.Internal(err)
	}

	params := s.storage.SignUpload(name)

	asset := &Asset{
		OwnerID:    ownerID,
		StorageKey: params.StorageKey,
		Status:     StatusPending,
	}
	if err := s.repo.Create(ctx, s.db, asset); err != nil {
		return nil, UploadParams{}, err
	}
	return asset, params, nil
}

// Confirm xác nhận ảnh đã upload: gọi Cloudinary kiểm chứng + lấy metadata, rồi đánh dấu USABLE.
// Chỉ CHỦ ảnh được confirm. Idempotent: ảnh đã USABLE thì trả lại luôn.
func (s *Service) Confirm(ctx context.Context, ownerID, mediaID int64) (*Asset, error) {
	asset, err := s.repo.GetByID(ctx, s.db, mediaID)
	if err != nil {
		return nil, err // ErrNotFound -> 404
	}
	if asset.OwnerID != ownerID {
		return nil, apperr.Forbidden("bạn chỉ có thể xác nhận ảnh của chính mình")
	}
	if asset.Status == StatusUsable {
		return asset, nil
	}
	if asset.Status != StatusPending {
		return nil, apperr.Conflict("ảnh không ở trạng thái chờ xác nhận")
	}

	// Nguồn sự thật là Cloudinary, KHÔNG tin client: server tự lấy metadata thật.
	info, err := s.storage.FetchResource(ctx, asset.StorageKey)
	if err != nil {
		return nil, err
	}

	mime := "image/" + info.Format
	if err := s.repo.MarkUsable(ctx, s.db, mediaID, ownerID, mime, info.Width, info.Height, info.Bytes); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, s.db, mediaID)
}

// AssetURL dựng URL công khai của một ảnh.
func (s *Service) AssetURL(a *Asset) string {
	return s.storage.PublicURL(a.StorageKey)
}

// randomName sinh tên ảnh ngẫu nhiên (16 byte hex) làm public_id.
func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random name: %w", err)
	}
	return hex.EncodeToString(b), nil
}
