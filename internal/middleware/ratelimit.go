package middleware

import (
	"net"
	"net/http"

	"github.com/madebyduy/food-social/internal/apperr"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/module/platform"
)

// RateLimit limits requests by client IP.
func RateLimit(limiter *platform.RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(clientIP(r)) {
				httpx.Error(w, apperr.TooMany("quá nhiều yêu cầu, vui lòng thử lại sau"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP intentionally uses RemoteAddr only. Do not trust X-Forwarded-For
// until the app has explicit trusted-proxy configuration.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
