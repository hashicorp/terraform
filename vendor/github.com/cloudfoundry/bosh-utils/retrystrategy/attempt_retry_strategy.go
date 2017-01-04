package retrystrategy

import (
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type attemptRetryStrategy struct {
	maxAttempts int
	delay       time.Duration
	retryable   Retryable
	logger      boshlog.Logger
	logTag      string
}

func NewAttemptRetryStrategy(
	maxAttempts int,
	delay time.Duration,
	retryable Retryable,
	logger boshlog.Logger,
) RetryStrategy {
	return &attemptRetryStrategy{
		maxAttempts: maxAttempts,
		delay:       delay,
		retryable:   retryable,
		logger:      logger,
		logTag:      "attemptRetryStrategy",
	}
}

func (s *attemptRetryStrategy) Try() error {
	var err error
	var isRetryable bool

	for i := 0; i < s.maxAttempts; i++ {
		s.logger.Debug(s.logTag, "Making attempt #%d", i)

		isRetryable, err = s.retryable.Attempt()
		if err == nil {
			return nil
		}

		if !isRetryable {
			return err
		}

		time.Sleep(s.delay)
	}

	return err
}
