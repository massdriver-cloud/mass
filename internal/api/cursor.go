package api

// Cursor is the GraphQL `Cursor` input type. Defined here (rather than letting
// genqlient generate it) so that the `omitempty` tags drop zero-value fields —
// otherwise paginated requests send `limit: 0`, which the server rejects since
// `Cursor.limit` is constrained to 1..100.
//
// Bound to the GraphQL `Cursor` type via genqlient.yaml.
type Cursor struct {
	Limit    int    `json:"limit,omitempty"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
}
