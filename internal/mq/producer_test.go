package mq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProducer_SendAccessLog_NilProducer(t *testing.T) {
	t.Run("nil producer returns nil", func(t *testing.T) {
		var p *Producer
		msg := &AccessLogMessage{
			ShortCode:  "ABC123",
			ClientIP:   "192.168.1.1",
			UserAgent:  "test-agent",
			Referer:    "https://example.com",
			AccessTime: time.Now(),
		}

		err := p.SendAccessLog(context.Background(), msg)
		assert.NoError(t, err)
	})
}

func TestProducer_Close(t *testing.T) {
	t.Run("nil producer close returns nil", func(t *testing.T) {
		var p *Producer
		err := p.Close()
		assert.NoError(t, err)
	})
}

func TestAccessLogMessage(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		now := time.Now()
		msg := &AccessLogMessage{
			ShortCode:  "ABC123",
			ClientIP:   "192.168.1.1",
			UserAgent:  "test-agent",
			Referer:    "https://example.com",
			AccessTime: now,
		}

		// Test JSON marshaling
		data, err := json.Marshal(msg)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Test JSON unmarshaling
		var unmarshaled AccessLogMessage
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, msg.ShortCode, unmarshaled.ShortCode)
		assert.Equal(t, msg.ClientIP, unmarshaled.ClientIP)
		assert.Equal(t, msg.UserAgent, unmarshaled.UserAgent)
		assert.Equal(t, msg.Referer, unmarshaled.Referer)
	})

	t.Run("empty message", func(t *testing.T) {
		msg := &AccessLogMessage{}
		data, err := json.Marshal(msg)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})
}
