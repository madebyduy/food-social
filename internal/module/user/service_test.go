package user

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/database"
)

// service_test.go minh hoạ CÁCH TEST tầng service KHÔNG cần DB thật.
//
// Bí quyết: Service phụ thuộc interface Repository, nên ở test ta thay bằng một
// "mock" in-memory (mockRepo bên dưới). Test chạy trong vài mili-giây.

// mockRepo implement interface Repository bằng một map trong RAM.
type mockRepo struct {
	users     map[int64]*User
	updateErr error // cho phép ép lỗi để test nhánh thất bại
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: make(map[int64]*User)}
}

func (m *mockRepo) GetByID(_ context.Context, _ database.Querier, id int64) (*User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, apperr.ErrNotFound
	}
	// Trả bản sao để test không vô tình sửa trực tiếp state của mock.
	clone := *u
	return &clone, nil
}

func (m *mockRepo) UpdateProfile(_ context.Context, _ database.Querier, u *User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.users[u.ID]; !ok {
		return apperr.ErrNotFound
	}
	m.users[u.ID] = u
	return nil
}

// newTestService dựng Service với mock repo. db = nil vì mock bỏ qua Querier.
func newTestService(repo Repository) *Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService((*sql.DB)(nil), repo, logger)
}

func strPtr(s string) *string { return &s }

func TestService_UpdateProfile_ChiSuaDuocChinhMinh(t *testing.T) {
	repo := newMockRepo()
	repo.users[1] = &User{ID: 1, Username: "duy", DisplayName: "Duy"}
	svc := newTestService(repo)

	// actorID = 2 (người khác) sửa targetID = 1 -> phải bị Forbidden (403).
	_, err := svc.UpdateProfile(context.Background(), 2, 1, UpdateProfileRequest{
		DisplayName: strPtr("Hacker"),
	})

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperr.KindForbidden {
		t.Fatalf("mong đợi lỗi Forbidden, nhận: %v", err)
	}
	// Đảm bảo dữ liệu KHÔNG bị đổi.
	if got := repo.users[1].DisplayName; got != "Duy" {
		t.Fatalf("display_name bị đổi ngoài ý muốn: %q", got)
	}
}

func TestService_UpdateProfile_KhongTonTai(t *testing.T) {
	svc := newTestService(newMockRepo())

	_, err := svc.UpdateProfile(context.Background(), 99, 99, UpdateProfileRequest{
		DisplayName: strPtr("Ai đó"),
	})

	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("mong đợi ErrNotFound, nhận: %v", err)
	}
}

func TestService_UpdateProfile_ThanhCong(t *testing.T) {
	repo := newMockRepo()
	repo.users[1] = &User{ID: 1, Username: "duy", DisplayName: "Duy"}
	svc := newTestService(repo)

	// Chỉ gửi display_name và bio; avatar_url = nil nên phải GIỮ NGUYÊN.
	updated, err := svc.UpdateProfile(context.Background(), 1, 1, UpdateProfileRequest{
		DisplayName: strPtr("Duy Trần"),
		Bio:         strPtr("Mê ăn ốc"),
	})
	if err != nil {
		t.Fatalf("không mong đợi lỗi: %v", err)
	}
	if updated.DisplayName != "Duy Trần" {
		t.Errorf("display_name = %q, muốn %q", updated.DisplayName, "Duy Trần")
	}
	if !updated.Bio.Valid || updated.Bio.String != "Mê ăn ốc" {
		t.Errorf("bio = %+v, muốn 'Mê ăn ốc'", updated.Bio)
	}
}

func TestUpdateProfileRequest_Validate(t *testing.T) {
	longName := make([]rune, 101)
	for i := range longName {
		longName[i] = 'a'
	}

	tests := []struct {
		name    string
		req     UpdateProfileRequest
		wantErr bool
	}{
		{"tất cả nil (không sửa gì) -> hợp lệ", UpdateProfileRequest{}, false},
		{"display_name hợp lệ", UpdateProfileRequest{DisplayName: strPtr("Duy")}, false},
		{"display_name rỗng -> lỗi", UpdateProfileRequest{DisplayName: strPtr("   ")}, true},
		{"display_name quá dài -> lỗi", UpdateProfileRequest{DisplayName: strPtr(string(longName))}, true},
		{"bio rỗng cho phép (xóa bio)", UpdateProfileRequest{Bio: strPtr("")}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("mong đợi lỗi, nhưng nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("không mong đợi lỗi, nhận: %v", err)
			}
		})
	}
}
