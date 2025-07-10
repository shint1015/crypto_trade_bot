package domain

import "time"

// Asset は暗号資産の情報を保持するエンティティです。
type Asset struct {
	Symbol        string
	CurrentPrice  float64
	Price1H       float64
	PurchasePrice float64 // 購入価格
	Volume24H     float64
	MACD          float64
	RSI           float64
	LastUpdatedAt time.Time
}

// CalculateROI は1時間の投資収益率（ROI）を計算します。
func (a *Asset) CalculateROI() float64 {
	if a.Price1H == 0 {
		return 0.0
	}
	return ((a.CurrentPrice - a.Price1H) / a.Price1H) * 100
}
