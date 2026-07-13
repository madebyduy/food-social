package vote

import "time"

// CalculateWeight snapshot trọng số theo tuổi tài khoản.
func CalculateWeight(createdAt, now time.Time) float64 {
	age := now.Sub(createdAt)
	switch {
	case age < 7*24*time.Hour:
		return 0.25
	case age < 30*24*time.Hour:
		return 0.5
	default:
		return 1
	}
}
func ShouldHide(total int, untrustedRatio float64) bool { return total >= 200 && untrustedRatio > 0.70 }
