package auth

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type mockRepo struct {
	usersByLogin  map[string]*User
	usersByID     map[int64]*User
	sessions      map[string]int64
	createdTokens []CreateSessionInput
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		usersByLogin: make(map[string]*User),
		usersByID:    make(map[int64]*User),
		sessions:     make(map[string]int64),
	}
}

func (m *mockRepo) CreateUser(_ context.Context, _ database.Querier, input CreateUserInput) (*User, error) {
	u := &User{
		ID:           int64(len(m.usersByID) + 1),
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: input.PasswordHash,
		DisplayName:  input.DisplayName,
		Role:         RoleUser,
		Status:       StatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.usersByLogin[input.Username] = u
	m.usersByLogin[input.Email] = u
	m.usersByID[u.ID] = u
	return u, nil
}

func (m *mockRepo) GetUserByLogin(_ context.Context, _ database.Querier, login string) (*User, error) {
	u, ok := m.usersByLogin[login]
	if !ok {
		return nil, apperr.ErrNotFound
	}
	clone := *u
	return &clone, nil
}

func (m *mockRepo) GetUserByID(_ context.Context, _ database.Querier, id int64) (*User, error) {
	u, ok := m.usersByID[id]
	if !ok {
		return nil, apperr.ErrNotFound
	}
	clone := *u
	return &clone, nil
}

func (m *mockRepo) CreateSession(_ context.Context, _ database.Querier, input CreateSessionInput) error {
	m.sessions[input.TokenHash] = input.UserID
	m.createdTokens = append(m.createdTokens, input)
	return nil
}

func (m *mockRepo) RevokeSession(_ context.Context, _ database.Querier, tokenHash string) error {
	delete(m.sessions, tokenHash)
	return nil
}

func (m *mockRepo) GetSessionUserID(_ context.Context, _ database.Querier, tokenHash string, _ time.Time) (int64, error) {
	userID, ok := m.sessions[tokenHash]
	if !ok {
		return 0, apperr.ErrNotFound
	}
	return userID, nil
}

func newTestService(repo Repository) *Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService((*sql.DB)(nil), repo, time.Hour, fixedClock{now: time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)}, logger)
}

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := hashPassword("secret123")
	if err != nil {
		t.Fatalf("hashPassword error: %v", err)
	}
	if hash == "secret123" {
		t.Fatal("password hash must not equal raw password")
	}
	if !verifyPassword(hash, "secret123") {
		t.Fatal("expected password to verify")
	}
	if verifyPassword(hash, "wrong123") {
		t.Fatal("wrong password should not verify")
	}
}

func TestService_LoginSuccess(t *testing.T) {
	repo := newMockRepo()
	hash, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	repo.usersByLogin["duy"] = &User{
		ID:           1,
		Username:     "duy",
		Email:        "duy@example.com",
		PasswordHash: hash,
		DisplayName:  "Duy",
		Role:         RoleUser,
		Status:       StatusActive,
	}
	svc := newTestService(repo)

	res, err := svc.Login(context.Background(), LoginRequest{Login: "duy", Password: "secret123"})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if res.Token == "" {
		t.Fatal("expected session token")
	}
	if len(repo.createdTokens) != 1 {
		t.Fatalf("expected 1 session, got %d", len(repo.createdTokens))
	}
}

func TestService_LoginWrongPassword(t *testing.T) {
	repo := newMockRepo()
	hash, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	repo.usersByLogin["duy"] = &User{ID: 1, Username: "duy", PasswordHash: hash, Status: StatusActive}
	svc := newTestService(repo)

	_, err = svc.Login(context.Background(), LoginRequest{Login: "duy", Password: "wrong123"})
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperr.KindUnauthorized {
		t.Fatalf("expected Unauthorized, got %v", err)
	}
}

func TestService_LoginBannedUser(t *testing.T) {
	repo := newMockRepo()
	hash, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	repo.usersByLogin["duy"] = &User{ID: 1, Username: "duy", PasswordHash: hash, Status: StatusBanned}
	svc := newTestService(repo)

	_, err = svc.Login(context.Background(), LoginRequest{Login: "duy", Password: "secret123"})
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperr.KindForbidden {
		t.Fatalf("expected Forbidden, got %v", err)
	}
}

func TestService_ResolveSession(t *testing.T) {
	repo := newMockRepo()
	repo.sessions[hashToken("raw-token")] = 42
	svc := newTestService(repo)

	userID, err := svc.ResolveSession(context.Background(), "raw-token")
	if err != nil {
		t.Fatalf("ResolveSession error: %v", err)
	}
	if userID != 42 {
		t.Fatalf("userID = %d, want 42", userID)
	}
}

func TestRegisterRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{"valid", RegisterRequest{Username: "duy_123", Email: "duy@example.com", Password: "secret123"}, false},
		{"short username", RegisterRequest{Username: "du", Email: "duy@example.com", Password: "secret123"}, true},
		{"bad username", RegisterRequest{Username: "duy!", Email: "duy@example.com", Password: "secret123"}, true},
		{"bad email", RegisterRequest{Username: "duy", Email: "not-email", Password: "secret123"}, true},
		{"short password", RegisterRequest{Username: "duy", Email: "duy@example.com", Password: "short"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
