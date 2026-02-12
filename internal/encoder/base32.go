package encoder

import (
	"strings"

	"octopus/pkg/util"
)

const (
	// Base32Alphabet is the character set for Base32 encoding
	Base32Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	// MinLength is the minimum short code length
	MinLength = 4
	// MaxLength is the maximum short code length
	MaxLength = 6
)

// Base32Encoder encodes numbers to Base32 strings
type Base32Encoder struct{}

// NewBase32Encoder creates a new Base32Encoder
func NewBase32Encoder() *Base32Encoder {
	return &Base32Encoder{}
}

// Encode encodes a uint64 to a Base32 string of specified length
func (e *Base32Encoder) Encode(n uint64, length int) string {
	if length < MinLength || length > MaxLength {
		length = MinLength
	}

	result := make([]byte, length)
	alphabetLen := uint64(len(Base32Alphabet))

	for i := length - 1; i >= 0; i-- {
		result[i] = Base32Alphabet[n%alphabetLen]
		n = n / alphabetLen
	}

	return string(result)
}

// EncodeString encodes a string to a Base32 string of specified length
func (e *Base32Encoder) EncodeString(s string, length int) string {
	hash := util.HashString(s)
	return e.Encode(hash, length)
}

// Decode decodes a Base32 string to uint64
func (e *Base32Encoder) Decode(s string) (uint64, error) {
	s = strings.ToUpper(s)
	var result uint64
	alphabetLen := uint64(len(Base32Alphabet))

	for _, c := range s {
		index := uint64(0)
		found := false
		for i := 0; i < len(Base32Alphabet); i++ {
			if c == rune(Base32Alphabet[i]) {
				index = uint64(i)
				found = true
				break
			}
		}
		if !found {
			return 0, &InvalidCharacterError{Char: c}
		}
		result = result*alphabetLen + index
	}

	return result, nil
}

// IsValid checks if a string is a valid Base32 code
func (e *Base32Encoder) IsValid(s string) bool {
	if len(s) < MinLength || len(s) > MaxLength {
		return false
	}

	s = strings.ToUpper(s)
	for _, c := range s {
		valid := false
		for i := 0; i < len(Base32Alphabet); i++ {
			if c == rune(Base32Alphabet[i]) {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
	}

	return true
}

// MaxCapacity returns the maximum capacity for a given length
func (e *Base32Encoder) MaxCapacity(length int) uint64 {
	alphabetLen := uint64(len(Base32Alphabet))
	capacity := uint64(1)
	for i := 0; i < length; i++ {
		capacity *= alphabetLen
	}
	return capacity
}

// InvalidCharacterError is returned when an invalid character is encountered
type InvalidCharacterError struct {
	Char rune
}

func (e *InvalidCharacterError) Error() string {
	return "invalid character: " + string(e.Char)
}
