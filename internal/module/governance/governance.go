package governance

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/middleware"
	trustvote "github.com/madebyduy/food-social/internal/module/vote"
)

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

type Notification struct {
	ID         int64          `json:"id"`
	Type       string         `json:"type"`
	ActorID    *int64         `json:"actor_id"`
	EntityType *string        `json:"entity_type"`
	EntityID   *int64         `json:"entity_id"`
	Data       map[string]any `json:"data"`
	ReadAt     *string        `json:"read_at"`
	CreatedAt  string         `json:"created_at"`
}
type ReportRequest struct {
	TargetType string `json:"target_type"`
	TargetID   int64  `json:"target_id"`
	Reason     string `json:"reason"`
	Detail     string `json:"detail"`
}
type VoteRequest struct {
	Vote string `json:"vote"`
}

func (s *Service) AddReport(ctx context.Context, userID int64, req ReportRequest) error {
	req.TargetType = strings.ToUpper(strings.TrimSpace(req.TargetType))
	req.Reason = strings.ToUpper(strings.TrimSpace(req.Reason))
	req.Detail = strings.TrimSpace(req.Detail)
	if req.TargetID <= 0 || (req.TargetType != "POST" && req.TargetType != "COMMENT" && req.TargetType != "USER") {
		return apperr.BadRequest("target_type hoặc target_id không hợp lệ")
	}
	if req.Reason == "" || utf8.RuneCountInString(req.Reason) > 40 || utf8.RuneCountInString(req.Detail) > 1000 {
		return apperr.BadRequest("reason/detail không hợp lệ")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO reports(reporter_id,target_type,target_id,reason,detail) VALUES($1,$2,$3,$4,$5) ON CONFLICT (reporter_id,target_type,target_id) WHERE status='OPEN' DO NOTHING`, userID, req.TargetType, req.TargetID, req.Reason, req.Detail)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (s *Service) Vote(ctx context.Context, userID, postID int64, vote string) error {
	vote = strings.ToUpper(strings.TrimSpace(vote))
	if vote != "TRUSTED" && vote != "UNTRUSTED" {
		return apperr.BadRequest("vote phải là TRUSTED hoặc UNTRUSTED")
	}
	return s.voteTx(ctx, userID, postID, &vote)
}
func (s *Service) RemoveVote(ctx context.Context, userID, postID int64) error {
	return s.voteTx(ctx, userID, postID, nil)
}
func (s *Service) voteTx(ctx context.Context, userID, postID int64, vote *string) error {
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var ownerID int64
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT user_id,status FROM posts WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, postID).Scan(&ownerID, &status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apperr.ErrNotFound
			}
			return err
		}
		if ownerID == userID {
			return apperr.Forbidden("không thể tự vote bài viết của mình")
		}
		if vote != nil && status != "VISIBLE" {
			return apperr.Conflict("bài viết không còn nhận vote")
		}
		var created time.Time
		if err := tx.QueryRowContext(ctx, `SELECT created_at FROM users WHERE id=$1`, userID).Scan(&created); err != nil {
			return err
		}
		weight := trustvote.CalculateWeight(created, time.Now().UTC())
		if vote == nil {
			_, err := tx.ExecContext(ctx, `DELETE FROM review_votes WHERE post_id=$1 AND user_id=$2`, postID, userID)
			if err != nil {
				return err
			}
		} else {
			_, err := tx.ExecContext(ctx, `INSERT INTO review_votes(post_id,user_id,vote,weight_at_vote) VALUES($1,$2,$3,$4) ON CONFLICT(post_id,user_id) DO UPDATE SET vote=EXCLUDED.vote,weight_at_vote=EXCLUDED.weight_at_vote,updated_at=now()`, postID, userID, *vote, weight)
			if err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `UPDATE posts p SET trusted_weight=COALESCE((SELECT SUM(weight_at_vote) FROM review_votes WHERE post_id=p.id AND vote='TRUSTED'),0), untrusted_weight=COALESCE((SELECT SUM(weight_at_vote) FROM review_votes WHERE post_id=p.id AND vote='UNTRUSTED'),0), total_vote_count=(SELECT COUNT(*) FROM review_votes WHERE post_id=p.id), untrusted_ratio=COALESCE((SELECT SUM(weight_at_vote) FROM review_votes WHERE post_id=p.id AND vote='UNTRUSTED')/NULLIF((SELECT SUM(weight_at_vote) FROM review_votes WHERE post_id=p.id),0),0), status=CASE WHEN (SELECT COUNT(*) FROM review_votes WHERE post_id=p.id)>=200 AND COALESCE((SELECT SUM(weight_at_vote) FROM review_votes WHERE post_id=p.id AND vote='UNTRUSTED')/NULLIF((SELECT SUM(weight_at_vote) FROM review_votes WHERE post_id=p.id),0),0)>0.70 THEN 'HIDDEN_BY_COMMUNITY' ELSE p.status END, hidden_at=CASE WHEN (SELECT COUNT(*) FROM review_votes WHERE post_id=p.id)>=200 AND COALESCE((SELECT SUM(weight_at_vote) FROM review_votes WHERE post_id=p.id AND vote='UNTRUSTED')/NULLIF((SELECT SUM(weight_at_vote) FROM review_votes WHERE post_id=p.id),0),0)>0.70 THEN COALESCE(p.hidden_at,now()) ELSE p.hidden_at END, updated_at=now() WHERE p.id=$1`, postID)
		return err
	})
	if err != nil {
		if _, ok := err.(*apperr.AppError); ok {
			return err
		}
		return apperr.Internal(err)
	}
	return nil
}

func (s *Service) Notifications(ctx context.Context, userID int64, limit int) ([]Notification, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,type,actor_id,entity_type,entity_id,data,read_at::text,created_at::text FROM notifications WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]Notification, 0, limit)
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.ActorID, &n.EntityType, &n.EntityID, &n.Data, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
func (s *Service) MarkNotificationRead(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE notifications SET read_at=COALESCE(read_at,now()) WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return apperr.Internal(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
func (s *Service) IsAdmin(ctx context.Context, userID int64) bool {
	var ok bool
	_ = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='ADMIN' AND status='ACTIVE')`, userID).Scan(&ok)
	return ok
}
func (s *Service) ModeratePost(ctx context.Context, adminID, postID int64, restore bool, reason string) error {
	if !s.IsAdmin(ctx, adminID) {
		return apperr.Forbidden("chỉ admin mới được thực hiện thao tác này")
	}
	status := "HIDDEN_BY_ADMIN"
	action := "HIDE_POST"
	if restore {
		status = "VISIBLE"
		action = "RESTORE_POST"
	}
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE posts SET status=$1,hidden_at=CASE WHEN $1='VISIBLE' THEN NULL ELSE COALESCE(hidden_at,now()) END,updated_at=now() WHERE id=$2 AND deleted_at IS NULL`, status, postID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return apperr.ErrNotFound
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO admin_actions(admin_id,action,target_type,target_id,reason) VALUES($1,$2,'POST',$3,$4)`, adminID, action, postID, reason)
		return err
	})
}

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }
func uid(r *http.Request) (int64, error) {
	id, ok := middleware.UserID(r.Context())
	if !ok {
		return 0, apperr.Unauthorized("bạn cần đăng nhập")
	}
	return id, nil
}
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	id, e := uid(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	var q ReportRequest
	if e = httpx.DecodeJSON(w, r, &q); e != nil {
		httpx.Error(w, e)
		return
	}
	if e = h.svc.AddReport(r.Context(), id, q); e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.NoContent(w)
}
func (h *Handler) Vote(w http.ResponseWriter, r *http.Request) {
	id, e := uid(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	postID, e := httpx.PathInt64(r, "id")
	if e != nil {
		httpx.Error(w, e)
		return
	}
	var q VoteRequest
	if e = httpx.DecodeJSON(w, r, &q); e != nil {
		httpx.Error(w, e)
		return
	}
	if e = h.svc.Vote(r.Context(), id, postID, q.Vote); e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.NoContent(w)
}
func (h *Handler) RemoveVote(w http.ResponseWriter, r *http.Request) {
	id, e := uid(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	postID, e := httpx.PathInt64(r, "id")
	if e != nil {
		httpx.Error(w, e)
		return
	}
	if e = h.svc.RemoveVote(r.Context(), id, postID); e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.NoContent(w)
}
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	id, e := uid(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	items, e := h.svc.Notifications(r.Context(), id, httpx.ParseLimit(r.URL.Query().Get("limit")))
	if e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.List(w, items, len(items), "")
}
func (h *Handler) ReadNotification(w http.ResponseWriter, r *http.Request) {
	id, e := uid(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	nid, e := httpx.PathInt64(r, "id")
	if e != nil {
		httpx.Error(w, e)
		return
	}
	if e = h.svc.MarkNotificationRead(r.Context(), id, nid); e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.NoContent(w)
}
func (h *Handler) HidePost(w http.ResponseWriter, r *http.Request)    { h.moderate(w, r, false) }
func (h *Handler) RestorePost(w http.ResponseWriter, r *http.Request) { h.moderate(w, r, true) }
func (h *Handler) moderate(w http.ResponseWriter, r *http.Request, restore bool) {
	id, e := uid(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	pid, e := httpx.PathInt64(r, "id")
	if e != nil {
		httpx.Error(w, e)
		return
	}
	if e = h.svc.ModeratePost(r.Context(), id, pid, restore, "admin moderation"); e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.NoContent(w)
}
