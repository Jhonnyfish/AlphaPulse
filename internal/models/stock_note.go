package models

import (
	"time"
)

// StockNote represents a user's note attached to a stock code.
type StockNote struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StockNoteListResponse is the response for GET /api/stock-notes/:code.
type StockNoteListResponse struct {
	OK    bool         `json:"ok"`
	Notes []StockNote  `json:"notes"`
}

// StockNoteResponse wraps a single note.
type StockNoteResponse struct {
	OK   bool       `json:"ok"`
	Note StockNote  `json:"note"`
}

// StockNoteTagsResponse is the response for GET /api/stock-notes/tags/all.
type StockNoteTagsResponse struct {
	OK   bool     `json:"ok"`
	Tags []string `json:"tags"`
}
