package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
)

type Repository interface {
	// Create chèn một media_assets PENDING (mới presign), đọc ngược id + trạng thái + created_at.
	Create(ctx context.Context, q database.Querier, a *Asset) error

	// GetByID lấy một asset theo id. Không có -> ErrNotFound.
	GetByID(ctx context.Context, q database.Querier, id int64) (*Asset, error)

	// MarkUsable chuyển PENDING -> USABLE và ghi metadata thật (mime/width/height/size).
	// Chỉ đổi khi asset thuộc ownerID và đang PENDING. Không khớp -> ErrNotFound.
	MarkUsable(ctx context.Context, q database.Querier, id, ownerID int64, mime string, width, height int, sizeBytes int64) error
}

type repository struct{}

func NewRepository(_ *sql.DB) Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, q database.Querier, a *Asset) error {
	const query = `
		INSERT INTO media_assets (owner_id, storage_key, status)
		VALUES ($1, $2, 'PENDING')
		RETURNING id, status, created_at`

	err := q.QueryRowContext(ctx, query, a.OwnerID, a.StorageKey).
		Scan(&a.ID, &a.Status, &a.CreatedAt)
	if err != nil {
		return fmt.Errorf("media.Create insert: %w", err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, q database.Querier, id int64) (*Asset, error) {
	const query = `
		SELECT id, owner_id, storage_key, mime_type, size_bytes, width, height,
		       has_gps, status, created_at, confirmed_at
		FROM media_assets
		WHERE id = $1`

	var a Asset
	err := q.QueryRowContext(ctx, query, id).Scan(
		&a.ID, &a.OwnerID, &a.StorageKey, &a.MimeType, &a.SizeBytes, &a.Width, &a.Height,
		&a.HasGPS, &a.Status, &a.CreatedAt, &a.ConfirmedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("media.GetByID scan: %w", err)
	}
	return &a, nil
}

func (r *repository) MarkUsable(ctx context.Context, q database.Querier, id, ownerID int64, mime string, width, height int, sizeBytes int64) error {
	const query = `
		UPDATE media_assets
		SET status = 'USABLE', mime_type = $1, width = $2, height = $3, size_bytes = $4,
		    confirmed_at = now()
		WHERE id = $5 AND owner_id = $6 AND status = 'PENDING'`

	res, err := q.ExecContext(ctx, query, mime, width, height, sizeBytes, id, ownerID)
	if err != nil {
		return fmt.Errorf("media.MarkUsable exec: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("media.MarkUsable rows: %w", err)
	}
	if n == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
