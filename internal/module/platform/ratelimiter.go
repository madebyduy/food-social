package platform

import (
	"sync"
	"time"
)

// RateLimiter giới hạn số request bằng thuật toán TOKEN BUCKET (gáo token).
//
// Hình dung mỗi "key" (vd: mỗi IP) có một cái gáo:
//   - Gáo chứa tối đa `burst` token, ban đầu đầy.
//   - Mỗi request tiêu 1 token. Hết token -> bị chặn (429).
//   - Token tự nạp lại đều đặn `rate` token/giây (tối đa tới `burst`).
//
// Ưu điểm so với "đếm N request mỗi phút": cho phép "bùng" ngắn (burst) nhưng vẫn
// giữ trần trung bình dài hạn. Đây là bản IN-MEMORY cho MVP — khi scale nhiều instance
// thì thay bằng Redis (mỗi instance có gáo riêng sẽ không còn chính xác).
//
// An toàn cho goroutine (có sync.Mutex) vì HTTP server chạy mỗi request một goroutine.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // token nạp mỗi giây
	burst   float64 // sức chứa tối đa của gáo
	ttl     time.Duration
	clock   Clock
}

type bucket struct {
	tokens float64   // số token còn lại (số thực để nạp mượt)
	last   time.Time // lần cuối cập nhật token
}

// NewRateLimiter tạo limiter với `rate` token/giây và sức chứa `burst`.
// Ví dụ rate=5, burst=60 ≈ trung bình 300 req/phút, cho phép dồn tối đa 60 req.
func NewRateLimiter(rate float64, burst int, clock Clock) *RateLimiter {
	if clock == nil {
		clock = SystemClock{}
	}
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   float64(burst),
		ttl:     10 * time.Minute, // gáo không dùng quá 10 phút sẽ bị dọn
		clock:   clock,
	}
}

// Allow trả true nếu request của `key` được phép (còn token) và tiêu 1 token.
// Trả false nếu hết token -> caller nên trả 429.
func (rl *RateLimiter) Allow(key string) bool {
	now := rl.clock.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		// Gáo mới: đầy token, tiêu 1 luôn.
		rl.buckets[key] = &bucket{tokens: rl.burst - 1, last: now}
		return true
	}

	// Nạp token theo thời gian đã trôi qua kể từ lần trước (elapsed * rate).
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(rl.burst, b.tokens+elapsed*rl.rate)
	b.last = now

	if b.tokens < 1 {
		return false // hết token -> chặn
	}
	b.tokens--
	return true
}

// Cleanup xóa các gáo lâu không dùng để map không phình vô hạn.
// Gọi định kỳ từ một goroutine nền (xem StartCleanup).
func (rl *RateLimiter) Cleanup() {
	now := rl.clock.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for key, b := range rl.buckets {
		if now.Sub(b.last) > rl.ttl {
			delete(rl.buckets, key)
		}
	}
}

// StartCleanup chạy Cleanup định kỳ cho tới khi stop được đóng.
// Trả về hàm stop để tắt goroutine khi shutdown.
func (rl *RateLimiter) StartCleanup(every time.Duration) (stop func()) {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.Cleanup()
			case <-done:
				return
			}
		}
	}()
	return func() { close(done) }
}
