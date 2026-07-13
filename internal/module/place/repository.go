package place

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
)

const placeColumns = `
	id, canonical_place_id, google_place_id, name, address,
	province_id, district_id, ward_id, latitude, longitude,
	post_count, status, created_at, updated_at`

type Repository interface {
	Create(ctx context.Context, q database.Querier, p *Place) error
	GetByID(ctx context.Context, q database.Querier, id int64) (*Place, error)

	// FindByGoogleID tìm place theo google_place_id (chống tạo trùng). Không có -> ErrNotFound.
	FindByGoogleID(ctx context.Context, q database.Querier, googleID string) (*Place, error)

	// Search tìm place ACTIVE theo tên (full-text, bỏ dấu). term rỗng -> liệt kê theo độ phổ biến.
	// provinceID != nil -> lọc thêm theo tỉnh.
	Search(ctx context.Context, q database.Querier, term string, provinceID *int64, limit int) ([]*Place, error)

	// ProvinceExists dùng để validate province_id khi tạo place.
	ProvinceExists(ctx context.Context, q database.Querier, provinceID int64) (bool, error)
}

type repository struct{}

func NewRepository(_ *sql.DB) Repository {
	return &repository{}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPlace(sc rowScanner) (*Place, error) {
	var p Place
	err := sc.Scan(
		&p.ID, &p.CanonicalPlaceID, &p.GooglePlaceID, &p.Name, &p.Address,
		&p.ProvinceID, &p.DistrictID, &p.WardID, &p.Latitude, &p.Longitude,
		&p.PostCount, &p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) Create(ctx context.Context, q database.Querier, p *Place) error {
	const query = `
		INSERT INTO places (google_place_id, name, address, province_id, district_id, ward_id, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, post_count, status, created_at, updated_at`

	err := q.QueryRowContext(ctx, query,
		p.GooglePlaceID, p.Name, p.Address, p.ProvinceID, p.DistrictID, p.WardID, p.Latitude, p.Longitude,
	).Scan(&p.ID, &p.PostCount, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("place.Create insert: %w", err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, q database.Querier, id int64) (*Place, error) {
	query := `SELECT ` + placeColumns + ` FROM places WHERE id = $1`

	p, err := scanPlace(q.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("place.GetByID scan: %w", err)
	}
	return p, nil
}

func (r *repository) FindByGoogleID(ctx context.Context, q database.Querier, googleID string) (*Place, error) {
	query := `SELECT ` + placeColumns + ` FROM places WHERE google_place_id = $1`

	p, err := scanPlace(q.QueryRowContext(ctx, query, googleID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, fmt.Errorf("place.FindByGoogleID scan: %w", err)
	}
	return p, nil
}

func (r *repository) Search(ctx context.Context, q database.Querier, term string, provinceID *int64, limit int) ([]*Place, error) {
	// $2 lọc tỉnh tùy chọn: NULL -> không lọc. $3 = giới hạn số kết quả.
	if term == "" {
		// Không có từ khóa -> duyệt theo độ phổ biến (post_count).
		query := `SELECT ` + placeColumns + `
			FROM places
			WHERE status = 'ACTIVE'
			  AND ($1::bigint IS NULL OR province_id = $1)
			ORDER BY post_count DESC, id DESC
			LIMIT $2`
		return queryPlaces(ctx, q, query, provinceID, limit)
	}

	// Có từ khóa -> full-text 'simple' + unaccent (khớp search_vector generated của bảng).
	query := `SELECT ` + placeColumns + `
		FROM places
		WHERE status = 'ACTIVE'
		  AND ($2::bigint IS NULL OR province_id = $2)
		  AND search_vector @@ plainto_tsquery('simple', unaccent($1))
		ORDER BY post_count DESC, id DESC
		LIMIT $3`
	return queryPlaces(ctx, q, query, term, provinceID, limit)
}

func (r *repository) ProvinceExists(ctx context.Context, q database.Querier, provinceID int64) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM provinces WHERE id = $1)`

	var ok bool
	if err := q.QueryRowContext(ctx, query, provinceID).Scan(&ok); err != nil {
		return false, fmt.Errorf("place.ProvinceExists: %w", err)
	}
	return ok, nil
}

func queryPlaces(ctx context.Context, q database.Querier, query string, args ...any) ([]*Place, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("place list query: %w", err)
	}
	defer rows.Close()

	var out []*Place
	for rows.Next() {
		p, err := scanPlace(rows)
		if err != nil {
			return nil, fmt.Errorf("place list scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
