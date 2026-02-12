package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"octopus/internal/config"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/rs/zerolog/log"
)

// Producer handles message production to RocketMQ
type Producer struct {
	client rocketmq.Producer
	topic  string
}

// NewProducer creates a new RocketMQ producer
func NewProducer(cfg *config.RocketMQConfig) (*Producer, error) {
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{cfg.NameServer}),
		producer.WithRetry(3),
		producer.WithGroupName(cfg.Group+"_producer"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create RocketMQ producer: %w", err)
	}

	if err := p.Start(); err != nil {
		return nil, fmt.Errorf("failed to start RocketMQ producer: %w", err)
	}

	log.Info().Str("topic", cfg.Topic).Msg("RocketMQ producer started")

	return &Producer{
		client: p,
		topic:  cfg.Topic,
	}, nil
}

// SendAccessLog sends an access log message to RocketMQ
func (p *Producer) SendAccessLog(ctx context.Context, msg *AccessLogMessage) error {
	if p == nil {
		return nil // Producer disabled
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	m := primitive.NewMessage(p.topic, bytes)
	m.WithTag("access_log")
	m.WithKeys([]string{msg.ShortCode})

	result, err := p.client.SendSync(ctx, m)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Debug().
		Str("msg_id", result.MsgID).
		Str("short_code", msg.ShortCode).
		Msg("Access log sent to RocketMQ")

	return nil
}

// Close closes the producer
func (p *Producer) Close() error {
	if p != nil && p.client != nil {
		return p.client.Shutdown()
	}
	return nil
}
