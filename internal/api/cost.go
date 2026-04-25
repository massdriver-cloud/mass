package api

// CostSummary holds cost data for a resource.
type CostSummary struct {
	LastMonth      CostSample `json:"lastMonth" mapstructure:"lastMonth"`
	MonthlyAverage CostSample `json:"monthlyAverage" mapstructure:"monthlyAverage"`
	LastDay        CostSample `json:"lastDay" mapstructure:"lastDay"`
	DailyAverage   CostSample `json:"dailyAverage" mapstructure:"dailyAverage"`
}

// CostSample is a single cost measurement. Fields may be null when no cost data exists.
type CostSample struct {
	Amount   *float64 `json:"amount" mapstructure:"amount"`
	Currency *string  `json:"currency" mapstructure:"currency"`
}
