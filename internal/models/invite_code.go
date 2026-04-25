package models

import "time"

type InviteCode struct {
	ID        string     `json:"id"`
	Code      string     `json:"code"`
	CreatedBy *string    `json:"created_by,omitempty"`
	MaxUses   int        `json:"max_uses"`
	Uses      int        `json:"uses"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
