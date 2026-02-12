package mq

import (
	"time"
)

// AccessLogMessage represents an access log message
type AccessLogMessage struct {
	ShortCode  string    `json:"short_code"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	Referer    string    `json:"referer"`
	AccessTime time.Time `json:"access_time"`
}
