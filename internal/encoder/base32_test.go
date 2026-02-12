package encoder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBase32Encoder_Encode(t *testing.T) {
	encoder := NewBase32Encoder()

	tests := []struct {
		name     string
		input    uint64
		length   int
		expected string
	}{
		{
			name:     "encode zero",
			input:    0,
			length:   4,
			expected: "AAAA",
		},
		{
			name:     "encode 1",
			input:    1,
			length:   4,
			expected: "AAAB",
		},
		{
			name:     "encode 32",
			input:    32,
			length:   4,
			expected: "AABA",
		},
		{
			name:     "encode 33",
			input:    33,
			length:   4,
			expected: "AABB",
		},
		{
			name:     "encode large number",
			input:    1000000,
			length:   4,
			expected: "6QSA",
		},
		{
			name:     "encode with max length",
			input:    123456789,
			length:   6,
			expected: "DVXTIV",
		},
		{
			name:     "encode with minimum length",
			input:    1,
			length:   4,
			expected: "AAAB",
		},
		{
			name:     "length below minimum defaults to 4",
			input:    1,
			length:   3,
			expected: "AAAB",
		},
		{
			name:     "length above maximum defaults to 4",
			input:    1,
			length:   10,
			expected: "AAAB",
		},
		{
			name:     "encode alphabet boundary",
			input:    31,
			length:   4,
			expected: "AAA7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encoder.Encode(tt.input, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBase32Encoder_Decode(t *testing.T) {
	encoder := NewBase32Encoder()

	tests := []struct {
		name        string
		input       string
		expected    uint64
		expectError bool
	}{
		{
			name:     "decode AAAA",
			input:    "AAAA",
			expected: 0,
		},
		{
			name:     "decode AAAB",
			input:    "AAAB",
			expected: 1,
		},
		{
			name:     "decode 6QSA",
			input:    "6QSA",
			expected: 1000000,
		},
		{
			name:     "decode lowercase",
			input:    "aaab",
			expected: 1,
		},
		{
			name:        "decode invalid character",
			input:       "AAA1",
			expectError: true,
		},
		{
			name:        "decode with special char",
			input:       "AAA-",
			expectError: true,
		},
		{
			name:     "decode single char",
			input:    "A",
			expected: 0,
		},
		{
			name:     "decode Z",
			input:    "Z",
			expected: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encoder.Decode(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.IsType(t, &InvalidCharacterError{}, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBase32Encoder_EncodeDecode(t *testing.T) {
	encoder := NewBase32Encoder()

	testValues := []uint64{
		0, 1, 2, 10, 100, 1000, 10000, 100000, 1000000, 12345678, 999999999,
	}

	for _, value := range testValues {
		t.Run("", func(t *testing.T) {
			encoded := encoder.Encode(value, 6)
			decoded, err := encoder.Decode(encoded)
			require.NoError(t, err)
			assert.Equal(t, value, decoded)
		})
	}
}

func TestBase32Encoder_EncodeString(t *testing.T) {
	encoder := NewBase32Encoder()

	tests := []struct {
		name   string
		input  string
		length int
	}{
		{
			name:   "encode empty string",
			input:  "",
			length: 4,
		},
		{
			name:   "encode URL",
			input:  "https://example.com",
			length: 6,
		},
		{
			name:   "encode string with special chars",
			input:  "hello!@#$%",
			length: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encoder.EncodeString(tt.input, tt.length)
			assert.Len(t, result, tt.length)
			// Verify it's a valid Base32 string
			assert.True(t, encoder.IsValid(result))
		})
	}
}

func TestBase32Encoder_IsValid(t *testing.T) {
	encoder := NewBase32Encoder()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid minimum length",
			input:    "AAAA",
			expected: true,
		},
		{
			name:     "valid maximum length",
			input:    "AAAAAA",
			expected: true,
		},
		{
			name:     "valid middle length",
			input:    "AAAAA",
			expected: true,
		},
		{
			name:     "too short",
			input:    "AAA",
			expected: false,
		},
		{
			name:     "too long",
			input:    "AAAAAAA",
			expected: false,
		},
		{
			name:     "invalid character - number",
			input:    "AAA1",
			expected: false,
		},
		{
			name:     "invalid character - lowercase (but IsValid converts to upper)",
			input:    "aaab",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "valid with all chars",
			input:    "AZZ7",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encoder.IsValid(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBase32Encoder_MaxCapacity(t *testing.T) {
	encoder := NewBase32Encoder()

	tests := []struct {
		name     string
		length   int
		expected uint64
	}{
		{
			name:     "length 1",
			length:   1,
			expected: 32,
		},
		{
			name:     "length 2",
			length:   2,
			expected: 1024,
		},
		{
			name:     "length 4",
			length:   4,
			expected: 1048576,
		},
		{
			name:     "length 6",
			length:   6,
			expected: 1073741824,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encoder.MaxCapacity(tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInvalidCharacterError(t *testing.T) {
	err := &InvalidCharacterError{Char: '1'}
	assert.Equal(t, "invalid character: 1", err.Error())
}

func TestBase32Encoder_Concurrent(t *testing.T) {
	encoder := NewBase32Encoder()
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func() {
			encoder.Encode(uint64(i), 4)
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}
