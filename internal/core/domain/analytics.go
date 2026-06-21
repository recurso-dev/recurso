package domain

type UsageStats struct {
	Dimension     string `json:"dimension"`
	TotalQuantity int64  `json:"total_quantity"`
}
