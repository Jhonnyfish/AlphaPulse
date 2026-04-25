package models

// SearchSuggestion represents a stock search result for autocomplete.
type SearchSuggestion struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"` // e.g. "stock", "index", "fund"
}
