package util

import (
	"hash/fnv"
)

// HashString returns a uint64 hash of the input string using FNV-1a
func HashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}
