package vote

import (
	"testing"
	"time"
)

func TestCalculateWeight(t *testing.T) {
	now := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		age  time.Duration
		want float64
	}{{"new", 5 * 24 * time.Hour, .25}, {"growing", 10 * 24 * time.Hour, .5}, {"trusted", 40 * 24 * time.Hour, 1}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateWeight(now.Add(-tt.age), now); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
func TestShouldHide(t *testing.T) {
	if !ShouldHide(200, .71) || ShouldHide(199, .99) || ShouldHide(200, .70) {
		t.Fatal("boundary rule failed")
	}
}
