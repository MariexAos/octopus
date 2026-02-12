package mq

import (
	"context"
)

// ProducerInterface defines the interface for message production
type ProducerInterface interface {
	SendAccessLog(ctx context.Context, msg *AccessLogMessage) error
	Close() error
}

// ConsumerInterface defines the interface for message consumption
type ConsumerInterface interface {
	Subscribe() error
	Close() error
}
