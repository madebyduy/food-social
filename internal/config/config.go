// Package config đọc biến môi trường vào một struct Config đã kiểm tra sẵn (validated).
//
// Nguyên tắc "fail-fast": nếu thiếu cấu hình bắt buộc (DATABASE_URL) thì trả lỗi ngay
// lúc khởi động, thay vì để server chạy rồi sập giữa chừng.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr        string        // địa chỉ server lắng nghe, vd ":8080"
	DatabaseURL string        // chuỗi kết nối Postgres (bắt buộc)
	LogLevel    slog.Level    // mức log: debug/info/warn/error
	SessionTTL  time.Duration // thời hạn session (dùng ở module auth về sau)
}

// Load đọc env -> Config. Trả lỗi nếu thiếu biến bắt buộc.
func Load() (Config, error) {
	cfg := Config{
		Addr:        getEnv("ADDR", ":8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		SessionTTL:  30 * 24 * time.Hour,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// parseLogLevel đổi chuỗi env thành slog.Level; không nhận diện được -> info.
func parseLogLevel(raw string) slog.Level {
	switch strings.ToLower(raw) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
