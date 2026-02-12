package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		notNil bool
	}{
		{
			name:   "empty string",
			input:  "",
			notNil: true,
		},
		{
			name:   "simple string",
			input:  "hello",
			notNil: true,
		},
		{
			name:   "URL",
			input:  "https://example.com/path",
			notNil: true,
		},
		{
			name:   "string with special chars",
			input:  "hello!@#$%^&*()",
			notNil: true,
		},
		{
			name:   "unicode string",
			input:  "你好世界",
			notNil: true,
		},
		{
			name:   "long string",
			input:  string(make([]byte, 1000)),
			notNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashString(tt.input)
			assert.True(t, tt.notNil, "hash should not be zero value")
			assert.Greater(t, result, uint64(0))
		})
	}
}

func TestHashString_Consistency(t *testing.T) {
	input := "test string"

	hash1 := HashString(input)
	hash2 := HashString(input)
	hash3 := HashString(input)

	assert.Equal(t, hash1, hash2, "hash should be consistent")
	assert.Equal(t, hash2, hash3, "hash should be consistent across multiple calls")
}

func TestHashString_Distribution(t *testing.T) {
	// Test that different strings produce different hashes
	hashes := make(map[uint64]bool)
	inputs := []string{
		"a", "b", "c", "aa", "ab", "abc", "test", "testing", "hello", "world",
	}

	for _, input := range inputs {
		hash := HashString(input)
		hashes[hash] = true
	}

	// Most hashes should be different
	assert.GreaterOrEqual(t, len(hashes), len(inputs)/2)
}

func TestHashString_EmptyVsNonEmpty(t *testing.T) {
	emptyHash := HashString("")
	nonEmptyHash := HashString("something")

	assert.NotEqual(t, emptyHash, nonEmptyHash)
}

func TestHashString_CaseSensitive(t *testing.T) {
	upper := HashString("HELLO")
	lower := HashString("hello")

	assert.NotEqual(t, upper, lower, "hash should be case sensitive")
}

func TestHashString_LongString(t *testing.T) {
	longStr := string(make([]byte, 10000))
	for i := range longStr {
		longStr = longStr[:i] + string(byte(i%256)) + longStr[i+1:]
	}

	hash := HashString(longStr)
	assert.Greater(t, hash, uint64(0))
}
