package geo

import (
	"net/http"

	"github.com/madebyduy/food-social/internal/httpx"
)

// Handler — tầng HTTP của module geo. Tất cả route đều 🟢 công khai (dữ liệu tra cứu).
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListProvinces: GET /api/v1/provinces
func (h *Handler) ListProvinces(w http.ResponseWriter, r *http.Request) {
	provinces, err := h.svc.ListProvinces(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toProvinceResponses(provinces))
}

// ListDistricts: GET /api/v1/provinces/{id}/districts
func (h *Handler) ListDistricts(w http.ResponseWriter, r *http.Request) {
	provinceID, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	districts, err := h.svc.ListDistricts(r.Context(), provinceID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toDistrictResponses(districts))
}

// ListWards: GET /api/v1/districts/{id}/wards
func (h *Handler) ListWards(w http.ResponseWriter, r *http.Request) {
	districtID, err := httpx.PathInt64(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	wards, err := h.svc.ListWards(r.Context(), districtID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toWardResponses(wards))
}
