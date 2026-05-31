package vision

type DetectRequest struct {
	ImagePath string  `json:"image_path"`
	Threshold float64 `json:"threshold"`
}

type DetectResponse struct {
	IsCapybara bool    `json:"is_capybara"`
	Score      float64 `json:"score"`
	Error      string  `json:"error"`
}

type ReadyResponse struct {
	Ready bool `json:"ready"`
}
