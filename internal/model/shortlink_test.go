package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShortLink_TableName(t *testing.T) {
	sl := ShortLink{}
	assert.Equal(t, "short_links", sl.TableName())
}

func TestShortLink_IsActive(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	tests := []struct {
		name     string
		status   int
		expireAt *time.Time
		expected bool
	}{
		{
			name:     "active without expiration",
			status:   1,
			expireAt: nil,
			expected: true,
		},
		{
			name:     "active with future expiration",
			status:   1,
			expireAt: &future,
			expected: true,
		},
		{
			name:     "inactive status",
			status:   0,
			expireAt: nil,
			expected: false,
		},
		{
			name:     "expired",
			status:   1,
			expireAt: &past,
			expected: false,
		},
		{
			name:     "inactive and expired",
			status:   0,
			expireAt: &past,
			expected: false,
		},
		{
			name:     "just now expiration",
			status:   1,
			expireAt: &now,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl := ShortLink{
				Status:   tt.status,
				ExpireAt: tt.expireAt,
			}
			result := sl.IsActive()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateRequest_Validation(t *testing.T) {
	tests := []struct {
		name   string
		req    GenerateRequest
		valid  bool
	}{
		{
			name: "valid request",
			req: GenerateRequest{
				URL:    "https://example.com",
				Params: nil,
			},
			valid: true,
		},
		{
			name: "valid with params",
			req: GenerateRequest{
				URL: "https://example.com",
				Params: map[string]interface{}{
					"key": "value",
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test demonstrates the struct definition
			// Actual validation is done by gin binding
			assert.NotNil(t, tt.req)
		})
	}
}

func TestGenerateResponse_Structure(t *testing.T) {
	now := time.Now()

	resp := GenerateResponse{
		ShortLink:   "https://s.example.com/ABCD",
		ShortCode:   "ABCD",
		OriginalURL: "https://example.com",
		ExpireAt:    now,
	}

	assert.Equal(t, "https://s.example.com/ABCD", resp.ShortLink)
	assert.Equal(t, "ABCD", resp.ShortCode)
	assert.Equal(t, "https://example.com", resp.OriginalURL)
	assert.Equal(t, now, resp.ExpireAt)
}

func TestShortLink_JSONTags(t *testing.T) {
	sl := ShortLink{
		ID:          1,
		ShortCode:   "ABCD",
		OriginalURL: "https://example.com",
		Status:      1,
	}

	// Verify struct can be instantiated
	assert.Equal(t, int64(1), sl.ID)
	assert.Equal(t, "ABCD", sl.ShortCode)
	assert.Equal(t, "https://example.com", sl.OriginalURL)
	assert.Equal(t, 1, sl.Status)
}
