package api

import "fmt"

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

// DisplayAmount returns a human-friendly value for rendering in tables.
func (cs CostSample) DisplayAmount() string {
	if cs.Amount == nil {
		return "-"
	}
	return fmt.Sprintf("%v", *cs.Amount)
}

// DisplayAmountUSD returns a value suitable for markdown templates where we want a "$" prefix.
func (cs CostSample) DisplayAmountUSD() string {
	if cs.Amount == nil {
		return "-"
	}
	return fmt.Sprintf("$%v", *cs.Amount)
}
