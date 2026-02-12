package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"octopus/internal/config"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/rs/zerolog/log"
)

// AccessLogHandler is the handler for access log messages
type AccessLogHandler func(ctx context.Context, msg *AccessLogMessage) error

// Consumer handles message consumption from RocketMQ
type Consumer struct {
	client   rocketmq.PushConsumer
	topic    string
	group    string
	handler  AccessLogHandler
	once     sync.Once
	started  bool
}

// NewConsumer creates a new RocketMQ consumer
func NewConsumer(cfg *config.RocketMQConfig, handler AccessLogHandler) (*Consumer, error) {
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{cfg.NameServer}),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithGroupName(cfg.Group),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create RocketMQ consumer: %w", err)
	}

	return &Consumer{
		client:  c,
		topic:   cfg.Topic,
		group:   cfg.Group,
		handler: handler,
	}, nil
}

// Subscribe subscribes to the topic and starts consuming messages
func (c *Consumer) Subscribe() error {
	if c.started {
		return nil
	}

	err := c.client.Subscribe(c.topic, consumer.MessageSelector{}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, msg := range msgs {
			var accessLog AccessLogMessage
			if err := json.Unmarshal(msg.Body, &accessLog); err != nil {
				log.Error().Err(err).Str("msg_id", msg.MsgId).Msg("Failed to unmarshal message")
				return consumer.ConsumeRetryLater, err
			}

			log.Debug().
				Str("msg_id", msg.MsgId).
				Str("short_code", accessLog.ShortCode).
				Msg("Processing access log")

			if c.handler != nil {
				if err := c.handler(ctx, &accessLog); err != nil {
					log.Error().Err(err).Str("msg_id", msg.MsgId).Msg("Handler failed")
					return consumer.ConsumeRetryLater, err
				}
			}
		}
		return consumer.ConsumeSuccess, nil
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	if err := c.client.Start(); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	c.started = true
	log.Info().Str("topic", c.topic).Msg("RocketMQ consumer started")

	return nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
	if c != nil && c.client != nil {
		return c.client.Shutdown()
	}
	return nil
}
