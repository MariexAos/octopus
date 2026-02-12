package model

import (
	"encoding/json"
	"time"
)

// ShortLink represents a short link entity
type ShortLink struct {
	ID          int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	ShortCode   string          `json:"short_code" gorm:"type:varchar(6);uniqueIndex;not null"`
	OriginalURL string          `json:"original_url" gorm:"type:varchar(2048);not null"`
	Params      json.RawMessage `json:"params" gorm:"type:json"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	ExpireAt    *time.Time      `json:"expire_at" gorm:"index"`
	Status      int             `json:"status" gorm:"default:1;comment:1-active,0-disabled"`
}

// TableName returns the table name for ShortLink
func (ShortLink) TableName() string {
	return "short_links"
}

// IsActive checks if the short link is active and not expired
func (sl *ShortLink) IsActive() bool {
	if sl.Status != 1 {
		return false
	}
	if sl.ExpireAt != nil && time.Now().After(*sl.ExpireAt) {
		return false
	}
	return true
}

// GenerateRequest represents the request to generate a short link
type GenerateRequest struct {
	URL    string                 `json:"url" binding:"required,url"`
	Params map[string]interface{} `json:"params"`
	ExpireAt string                `json:"expire_at"`
}

// GenerateResponse represents the response of short link generation
type GenerateResponse struct {
	ShortLink   string    `json:"short_link"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	ExpireAt    time.Time `json:"expire_at,omitempty"`
}
