package logs

import (
	"errors"
	"fmt"
	"time"

	. "code.cloudfoundry.org/cli/cf/i18n"

	"code.cloudfoundry.org/cli/cf/api/authentication"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"

	"github.com/cloudfoundry/noaa"
	noaaerrors "github.com/cloudfoundry/noaa/errors"
	"github.com/cloudfoundry/sonde-go/events"
)

type NoaaLogsRepository struct {
	config         coreconfig.Reader
	consumer       NoaaConsumer
	tokenRefresher authentication.TokenRefresher
	messageQueue   *NoaaMessageQueue
	BufferTime     time.Duration
	retryTimeout   time.Duration
}

func NewNoaaLogsRepository(config coreconfig.Reader, consumer NoaaConsumer, tr authentication.TokenRefresher, retryTimeout time.Duration) *NoaaLogsRepository {
	consumer.RefreshTokenFrom(tr)
	return &NoaaLogsRepository{
		config:         config,
		consumer:       consumer,
		tokenRefresher: tr,
		messageQueue:   NewNoaaMessageQueue(),
		BufferTime:     defaultBufferTime,
		retryTimeout:   retryTimeout,
	}
}

func (repo *NoaaLogsRepository) Close() {
	_ = repo.consumer.Close()
}

func loggableMessagesFromNoaaMessages(messages []*events.LogMessage) []Loggable {
	loggableMessages := make([]Loggable, len(messages))

	for i, m := range messages {
		loggableMessages[i] = NewNoaaLogMessage(m)
	}

	return loggableMessages
}

func (repo *NoaaLogsRepository) RecentLogsFor(appGUID string) ([]Loggable, error) {
	logs, err := repo.consumer.RecentLogs(appGUID, repo.config.AccessToken())

	if err != nil {
		return loggableMessagesFromNoaaMessages(logs), err
	}
	return loggableMessagesFromNoaaMessages(noaa.SortRecent(logs)), err
}

func (repo *NoaaLogsRepository) TailLogsFor(appGUID string, onConnect func(), logChan chan<- Loggable, errChan chan<- error) {
	ticker := time.NewTicker(repo.BufferTime)
	retryTimer := newUnstartedTimer()

	endpoint := repo.config.DopplerEndpoint()
	if endpoint == "" {
		errChan <- errors.New(T("Loggregator endpoint missing from config file"))
		return
	}

	repo.consumer.SetOnConnectCallback(func() {
		retryTimer.Stop()
		onConnect()
	})
	c, e := repo.consumer.TailingLogs(appGUID, repo.config.AccessToken())

	go func() {
		defer close(logChan)
		defer close(errChan)

		timerRunning := false
		for {
			select {
			case msg, ok := <-c:
				if !ok {
					ticker.Stop()
					repo.flushMessages(logChan)
					return
				}
				timerRunning = false
				repo.messageQueue.PushMessage(msg)
			case err := <-e:
				if err != nil {
					if _, ok := err.(noaaerrors.RetryError); ok {
						if !timerRunning {
							timerRunning = true
							retryTimer.Reset(repo.retryTimeout)
						}
						continue
					}

					errChan <- err

					ticker.Stop()
					return
				}
			case <-retryTimer.C:
				errChan <- fmt.Errorf("Timed out waiting for connection to Loggregator (%s).", repo.config.DopplerEndpoint())
				ticker.Stop()
				return
			}
		}
	}()

	go func() {
		for range ticker.C {
			repo.flushMessages(logChan)
		}
	}()
}

func (repo *NoaaLogsRepository) flushMessages(c chan<- Loggable) {
	repo.messageQueue.EnumerateAndClear(func(m *events.LogMessage) {
		c <- NewNoaaLogMessage(m)
	})
}

// newUnstartedTimer returns a *time.Timer that is in an unstarted
// state.
func newUnstartedTimer() *time.Timer {
	timer := time.NewTimer(time.Second)
	timer.Stop()
	return timer
}
