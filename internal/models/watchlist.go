package models

import "time"

type WatchlistItem struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name,omitempty"`
	GroupName string    `json:"group_name"`
	AddedAt   time.Time `json:"added_at"`
}
