package retrystrategy

type RetryStrategy interface {
	Try() error
}

type Retryable interface {
	Attempt() (bool, error)
}

type retryable struct {
	attemptFunc func() (bool, error)
}

func (r *retryable) Attempt() (bool, error) {
	return r.attemptFunc()
}

func NewRetryable(attemptFunc func() (bool, error)) Retryable {
	return &retryable{
		attemptFunc: attemptFunc,
	}
}
