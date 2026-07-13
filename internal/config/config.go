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
	Cloudinary  CloudinaryConfig
}

// CloudinaryConfig — cấu hình upload ảnh qua Cloudinary (module media).
// KHÔNG bắt buộc để server khởi động; chỉ các endpoint media mới cần. Secret đọc từ .env
// (đã gitignore), KHÔNG bao giờ hardcode trong mã nguồn.
type CloudinaryConfig struct {
	CloudName string
	APIKey    string
	APISecret string
	Folder    string
}

// Configured cho biết đã có đủ khóa để dùng Cloudinary chưa.
func (c CloudinaryConfig) Configured() bool {
	return c.CloudName != "" && c.APIKey != "" && c.APISecret != ""
}

// Load đọc env -> Config. Trả lỗi nếu thiếu biến bắt buộc.
func Load() (Config, error) {
	cfg := Config{
		Addr:        getEnv("ADDR", ":8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		SessionTTL:  30 * 24 * time.Hour,
		Cloudinary: CloudinaryConfig{
			CloudName: strings.TrimSpace(os.Getenv("CLOUDINARY_CLOUD_NAME")),
			APIKey:    strings.TrimSpace(os.Getenv("CLOUDINARY_API_KEY")),
			APISecret: strings.TrimSpace(os.Getenv("CLOUDINARY_API_SECRET")),
			Folder:    getEnv("CLOUDINARY_FOLDER", "anngon"),
		},
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
