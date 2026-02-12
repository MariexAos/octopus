package util

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateUUID(t *testing.T) {
	uuid := GenerateUUID()

	// Check format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	uuidPattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	assert.True(t, uuidPattern.MatchString(uuid), "UUID should match standard format")
	assert.NotEmpty(t, uuid)
}

func TestGenerateUUID_Uniqueness(t *testing.T) {
	uuids := make(map[string]bool)

	// Generate 100 UUIDs and check uniqueness
	for i := 0; i < 100; i++ {
		uuid := GenerateUUID()
		assert.False(t, uuids[uuid], "UUID should be unique")
		uuids[uuid] = true
	}

	assert.Equal(t, 100, len(uuids))
}

func TestGenerateUUID_Concurrent(t *testing.T) {
	uuids := make(map[string]bool)
	done := make(chan string, 100)

	// Generate UUIDs concurrently
	for i := 0; i < 100; i++ {
		go func() {
			done <- GenerateUUID()
		}()
	}

	// Collect all UUIDs
	for i := 0; i < 100; i++ {
		uuid := <-done
		assert.False(t, uuids[uuid], "UUID should be unique in concurrent generation")
		uuids[uuid] = true
	}

	assert.Equal(t, 100, len(uuids))
}
