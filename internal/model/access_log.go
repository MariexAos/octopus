package model

import (
	"time"
)

// AccessLog represents an access log entity
type AccessLog struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ShortCode  string    `json:"short_code" gorm:"type:varchar(6);index;not null"`
	ClientIP   string    `json:"client_ip" gorm:"type:varchar(64)"`
	UserAgent  string    `json:"user_agent" gorm:"type:varchar(512)"`
	Referer    string    `json:"referer" gorm:"type:varchar(512)"`
	AccessTime time.Time `json:"access_time" gorm:"autoCreateTime"`
}

// TableName returns the table name for AccessLog
func (AccessLog) TableName() string {
	return "access_logs"
}

// AccessLogMessage represents the message sent to RocketMQ
type AccessLogMessage struct {
	ShortCode  string    `json:"short_code"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	Referer    string    `json:"referer"`
	AccessTime time.Time `json:"access_time"`
}

// AnalyticsResponse represents the analytics data
type AnalyticsResponse struct {
	ShortCode  string          `json:"short_code"`
	PV         int64           `json:"pv"`
	UV         int64           `json:"uv"`
	TopSources []SourceStat    `json:"top_sources"`
}

// SourceStat represents source statistics
type SourceStat struct {
	Source string `json:"source"`
	Count  int64  `json:"count"`
}

// Stats represents general statistics
type Stats struct {
	PV int64 `json:"pv"`
	UV int64 `json:"uv"`
}
