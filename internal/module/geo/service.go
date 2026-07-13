package geo

import (
	"context"
	"database/sql"
	"log/slog"
)

// Service — nghiệp vụ module geo. Hiện chỉ là các thao tác đọc thuần nên service mỏng;
// giữ đúng khuôn 3 tầng để nhất quán và dễ mở rộng (vd cache danh sách tỉnh sau này).
type Service struct {
	db   *sql.DB
	repo Repository
	log  *slog.Logger
}

func NewService(db *sql.DB, repo Repository, log *slog.Logger) *Service {
	return &Service{db: db, repo: repo, log: log}
}

func (s *Service) ListProvinces(ctx context.Context) ([]Province, error) {
	return s.repo.ListProvinces(ctx, s.db)
}

func (s *Service) ListDistricts(ctx context.Context, provinceID int64) ([]District, error) {
	return s.repo.ListDistrictsByProvince(ctx, s.db, provinceID)
}

func (s *Service) ListWards(ctx context.Context, districtID int64) ([]Ward, error) {
	return s.repo.ListWardsByDistrict(ctx, s.db, districtID)
}
