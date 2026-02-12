package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAccessLog_TableName(t *testing.T) {
	log := AccessLog{}
	assert.Equal(t, "access_logs", log.TableName())
}

func TestAccessLog_Structure(t *testing.T) {
	now := time.Now()

	log := AccessLog{
		ID:         1,
		ShortCode:  "ABCD",
		ClientIP:   "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Referer:    "https://google.com",
		AccessTime: now,
	}

	assert.Equal(t, int64(1), log.ID)
	assert.Equal(t, "ABCD", log.ShortCode)
	assert.Equal(t, "192.168.1.1", log.ClientIP)
	assert.Equal(t, "Mozilla/5.0", log.UserAgent)
	assert.Equal(t, "https://google.com", log.Referer)
	assert.Equal(t, now, log.AccessTime)
}

func TestAccessLogMessage_Structure(t *testing.T) {
	now := time.Now()

	msg := AccessLogMessage{
		ShortCode:  "ABCD",
		ClientIP:   "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Referer:    "https://google.com",
		AccessTime: now,
	}

	assert.Equal(t, "ABCD", msg.ShortCode)
	assert.Equal(t, "192.168.1.1", msg.ClientIP)
	assert.Equal(t, "Mozilla/5.0", msg.UserAgent)
	assert.Equal(t, "https://google.com", msg.Referer)
	assert.Equal(t, now, msg.AccessTime)
}

func TestAnalyticsResponse_Structure(t *testing.T) {
	resp := AnalyticsResponse{
		ShortCode: "ABCD",
		PV:        1000,
		UV:        500,
		TopSources: []SourceStat{
			{Source: "google", Count: 500},
			{Source: "direct", Count: 300},
		},
	}

	assert.Equal(t, "ABCD", resp.ShortCode)
	assert.Equal(t, int64(1000), resp.PV)
	assert.Equal(t, int64(500), resp.UV)
	assert.Len(t, resp.TopSources, 2)
	assert.Equal(t, "google", resp.TopSources[0].Source)
	assert.Equal(t, int64(500), resp.TopSources[0].Count)
}

func TestSourceStat_Structure(t *testing.T) {
	stat := SourceStat{
		Source: "google",
		Count:  100,
	}

	assert.Equal(t, "google", stat.Source)
	assert.Equal(t, int64(100), stat.Count)
}

func TestStats_Structure(t *testing.T) {
	stats := Stats{
		PV: 1000,
		UV: 500,
	}

	assert.Equal(t, int64(1000), stats.PV)
	assert.Equal(t, int64(500), stats.UV)
}
