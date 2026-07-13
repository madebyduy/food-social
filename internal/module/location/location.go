package location

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/middleware"
)

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

type Request struct {
	PlaceID int64  `json:"place_id"`
	Note    string `json:"note"`
}
type Suggestion struct {
	ID          int64  `json:"id"`
	PostID      int64  `json:"post_id"`
	SuggestedBy int64  `json:"suggested_by"`
	PlaceID     int64  `json:"place_id"`
	Status      string `json:"status"`
	Note        string `json:"note,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func (s *Service) Suggest(ctx context.Context, userID, postID int64, q Request) (*Suggestion, error) {
	if q.PlaceID <= 0 {
		return nil, apperr.BadRequest("place_id không hợp lệ")
	}
	q.Note = strings.TrimSpace(q.Note)
	if len(q.Note) > 500 {
		return nil, apperr.BadRequest("note tối đa 500 ký tự")
	}
	var out Suggestion
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var owner int64
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT user_id,status FROM posts WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, postID).Scan(&owner, &status); err != nil {
			return apperr.ErrNotFound
		}
		if owner == userID {
			return apperr.BadRequest("tác giả không thể tự đề xuất địa điểm")
		}
		if status != "VISIBLE" {
			return apperr.Conflict("bài viết không còn nhận đề xuất")
		}
		var ok bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM places WHERE id=$1 AND status='ACTIVE')`, q.PlaceID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return apperr.NotFound("không tìm thấy địa điểm")
		}
		return tx.QueryRowContext(ctx, `INSERT INTO location_suggestions(post_id,suggested_by,place_id,note) VALUES($1,$2,$3,$4) ON CONFLICT(post_id,suggested_by,place_id) WHERE status='PENDING' DO UPDATE SET note=EXCLUDED.note RETURNING id,post_id,suggested_by,place_id,status,COALESCE(note,''),created_at::text`, postID, userID, q.PlaceID, q.Note).Scan(&out.ID, &out.PostID, &out.SuggestedBy, &out.PlaceID, &out.Status, &out.Note, &out.CreatedAt)
	})
	if err != nil {
		if _, ok := err.(*apperr.AppError); ok {
			return nil, err
		}
		return nil, apperr.Internal(err)
	}
	return &out, nil
}
func (s *Service) Resolve(ctx context.Context, userID, suggestionID int64, accept bool) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var postID, owner, placeID int64
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT ls.post_id,p.user_id,ls.place_id,ls.status FROM location_suggestions ls JOIN posts p ON p.id=ls.post_id WHERE ls.id=$1 FOR UPDATE`, suggestionID).Scan(&postID, &owner, &placeID, &status); err != nil {
			return apperr.ErrNotFound
		}
		if owner != userID {
			return apperr.Forbidden("chỉ tác giả bài viết được duyệt đề xuất")
		}
		if status != "PENDING" {
			return apperr.Conflict("đề xuất đã được xử lý")
		}
		if !accept {
			_, err := tx.ExecContext(ctx, `UPDATE location_suggestions SET status='REJECTED',resolved_at=now() WHERE id=$1`, suggestionID)
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE posts SET place_id=$1,location_status='CONFIRMED',province_id=NULL,updated_at=now() WHERE id=$2`, placeID, postID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE places SET post_count=post_count+1,updated_at=now() WHERE id=$1`, placeID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE location_suggestions SET status=CASE WHEN id=$1 THEN 'ACCEPTED' ELSE 'REJECTED' END,resolved_at=now() WHERE post_id=$2 AND status='PENDING'`, suggestionID, postID); err != nil {
			return err
		}
		var proposer int64
		if err := tx.QueryRowContext(ctx, `SELECT suggested_by FROM location_suggestions WHERE id=$1`, suggestionID).Scan(&proposer); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,actor_id,type,entity_type,entity_id) VALUES($1,$2,'LOCATION_ACCEPTED','POST',$3)`, proposer, userID, postID)
		return err
	})
}

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }
func uid(r *http.Request) (int64, error) {
	id, ok := middleware.UserID(r.Context())
	if !ok {
		return 0, apperr.Unauthorized("bạn cần đăng nhập")
	}
	return id, nil
}
func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	u, e := uid(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	pid, e := httpx.PathInt64(r, "id")
	if e != nil {
		httpx.Error(w, e)
		return
	}
	var q Request
	if e = httpx.DecodeJSON(w, r, &q); e != nil {
		httpx.Error(w, e)
		return
	}
	v, e := h.svc.Suggest(r.Context(), u, pid, q)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.Created(w, v)
}
func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) { h.resolve(w, r, true) }
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) { h.resolve(w, r, false) }
func (h *Handler) resolve(w http.ResponseWriter, r *http.Request, accept bool) {
	u, e := uid(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	id, e := httpx.PathInt64(r, "id")
	if e != nil {
		httpx.Error(w, e)
		return
	}
	if e = h.svc.Resolve(r.Context(), u, id, accept); e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.NoContent(w)
}
