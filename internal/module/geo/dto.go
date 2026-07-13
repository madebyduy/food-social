package geo

// dto.go — hình dạng JSON cho các endpoint tra cứu địa lý (chỉ đọc).

type ProvinceResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type DistrictResponse struct {
	ID         int64  `json:"id"`
	ProvinceID int64  `json:"province_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
}

type WardResponse struct {
	ID         int64  `json:"id"`
	DistrictID int64  `json:"district_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
}

func toProvinceResponses(items []Province) []ProvinceResponse {
	out := make([]ProvinceResponse, 0, len(items))
	for _, p := range items {
		out = append(out, ProvinceResponse{ID: p.ID, Name: p.Name, Slug: p.Slug})
	}
	return out
}

func toDistrictResponses(items []District) []DistrictResponse {
	out := make([]DistrictResponse, 0, len(items))
	for _, d := range items {
		out = append(out, DistrictResponse{ID: d.ID, ProvinceID: d.ProvinceID, Name: d.Name, Slug: d.Slug})
	}
	return out
}

func toWardResponses(items []Ward) []WardResponse {
	out := make([]WardResponse, 0, len(items))
	for _, w := range items {
		out = append(out, WardResponse{ID: w.ID, DistrictID: w.DistrictID, Name: w.Name, Slug: w.Slug})
	}
	return out
}
