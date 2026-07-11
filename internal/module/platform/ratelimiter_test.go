package platform

import (
	"testing"
	"time"
)

// fakeClock là Clock giả cho test: thời gian do ta điều khiển, không trôi thật.
// Nhờ nó ta test được logic "nạp token theo thời gian" mà KHÔNG cần sleep.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestRateLimiter_ChanKhiHetToken(t *testing.T) {
	clock := &fakeClock{t: time.Unix(0, 0)}
	// rate = 1 token/giây, gáo chứa tối đa 3.
	rl := NewRateLimiter(1, 3, clock)

	// 3 request đầu: gáo đầy -> đều được phép.
	for i := 1; i <= 3; i++ {
		if !rl.Allow("1.2.3.4") {
			t.Fatalf("request %d đáng lẽ được phép", i)
		}
	}

	// Request thứ 4 (chưa nạp lại token) -> bị chặn.
	if rl.Allow("1.2.3.4") {
		t.Fatal("request thứ 4 đáng lẽ bị chặn")
	}

	// Sau 1 giây -> nạp 1 token -> lại được 1 request.
	clock.advance(time.Second)
	if !rl.Allow("1.2.3.4") {
		t.Fatal("sau 1 giây đáng lẽ được phép lại")
	}
}

func TestRateLimiter_MoiKeyDocLap(t *testing.T) {
	clock := &fakeClock{t: time.Unix(0, 0)}
	rl := NewRateLimiter(1, 1, clock)

	// IP A tiêu hết token của mình...
	if !rl.Allow("A") {
		t.Fatal("A lần đầu phải được phép")
	}
	if rl.Allow("A") {
		t.Fatal("A lần hai phải bị chặn")
	}
	// ...nhưng KHÔNG ảnh hưởng IP B (gáo riêng).
	if !rl.Allow("B") {
		t.Fatal("B phải được phép vì có gáo riêng")
	}
}
