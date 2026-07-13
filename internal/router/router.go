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
	"github.com/madebyduy/food-social/internal/module/geo"
	"github.com/madebyduy/food-social/internal/module/governance"
	"github.com/madebyduy/food-social/internal/module/location"
	"github.com/madebyduy/food-social/internal/module/media"
	"github.com/madebyduy/food-social/internal/module/place"
	"github.com/madebyduy/food-social/internal/module/post"
	"github.com/madebyduy/food-social/internal/module/search"
	"github.com/madebyduy/food-social/internal/module/social"
	"github.com/madebyduy/food-social/internal/module/user"
)

// Dependencies gom mọi thứ router cần để gắn route. main.go tạo và truyền vào.
// Khi thêm module mới (post, comment...), chỉ cần thêm *Handler tương ứng ở đây.
type Dependencies struct {
	DB                *sql.DB
	AuthHandler       *auth.Handler
	UserHandler       *user.Handler
	PostHandler       *post.Handler
	GeoHandler        *geo.Handler
	PlaceHandler      *place.Handler
	MediaHandler      *media.Handler
	SocialHandler     *social.Handler
	GovernanceHandler *governance.Handler
	SearchHandler     *search.Handler
	LocationHandler   *location.Handler
	SessionResolver   middleware.SessionResolver // để bọc route cần đăng nhập
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
	mux.HandleFunc("GET /api/v1/search", deps.SearchHandler.Search)

	// --- Auth ---
	mux.HandleFunc("POST /api/v1/auth/register", deps.AuthHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", deps.AuthHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/password-reset/request", deps.AuthHandler.RequestPasswordReset)
	mux.HandleFunc("POST /api/v1/auth/password-reset/confirm", deps.AuthHandler.ResetPassword)
	mux.Handle("POST /api/v1/auth/logout", requireAuth(deps.AuthHandler.Logout))
	mux.Handle("GET /api/v1/me", requireAuth(deps.AuthHandler.Me))
	mux.Handle("DELETE /api/v1/me", requireAuth(deps.AuthHandler.DeleteAccount))

	// --- Users ---
	// 🟢 Công khai: xem hồ sơ.
	mux.HandleFunc("GET /api/v1/users/{id}", deps.UserHandler.GetByID)
	// 🔵 Cần đăng nhập: sửa hồ sơ chính mình.
	mux.Handle("PATCH /api/v1/users/{id}", requireAuth(deps.UserHandler.Update))
	// 🟢 Công khai: bài viết của một user + số lượng bài.
	mux.HandleFunc("GET /api/v1/users/{id}/posts", deps.PostHandler.ListByUser)
	mux.HandleFunc("GET /api/v1/users/{id}/posts/count", deps.PostHandler.CountVisibleByUser)

	// --- Posts ---
	// 🟢 Công khai: feed + xem chi tiết.
	mux.HandleFunc("GET /api/v1/posts", deps.PostHandler.Feed)
	mux.HandleFunc("GET /api/v1/posts/{id}", deps.PostHandler.GetByID)
	// 🔵 Cần đăng nhập: tạo / sửa / xóa bài (sửa-xóa chỉ tác giả, kiểm ở service).
	mux.Handle("POST /api/v1/posts", requireAuth(deps.PostHandler.Create))
	mux.Handle("PATCH /api/v1/posts/{id}", requireAuth(deps.PostHandler.Update))
	mux.Handle("DELETE /api/v1/posts/{id}", requireAuth(deps.PostHandler.Delete))

	// --- Social interactions ---
	mux.HandleFunc("GET /api/v1/posts/{id}/comments", deps.SocialHandler.ListComments)
	mux.Handle("POST /api/v1/posts/{id}/comments", requireAuth(deps.SocialHandler.AddComment))
	mux.Handle("POST /api/v1/posts/{id}/location-suggestions", requireAuth(deps.LocationHandler.Suggest))
	mux.Handle("POST /api/v1/location-suggestions/{id}/accept", requireAuth(deps.LocationHandler.Accept))
	mux.Handle("POST /api/v1/location-suggestions/{id}/reject", requireAuth(deps.LocationHandler.Reject))
	mux.Handle("POST /api/v1/posts/{id}/like", requireAuth(deps.SocialHandler.Like))
	mux.Handle("DELETE /api/v1/posts/{id}/like", requireAuth(deps.SocialHandler.Unlike))
	mux.Handle("POST /api/v1/posts/{id}/save", requireAuth(deps.SocialHandler.Save))
	mux.Handle("DELETE /api/v1/posts/{id}/save", requireAuth(deps.SocialHandler.Unsave))
	mux.Handle("POST /api/v1/users/{id}/follow", requireAuth(deps.SocialHandler.Follow))
	mux.Handle("DELETE /api/v1/users/{id}/follow", requireAuth(deps.SocialHandler.Unfollow))
	mux.Handle("GET /api/v1/me/saved-posts", requireAuth(deps.SocialHandler.Saved))
	mux.Handle("GET /api/v1/feed/following", requireAuth(deps.SocialHandler.FollowingFeed))
	mux.Handle("POST /api/v1/users/{id}/block", requireAuth(deps.SocialHandler.Block))
	mux.Handle("DELETE /api/v1/users/{id}/block", requireAuth(deps.SocialHandler.Unblock))
	mux.Handle("POST /api/v1/users/{id}/mute", requireAuth(deps.SocialHandler.Mute))
	mux.Handle("DELETE /api/v1/users/{id}/mute", requireAuth(deps.SocialHandler.Unmute))

	// --- Trust, notification, report and moderation ---
	mux.Handle("POST /api/v1/reports", requireAuth(deps.GovernanceHandler.Report))
	mux.Handle("POST /api/v1/posts/{id}/vote", requireAuth(deps.GovernanceHandler.Vote))
	mux.Handle("DELETE /api/v1/posts/{id}/vote", requireAuth(deps.GovernanceHandler.RemoveVote))
	mux.Handle("GET /api/v1/notifications", requireAuth(deps.GovernanceHandler.ListNotifications))
	mux.Handle("POST /api/v1/notifications/{id}/read", requireAuth(deps.GovernanceHandler.ReadNotification))
	mux.Handle("POST /api/v1/admin/posts/{id}/hide", requireAuth(deps.GovernanceHandler.HidePost))
	mux.Handle("POST /api/v1/admin/posts/{id}/restore", requireAuth(deps.GovernanceHandler.RestorePost))

	// --- Geo (🟢 công khai: dữ liệu tra cứu tỉnh/quận/phường) ---
	mux.HandleFunc("GET /api/v1/provinces", deps.GeoHandler.ListProvinces)
	mux.HandleFunc("GET /api/v1/provinces/{id}/districts", deps.GeoHandler.ListDistricts)
	mux.HandleFunc("GET /api/v1/districts/{id}/wards", deps.GeoHandler.ListWards)

	// --- Places ---
	// 🟢 Công khai: tìm/gợi ý địa điểm + xem chi tiết.
	mux.HandleFunc("GET /api/v1/places", deps.PlaceHandler.Search)
	mux.HandleFunc("GET /api/v1/places/{id}", deps.PlaceHandler.GetByID)
	// 🔵 Cần đăng nhập: tạo địa điểm (hậu kiểm).
	mux.Handle("POST /api/v1/places", requireAuth(deps.PlaceHandler.Create))

	// --- Media (🔵 cần đăng nhập: ký upload + xác nhận ảnh) ---
	mux.Handle("POST /api/v1/media/sign", requireAuth(deps.MediaHandler.Sign))
	mux.Handle("POST /api/v1/media/{id}/confirm", requireAuth(deps.MediaHandler.Confirm))

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
