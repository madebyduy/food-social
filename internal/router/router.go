// Package router định nghĩa TẤT CẢ route của API và gắn chúng vào http.ServeMux.
//
// Go 1.22+ có pattern routing sẵn: "GET /api/v1/users/{id}" — không cần thư viện
// router ngoài. {id} lấy qua r.PathValue("id") (đã bọc trong httpx.PathInt64).
package router

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/madebyduy/food-social/internal/auth"
	"github.com/madebyduy/food-social/internal/httpx"
	"github.com/madebyduy/food-social/internal/middleware"
	"github.com/madebyduy/food-social/internal/module/user"
)

// Dependencies gom mọi thứ router cần để gắn route. main.go tạo và truyền vào.
// Khi thêm module mới (post, comment...), chỉ cần thêm *Handler tương ứng ở đây.
type Dependencies struct {
	DB              *sql.DB
	AuthHandler     *auth.Handler
	UserHandler     *user.Handler
	SessionResolver middleware.SessionResolver // để bọc route cần đăng nhập
}

// New dựng ServeMux và đăng ký toàn bộ route. Trả về http.Handler (chưa gắn middleware
// toàn cục — việc đó do main.go làm qua middleware.Chain).
func New(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	// requireAuth: helper bọc một handler đơn lẻ bằng middleware Authenticate.
	// Dùng cho các route 🔵 (cần đăng nhập).
	authenticate := middleware.Authenticate(deps.SessionResolver)
	requireAuth := func(h http.HandlerFunc) http.Handler {
		return authenticate(h)
	}

	// --- Health check ---
	mux.HandleFunc("GET /healthz", liveness)          // process còn sống?
	mux.HandleFunc("GET /readyz", readiness(deps.DB)) // sẵn sàng phục vụ (DB ok)?
	mux.HandleFunc("GET /api/v1", apiInfo)

	// --- Auth ---
	mux.HandleFunc("POST /api/v1/auth/register", deps.AuthHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", deps.AuthHandler.Login)
	mux.Handle("POST /api/v1/auth/logout", requireAuth(deps.AuthHandler.Logout))
	mux.Handle("GET /api/v1/me", requireAuth(deps.AuthHandler.Me))

	// --- Users ---
	// 🟢 Công khai: xem hồ sơ.
	mux.HandleFunc("GET /api/v1/users/{id}", deps.UserHandler.GetByID)
	// 🔵 Cần đăng nhập: sửa hồ sơ chính mình.
	mux.Handle("PATCH /api/v1/users/{id}", requireAuth(deps.UserHandler.Update))

	return mux
}

// liveness luôn trả 200 nếu tiến trình còn chạy (dùng cho k8s livenessProbe).
func liveness(w http.ResponseWriter, _ *http.Request) {
	httpx.OK(w, map[string]string{"status": "ok"})
}

// readiness ping DB: DB sống -> 200, DB chết -> 503 (dùng cho readinessProbe).
func readiness(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, httpx.Envelope{
				Data: map[string]string{"status": "unavailable", "database": "down"},
			})
			return
		}
		httpx.OK(w, map[string]any{
			"status":    "ok",
			"database":  "up",
			"timestamp": time.Now().UTC(),
		})
	}
}

func apiInfo(w http.ResponseWriter, _ *http.Request) {
	httpx.OK(w, map[string]string{"name": "AnNgon API", "version": "v1"})
}
