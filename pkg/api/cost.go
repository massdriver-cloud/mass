package api

type Cost struct {
	Monthly Summary `json:"monthly"`
	Daily   Summary `json:"daily"`
}

// Summary of costs over a time period.
type Summary struct {
	Previous CostSample `json:"previous"`
	Average  CostSample `json:"average"`
}

// A single cost measurement. Fields may be null when no cost data exists.
type CostSample struct {
	Amount   *float64 `json:"amount"`
	Currency *string  `json:"currency"`
}
