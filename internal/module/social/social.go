package social

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/middleware"
)

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

type Comment struct {
	ID        int64  `json:"id"`
	PostID    int64  `json:"post_id"`
	UserID    int64  `json:"user_id"`
	ParentID  *int64 `json:"parent_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}
type commentRequest struct {
	Content  string `json:"content"`
	ParentID *int64 `json:"parent_id"`
}

func notify(tx *sql.Tx, ctx context.Context, userID, actorID int64, typ, entityType string, entityID int64) error {
	if userID == actorID {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,actor_id,type,entity_type,entity_id) VALUES($1,$2,$3,$4,$5)`, userID, actorID, typ, entityType, entityID)
	return err
}

func (s *Service) AddComment(ctx context.Context, userID, postID int64, req commentRequest) (*Comment, error) {
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" || utf8.RuneCountInString(req.Content) > 2000 {
		return nil, apperr.BadRequest("bình luận phải dài từ 1 đến 2000 ký tự")
	}
	if req.ParentID != nil && *req.ParentID <= 0 {
		return nil, apperr.BadRequest("parent_id không hợp lệ")
	}
	var c Comment
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var postExists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id=$1 AND status='VISIBLE' AND deleted_at IS NULL)`, postID).Scan(&postExists); err != nil {
			return err
		}
		if !postExists {
			return apperr.ErrNotFound
		}
		if req.ParentID != nil {
			var parentPost int64
			if err := tx.QueryRowContext(ctx, `SELECT post_id FROM comments WHERE id=$1 AND deleted_at IS NULL`, *req.ParentID).Scan(&parentPost); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return apperr.NotFound("không tìm thấy bình luận cha")
				}
				return err
			}
			if parentPost != postID {
				return apperr.BadRequest("bình luận cha không thuộc bài viết")
			}
		}
		if err := tx.QueryRowContext(ctx, `INSERT INTO comments(post_id,user_id,parent_id,content) VALUES($1,$2,$3,$4) RETURNING id,post_id,user_id,parent_id,content,created_at`, postID, userID, req.ParentID, req.Content).Scan(&c.ID, &c.PostID, &c.UserID, &c.ParentID, &c.Content, &c.CreatedAt); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE posts SET comment_count=comment_count+1, updated_at=now() WHERE id=$1`, postID)
		if err != nil {
			return err
		}
		var ownerID int64
		if err := tx.QueryRowContext(ctx, `SELECT user_id FROM posts WHERE id=$1`, postID).Scan(&ownerID); err != nil {
			return err
		}
		return notify(tx, ctx, ownerID, userID, "COMMENT", "POST", postID)
	})
	if err != nil {
		if _, ok := err.(*apperr.AppError); ok {
			return nil, err
		}
		return nil, apperr.Internal(err)
	}
	return &c, nil
}

func (s *Service) ListComments(ctx context.Context, postID int64, limit int) ([]Comment, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,post_id,user_id,parent_id,content,created_at::text FROM comments WHERE post_id=$1 AND deleted_at IS NULL ORDER BY created_at ASC,id ASC LIMIT $2`, postID, limit)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	items := make([]Comment, 0, limit)
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.ParentID, &c.Content, &c.CreatedAt); err != nil {
			return nil, apperr.Internal(err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (s *Service) toggle(ctx context.Context, userID, postID int64, table, counter string, add bool) error {
	if table != "post_likes" && table != "saved_posts" {
		return apperr.BadRequest("tác vụ không hợp lệ")
	}
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id=$1 AND deleted_at IS NULL)`, postID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return apperr.ErrNotFound
		}
		if add {
			res, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s(post_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, table), postID, userID)
			if err != nil {
				return err
			}
			n, _ := res.RowsAffected()
			_, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE posts SET %s=%s+1 WHERE id=$1 AND %s < (SELECT COUNT(*) FROM %s WHERE post_id=$1)`, counter, counter, counter, table), postID)
			if err != nil {
				return err
			}
			if n == 1 && table == "post_likes" {
				var ownerID int64
				if err := tx.QueryRowContext(ctx, `SELECT user_id FROM posts WHERE id=$1`, postID).Scan(&ownerID); err != nil {
					return err
				}
				return notify(tx, ctx, ownerID, userID, "LIKE", "POST", postID)
			}
			return nil
		}
		res, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE post_id=$1 AND user_id=$2`, table), postID, userID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			_, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE posts SET %s=GREATEST(%s-1,0) WHERE id=$1`, counter, counter), postID)
		}
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

func (s *Service) Follow(ctx context.Context, follower, following int64, add bool) error {
	if follower == following {
		return apperr.BadRequest("không thể tự theo dõi chính mình")
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND status <> 'BANNED')`, following).Scan(&exists); err != nil {
		return apperr.Internal(err)
	}
	if !exists {
		return apperr.ErrNotFound
	}
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if add {
			res, err := tx.ExecContext(ctx, `INSERT INTO follows(follower_id,following_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, follower, following)
			if err != nil {
				return err
			}
			n, _ := res.RowsAffected()
			if n == 1 {
				if _, err = tx.ExecContext(ctx, `UPDATE users SET following_count=following_count+1 WHERE id=$1`, follower); err != nil {
					return err
				}
				_, err = tx.ExecContext(ctx, `UPDATE users SET follower_count=follower_count+1 WHERE id=$1`, following)
				if err != nil {
					return err
				}
				return notify(tx, ctx, following, follower, "FOLLOW", "USER", following)
			}
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM follows WHERE follower_id=$1 AND following_id=$2`, follower, following)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 1 {
			if _, err = tx.ExecContext(ctx, `UPDATE users SET following_count=GREATEST(following_count-1,0) WHERE id=$1`, follower); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE users SET follower_count=GREATEST(follower_count-1,0) WHERE id=$1`, following)
		}
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

type Handler struct{ svc *Service }

func (s *Service) SavedPostIDs(ctx context.Context, userID int64, limit int) ([]int64, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT post_id FROM saved_posts WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func (s *Service) FollowingPostIDs(ctx context.Context, userID int64, limit int) ([]int64, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.id FROM posts p JOIN follows f ON f.following_id=p.user_id WHERE f.follower_id=$1 AND p.status='VISIBLE' AND p.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM blocks b WHERE b.blocker_id=$1 AND b.blocked_id=p.user_id) AND NOT EXISTS(SELECT 1 FROM mutes m WHERE m.muter_id=$1 AND m.muted_id=p.user_id) ORDER BY p.created_at DESC,p.id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	out := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, apperr.Internal(err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func (s *Service) SetRelation(ctx context.Context, actor, target int64, table string, add bool) error {
	if actor == target {
		return apperr.BadRequest("không thể áp dụng cho chính mình")
	}
	if table != "blocks" && table != "mutes" {
		return apperr.BadRequest("quan hệ không hợp lệ")
	}
	var err error
	if add {
		_, err = s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s(blocker_id,blocked_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, table), actor, target)
	} else {
		_, err = s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE blocker_id=$1 AND blocked_id=$2`, table), actor, target)
	}
	if err != nil {
		return apperr.Internal(err)
	}
	if table == "blocks" && add {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM follows WHERE (follower_id=$1 AND following_id=$2) OR (follower_id=$2 AND following_id=$1)`, actor, target)
	}
	return nil
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }
func actor(r *http.Request) (int64, error) {
	id, ok := middleware.UserID(r.Context())
	if !ok {
		return 0, apperr.Unauthorized("bạn cần đăng nhập")
	}
	return id, nil
}
func (h *Handler) AddComment(w http.ResponseWriter, r *http.Request) {
	userID, err := actor(r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	id, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	var req commentRequest
	if err = httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	c, err := h.svc.AddComment(r.Context(), userID, id, req)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, c)
}
func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	items, err := h.svc.ListComments(r.Context(), id, httpx.ParseLimit(r.URL.Query().Get("limit")))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.List(w, items, len(items), "")
}
func (h *Handler) Like(w http.ResponseWriter, r *http.Request) {
	h.toggle(w, r, "post_likes", "like_count", true)
}
func (h *Handler) Unlike(w http.ResponseWriter, r *http.Request) {
	h.toggle(w, r, "post_likes", "like_count", false)
}
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	h.toggle(w, r, "saved_posts", "save_count", true)
}
func (h *Handler) Unsave(w http.ResponseWriter, r *http.Request) {
	h.toggle(w, r, "saved_posts", "save_count", false)
}
func (h *Handler) toggle(w http.ResponseWriter, r *http.Request, table, counter string, add bool) {
	u, e := actor(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	id, e := httpx.PathInt64(r, "id")
	if e != nil {
		httpx.Error(w, e)
		return
	}
	if e = h.svc.toggle(r.Context(), u, id, table, counter, add); e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.NoContent(w)
}
func (h *Handler) Follow(w http.ResponseWriter, r *http.Request)   { h.follow(w, r, true) }
func (h *Handler) Unfollow(w http.ResponseWriter, r *http.Request) { h.follow(w, r, false) }
func (h *Handler) Saved(w http.ResponseWriter, r *http.Request) {
	id, e := actor(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	items, e := h.svc.SavedPostIDs(r.Context(), id, httpx.ParseLimit(r.URL.Query().Get("limit")))
	if e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.List(w, items, len(items), "")
}
func (h *Handler) FollowingFeed(w http.ResponseWriter, r *http.Request) {
	id, e := actor(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	items, e := h.svc.FollowingPostIDs(r.Context(), id, httpx.ParseLimit(r.URL.Query().Get("limit")))
	if e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.List(w, items, len(items), "")
}
func (h *Handler) Block(w http.ResponseWriter, r *http.Request)   { h.relation(w, r, "blocks", true) }
func (h *Handler) Unblock(w http.ResponseWriter, r *http.Request) { h.relation(w, r, "blocks", false) }
func (h *Handler) Mute(w http.ResponseWriter, r *http.Request)    { h.relation(w, r, "mutes", true) }
func (h *Handler) Unmute(w http.ResponseWriter, r *http.Request)  { h.relation(w, r, "mutes", false) }
func (h *Handler) relation(w http.ResponseWriter, r *http.Request, table string, add bool) {
	id, e := actor(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	target, e := httpx.PathInt64(r, "id")
	if e != nil {
		httpx.Error(w, e)
		return
	}
	if e = h.svc.SetRelation(r.Context(), id, target, table, add); e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.NoContent(w)
}
func (h *Handler) follow(w http.ResponseWriter, r *http.Request, add bool) {
	u, e := actor(r)
	if e != nil {
		httpx.Error(w, e)
		return
	}
	id, e := httpx.PathInt64(r, "id")
	if e != nil {
		httpx.Error(w, e)
		return
	}
	if e = h.svc.Follow(r.Context(), u, id, add); e != nil {
		httpx.Error(w, e)
		return
	}
	httpx.NoContent(w)
}
