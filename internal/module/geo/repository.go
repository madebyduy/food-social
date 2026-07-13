package geo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/madebyduy/food-social/internal/database"
)

// Repository — thao tác đọc dữ liệu địa lý. Bảng provinces/districts/wards là dữ liệu tra cứu
// (seed sẵn, ít đổi) nên module này CHỈ có thao tác đọc, không create/update.
type Repository interface {
	ListProvinces(ctx context.Context, q database.Querier) ([]Province, error)
	ListDistrictsByProvince(ctx context.Context, q database.Querier, provinceID int64) ([]District, error)
	ListWardsByDistrict(ctx context.Context, q database.Querier, districtID int64) ([]Ward, error)
}

type repository struct{}

func NewRepository(_ *sql.DB) Repository {
	return &repository{}
}

func (r *repository) ListProvinces(ctx context.Context, q database.Querier) ([]Province, error) {
	const query = `SELECT id, name, slug, created_at FROM provinces ORDER BY name`

	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("geo.ListProvinces query: %w", err)
	}
	defer rows.Close()

	var out []Province
	for rows.Next() {
		var p Province
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("geo.ListProvinces scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *repository) ListDistrictsByProvince(ctx context.Context, q database.Querier, provinceID int64) ([]District, error) {
	const query = `
		SELECT id, province_id, name, slug, created_at
		FROM districts
		WHERE province_id = $1
		ORDER BY name`

	rows, err := q.QueryContext(ctx, query, provinceID)
	if err != nil {
		return nil, fmt.Errorf("geo.ListDistrictsByProvince query: %w", err)
	}
	defer rows.Close()

	var out []District
	for rows.Next() {
		var d District
		if err := rows.Scan(&d.ID, &d.ProvinceID, &d.Name, &d.Slug, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("geo.ListDistrictsByProvince scan: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *repository) ListWardsByDistrict(ctx context.Context, q database.Querier, districtID int64) ([]Ward, error) {
	const query = `
		SELECT id, district_id, name, slug, created_at
		FROM wards
		WHERE district_id = $1
		ORDER BY name`

	rows, err := q.QueryContext(ctx, query, districtID)
	if err != nil {
		return nil, fmt.Errorf("geo.ListWardsByDistrict query: %w", err)
	}
	defer rows.Close()

	var out []Ward
	for rows.Next() {
		var w Ward
		if err := rows.Scan(&w.ID, &w.DistrictID, &w.Name, &w.Slug, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("geo.ListWardsByDistrict scan: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
