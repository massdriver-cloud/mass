// Package api provides client functions for interacting with the Massdriver API.
package api

// Cost holds monthly and daily cost summaries for a package.
type Cost struct {
	Monthly Summary `json:"monthly" mapstructure:"monthly"`
	Daily   Summary `json:"daily" mapstructure:"daily"`
}

// Summary of costs over a time period.
type Summary struct {
	Previous CostSample `json:"previous" mapstructure:"previous"`
	Average  CostSample `json:"average" mapstructure:"average"`
}

// CostSample is a single cost measurement. Fields may be null when no cost data exists.
type CostSample struct {
	Amount   *float64 `json:"amount" mapstructure:"amount"`
	Currency *string  `json:"currency" mapstructure:"currency"`
}
