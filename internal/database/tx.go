package database

import (
	"context"
	"database/sql"
	"fmt"
)

// Querier là interface tối thiểu để chạy câu lệnh SQL.
//
// ĐIỂM MẤU CHỐT (mid-level): cả *sql.DB VÀ *sql.Tx đều đã thỏa interface này sẵn.
// Vì repository nhận Querier (không giữ *sql.DB bên trong), nên CÙNG MỘT hàm repo
// chạy được cả trong lẫn ngoài transaction — service là bên quyết định truyền gì:
//   - Thao tác đọc đơn giản: truyền thẳng *sql.DB.
//   - Thao tác nhiều bước cần all-or-nothing: mở tx bằng WithTx, truyền *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// WithTx mở một transaction, chạy fn, rồi TỰ ĐỘNG:
//   - rollback nếu fn trả lỗi hoặc panic,
//   - commit nếu fn trả nil.
//
// Nhờ vậy service chỉ cần viết logic trong fn, không phải nhớ commit/rollback thủ công
// (nguồn bug kinh điển: quên rollback -> giữ khóa row mãi mãi).
//
// Module user hiện chỉ có thao tác đơn (đọc/ghi 1 bảng) nên chưa cần WithTx, nhưng
// các module sau (vote, follow, accept suggestion...) sẽ dùng liên tục — để sẵn ở đây.
func WithTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // ném lại để middleware Recovery bắt và trả 500
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback: %v (lỗi gốc: %w)", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}
