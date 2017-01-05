package logs

import (
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
)

// Should be satisfied automatically by *noaa.Consumer
//go:generate counterfeiter . NoaaConsumer

type NoaaConsumer interface {
	TailingLogs(string, string) (<-chan *events.LogMessage, <-chan error)
	RecentLogs(appGUID string, authToken string) ([]*events.LogMessage, error)
	Close() error
	SetOnConnectCallback(cb func())
	RefreshTokenFrom(tr consumer.TokenRefresher)
}
