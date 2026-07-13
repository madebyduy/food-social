package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
	"github.com/madebyduy/food-social/internal/module/platform"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]+$`)

type Service struct {
	db         *sql.DB
	repo       Repository
	sessionTTL time.Duration
	clock      platform.Clock
	log        *slog.Logger
}

func NewService(db *sql.DB, repo Repository, sessionTTL time.Duration, clock platform.Clock, log *slog.Logger) *Service {
	if clock == nil {
		clock = platform.SystemClock{}
	}
	return &Service{db: db, repo: repo, sessionTTL: sessionTTL, clock: clock, log: log}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	username := strings.ToLower(strings.TrimSpace(req.Username))
	email := strings.ToLower(strings.TrimSpace(req.Email))
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = username
	}

	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	token, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, fmt.Errorf("new session token: %w", err)
	}

	var created *User
	expiresAt := s.clock.Now().Add(s.sessionTTL)
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		u, err := s.repo.CreateUser(ctx, tx, CreateUserInput{
			Username:     username,
			Email:        email,
			PasswordHash: passwordHash,
			DisplayName:  displayName,
		})
		if err != nil {
			return err
		}
		if err := s.repo.CreateSession(ctx, tx, CreateSessionInput{
			UserID:    u.ID,
			TokenHash: tokenHash,
			ExpiresAt: expiresAt,
		}); err != nil {
			return err
		}
		created = u
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, ExpiresAt: expiresAt, User: toUserResponse(created)}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	u, err := s.repo.GetUserByLogin(ctx, s.db, strings.TrimSpace(req.Login))
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, apperr.Unauthorized("thông tin đăng nhập không đúng")
		}
		return nil, err
	}
	if !verifyPassword(u.PasswordHash, req.Password) {
		return nil, apperr.Unauthorized("thông tin đăng nhập không đúng")
	}
	if u.Status != StatusActive {
		return nil, apperr.Forbidden("tài khoản không được phép đăng nhập")
	}

	token, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, fmt.Errorf("new session token: %w", err)
	}

	expiresAt := s.clock.Now().Add(s.sessionTTL)
	if err := s.repo.CreateSession(ctx, s.db, CreateSessionInput{
		UserID:    u.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, ExpiresAt: expiresAt, User: toUserResponse(u)}, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return apperr.Unauthorized("thiếu session token")
	}
	return s.repo.RevokeSession(ctx, s.db, hashToken(rawToken))
}

func (s *Service) Me(ctx context.Context, userID int64) (*UserResponse, error) {
	u, err := s.repo.GetUserByID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	res := toUserResponse(u)
	return &res, nil
}

// RequestPasswordReset tạo token một lần. Việc gửi token ra ngoài được giao cho
// email/SMS adapter ở tầng triển khai; API luôn trả cùng một kết quả để không lộ email.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := validateEmail(email); err != nil {
		return err
	}
	var userID int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(email)=lower($1) AND status='ACTIVE' AND deleted_at IS NULL`, email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find reset user: %w", err)
	}
	token, hash, err := newSessionToken()
	if err != nil {
		return fmt.Errorf("new reset token: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO password_reset_tokens(user_id,token_hash,expires_at) VALUES($1,$2,$3)`, userID, hash, s.clock.Now().Add(30*time.Minute))
	if err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}
	// TODO(production): inject an email/SMS sender. Không log token thật.
	s.log.Info("password reset requested", "user_id", userID, "delivery_token_created", token != "")
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if strings.TrimSpace(token) == "" {
		return apperr.BadRequest("token không được để trống")
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var userID int64
		if err := tx.QueryRowContext(ctx, `SELECT user_id FROM password_reset_tokens WHERE token_hash=$1 AND used_at IS NULL AND expires_at>$2 FOR UPDATE`, hashToken(token), s.clock.Now()).Scan(&userID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apperr.BadRequest("token không hợp lệ hoặc đã hết hạn")
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash=$1,updated_at=now() WHERE id=$2 AND deleted_at IS NULL`, hash, userID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE password_reset_tokens SET used_at=now() WHERE token_hash=$1`, hashToken(token)); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
		return err
	})
}

func (s *Service) DeleteAccount(ctx context.Context, userID int64) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE users SET status='DELETED',deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, userID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return apperr.ErrNotFound
		}
		_, err = tx.ExecContext(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
		return err
	})
}

func (s *Service) ResolveSession(ctx context.Context, rawToken string) (int64, error) {
	if strings.TrimSpace(rawToken) == "" {
		return 0, apperr.Unauthorized("thiếu session token")
	}
	userID, err := s.repo.GetSessionUserID(ctx, s.db, hashToken(rawToken), s.clock.Now())
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return 0, apperr.Unauthorized("phiên đăng nhập không hợp lệ")
		}
		return 0, err
	}
	return userID, nil
}

func validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if l := utf8.RuneCountInString(username); l < 3 || l > 30 {
		return apperr.BadRequest("username phải từ 3 đến 30 ký tự")
	}
	if !usernamePattern.MatchString(strings.ToLower(username)) {
		return apperr.BadRequest("username chỉ gồm chữ thường, số và dấu gạch dưới")
	}
	return nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > 255 {
		return apperr.BadRequest("email không hợp lệ")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return apperr.BadRequest("email không hợp lệ")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return apperr.BadRequest("password phải có ít nhất 8 ký tự")
	}
	if len(password) > 72 {
		return apperr.BadRequest("password tối đa 72 byte")
	}
	return nil
}

func newSessionToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, hashToken(raw), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
