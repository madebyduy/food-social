package search

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
)

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

type Result struct {
	Type     string `json:"type"`
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
}

func (s *Service) Search(ctx context.Context, q, kind string, limit int) ([]Result, error) {
	q = strings.TrimSpace(q)
	kind = strings.ToLower(strings.TrimSpace(kind))
	if q == "" || len(q) > 100 {
		return nil, apperr.BadRequest("q phải từ 1 đến 100 ký tự")
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	if kind == "" {
		kind = "all"
	}
	if kind != "all" && kind != "posts" && kind != "users" && kind != "places" {
		return nil, apperr.BadRequest("type không hợp lệ")
	}
	out := make([]Result, 0, limit)
	add := func(query string, args ...any) error {
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var v Result
			if err := rows.Scan(&v.ID, &v.Title, &v.Subtitle); err != nil {
				return err
			}
			out = append(out, v)
		}
		return rows.Err()
	}
	if kind == "all" || kind == "posts" {
		err := add(`SELECT id,LEFT(content,140),'' FROM posts WHERE status='VISIBLE' AND deleted_at IS NULL AND search_vector @@ plainto_tsquery('simple',unaccent($1)) ORDER BY created_at DESC,id DESC LIMIT $2`, q, limit)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		for i := range out {
			out[i].Type = "post"
		}
	}
	if (kind == "all" || kind == "users") && len(out) < limit {
		start := len(out)
		err := add(`SELECT id,display_name,username FROM users WHERE status='ACTIVE' AND deleted_at IS NULL AND (lower(username) LIKE lower($1)||'%' OR lower(display_name) LIKE '%'||lower($1)||'%') ORDER BY follower_count DESC,id DESC LIMIT $2`, q, limit-len(out))
		if err != nil {
			return nil, apperr.Internal(err)
		}
		for i := start; i < len(out); i++ {
			out[i].Type = "user"
		}
	}
	if (kind == "all" || kind == "places") && len(out) < limit {
		start := len(out)
		err := add(`SELECT id,name,address FROM places WHERE status='ACTIVE' AND search_vector @@ plainto_tsquery('simple',unaccent($1)) ORDER BY post_count DESC,id DESC LIMIT $2`, q, limit-len(out))
		if err != nil {
			return nil, apperr.Internal(err)
		}
		for i := start; i < len(out); i++ {
			out[i].Type = "place"
		}
	}
	return out, nil
}

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Search(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("type"), httpx.ParseLimit(r.URL.Query().Get("limit")))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.List(w, items, len(items), "")
}
