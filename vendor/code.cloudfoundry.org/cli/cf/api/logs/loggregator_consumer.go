package logs

import (
	"github.com/cloudfoundry/loggregator_consumer"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
)

//go:generate counterfeiter . LoggregatorConsumer

type LoggregatorConsumer interface {
	Tail(appGUID string, authToken string) (<-chan *logmessage.LogMessage, error)
	Recent(appGUID string, authToken string) ([]*logmessage.LogMessage, error)
	Close() error
	SetOnConnectCallback(func())
	SetDebugPrinter(loggregator_consumer.DebugPrinter)
}
