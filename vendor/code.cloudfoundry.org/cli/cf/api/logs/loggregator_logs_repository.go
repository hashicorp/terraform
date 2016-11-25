package logs

import (
	"errors"
	"time"

	. "code.cloudfoundry.org/cli/cf/i18n"

	"code.cloudfoundry.org/cli/cf/api/authentication"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	consumer "github.com/cloudfoundry/loggregator_consumer"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	noaa_errors "github.com/cloudfoundry/noaa/errors"
)

type LoggregatorLogsRepository struct {
	consumer       consumer.LoggregatorConsumer
	config         coreconfig.Reader
	tokenRefresher authentication.TokenRefresher
	messageQueue   *LoggregatorMessageQueue
	BufferTime     time.Duration
}

func NewLoggregatorLogsRepository(config coreconfig.Reader, consumer consumer.LoggregatorConsumer, refresher authentication.TokenRefresher) *LoggregatorLogsRepository {
	return &LoggregatorLogsRepository{
		config:         config,
		consumer:       consumer,
		tokenRefresher: refresher,
		messageQueue:   NewLoggregatorMessageQueue(),
		BufferTime:     defaultBufferTime,
	}
}

func (repo *LoggregatorLogsRepository) Close() {
	_ = repo.consumer.Close()
}

func loggableMessagesFromLoggregatorMessages(messages []*logmessage.LogMessage) []Loggable {
	loggableMessages := make([]Loggable, len(messages))

	for i, m := range messages {
		loggableMessages[i] = NewLoggregatorLogMessage(m)
	}

	return loggableMessages
}

func (repo *LoggregatorLogsRepository) RecentLogsFor(appGUID string) ([]Loggable, error) {
	messages, err := repo.consumer.Recent(appGUID, repo.config.AccessToken())

	switch err.(type) {
	case nil: // do nothing
	case *noaa_errors.UnauthorizedError:
		_, _ = repo.tokenRefresher.RefreshAuthToken()
		return repo.RecentLogsFor(appGUID)
	default:
		return loggableMessagesFromLoggregatorMessages(messages), err
	}

	consumer.SortRecent(messages)

	return loggableMessagesFromLoggregatorMessages(messages), nil
}

func (repo *LoggregatorLogsRepository) TailLogsFor(appGUID string, onConnect func(), logChan chan<- Loggable, errChan chan<- error) {
	ticker := time.NewTicker(repo.BufferTime)
	endpoint := repo.config.LoggregatorEndpoint()
	if endpoint == "" {
		errChan <- errors.New(T("Loggregator endpoint missing from config file"))
		return
	}

	repo.consumer.SetOnConnectCallback(onConnect)
	c, err := repo.consumer.Tail(appGUID, repo.config.AccessToken())

	switch err.(type) {
	case nil: // do nothing
	case *noaa_errors.UnauthorizedError:
		_, _ = repo.tokenRefresher.RefreshAuthToken()
		c, err = repo.consumer.Tail(appGUID, repo.config.AccessToken())
	default:
		errChan <- err
		return
	}

	if err != nil {
		errChan <- err
		return
	}

	go func() {
		for range ticker.C {
			repo.flushMessages(logChan)
		}
	}()

	go func() {
		for msg := range c {
			repo.messageQueue.PushMessage(msg)
		}

		repo.flushMessages(logChan)
		close(logChan)
	}()
}

func (repo *LoggregatorLogsRepository) flushMessages(c chan<- Loggable) {
	repo.messageQueue.EnumerateAndClear(func(m *logmessage.LogMessage) {
		c <- NewLoggregatorLogMessage(m)
	})
}
