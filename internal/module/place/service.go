package place

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/madebyduy/food-social/internal/apperr"
)

type Service struct {
	db   *sql.DB
	repo Repository
	log  *slog.Logger
}

func NewService(db *sql.DB, repo Repository, log *slog.Logger) *Service {
	return &Service{db: db, repo: repo, log: log}
}

// Create tạo một place mới (do user tạo, hậu kiểm — không duyệt trước).
//
// Chống trùng: nếu client gửi google_place_id mà place đó đã tồn tại thì TRẢ LẠI place cũ
// (idempotent) thay vì tạo bản sao — tránh nhiều place cho cùng một địa điểm Google.
func (s *Service) Create(ctx context.Context, req CreatePlaceRequest) (*Place, error) {
	if req.GooglePlaceID != nil {
		existing, err := s.repo.FindByGoogleID(ctx, s.db, *req.GooglePlaceID)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, apperr.ErrNotFound) {
			return nil, err
		}
	}

	if req.ProvinceID != nil {
		ok, err := s.repo.ProvinceExists(ctx, s.db, *req.ProvinceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apperr.BadRequest("province_id không tồn tại")
		}
	}

	p := &Place{
		Name:          strings.TrimSpace(req.Name),
		GooglePlaceID: nullString(req.GooglePlaceID),
		Address:       nullString(req.Address),
		ProvinceID:    nullInt64(req.ProvinceID),
		DistrictID:    nullInt64(req.DistrictID),
		WardID:        nullInt64(req.WardID),
		Latitude:      nullFloat64(req.Latitude),
		Longitude:     nullFloat64(req.Longitude),
	}
	if err := s.repo.Create(ctx, s.db, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Place, error) {
	return s.repo.GetByID(ctx, s.db, id)
}

func (s *Service) Search(ctx context.Context, term string, provinceID *int64, limit int) ([]*Place, error) {
	return s.repo.Search(ctx, s.db, strings.TrimSpace(term), provinceID, limit)
}

// --- helper: *T (từ request) -> sql.NullX (để ghi DB) ---

func nullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.TrimSpace(*p), Valid: true}
}

func nullInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func nullFloat64(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *p, Valid: true}
}
