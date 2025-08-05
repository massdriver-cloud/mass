package api

type Cost struct {
	Monthly *CostType `json:"monthly"`
	Daily   *CostType `json:"daily"`
}

type CostType struct {
	Average *CostSummary `json:"average"`
}

type CostSummary struct {
	Amount float64 `json:"amount"`
}
