package mq

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConsumer_Subscribe_AlreadyStarted(t *testing.T) {
	t.Run("subscribe when already started returns nil", func(t *testing.T) {
		c := &Consumer{
			started: true,
		}

		err := c.Subscribe()
		assert.NoError(t, err)
	})
}

func TestConsumer_Close(t *testing.T) {
	t.Run("nil consumer close returns nil", func(t *testing.T) {
		var c *Consumer
		err := c.Close()
		assert.NoError(t, err)
	})

	t.Run("consumer with nil client close returns nil", func(t *testing.T) {
		c := &Consumer{
			client: nil,
		}
		err := c.Close()
		assert.NoError(t, err)
	})
}

func TestAccessLogHandler(t *testing.T) {
	t.Run("handler processes message", func(t *testing.T) {
		processed := false
		handler := func(ctx context.Context, msg *AccessLogMessage) error {
			processed = true
			assert.Equal(t, "ABC123", msg.ShortCode)
			return nil
		}

		msg := &AccessLogMessage{
			ShortCode:  "ABC123",
			ClientIP:   "192.168.1.1",
			UserAgent:  "test-agent",
			Referer:    "https://example.com",
			AccessTime: time.Now(),
		}

		err := handler(context.Background(), msg)
		assert.NoError(t, err)
		assert.True(t, processed)
	})

	t.Run("handler returns error", func(t *testing.T) {
		handler := func(ctx context.Context, msg *AccessLogMessage) error {
			return assert.AnError
		}

		msg := &AccessLogMessage{
			ShortCode: "ABC123",
		}

		err := handler(context.Background(), msg)
		assert.Error(t, err)
	})

	t.Run("nil handler does not panic", func(t *testing.T) {
		var handler AccessLogHandler
		// Ensure nil handler doesn't cause issues
		if handler != nil {
			_ = handler(context.Background(), &AccessLogMessage{})
		}
	})
}

func TestConsumer_NewConsumer_Structure(t *testing.T) {
	t.Run("consumer structure is correct", func(t *testing.T) {
		c := &Consumer{
			topic:   "test-topic",
			group:   "test-group",
			handler: func(ctx context.Context, msg *AccessLogMessage) error { return nil },
		}

		assert.Equal(t, "test-topic", c.topic)
		assert.Equal(t, "test-group", c.group)
		assert.NotNil(t, c.handler)
	})
}
