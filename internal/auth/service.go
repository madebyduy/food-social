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
