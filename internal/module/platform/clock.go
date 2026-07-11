// Package platform chứa các thứ HẠ TẦNG kỹ thuật dùng chung, không thuộc nghiệp vụ
// của riêng module nào: đồng hồ (Clock), rate limiter, id generator...
package platform

import "time"

// Clock là interface bọc "lấy thời gian hiện tại".
//
// Vì sao không gọi thẳng time.Now()? Để TEST ĐƯỢC logic phụ thuộc thời gian mà không
// phải sleep: trong test ta truyền một Clock giả trả về mốc thời gian cố định/điều khiển
// được (xem ratelimiter_test.go). Code production dùng SystemClock.
type Clock interface {
	Now() time.Time
}

// SystemClock là Clock thật, luôn trả thời gian hệ thống theo UTC.
// Luôn .UTC() để đồng bộ với cột TIMESTAMPTZ (triệt tiêu cả lớp bug lệch múi giờ).
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
