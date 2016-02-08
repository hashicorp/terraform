package flags

import "sync"

// Key type for storing flag instances in a context.Context.
type flagKey string

// Type to help flags out with only registering/processing once.
type common struct {
	register sync.Once
	process  sync.Once
}

func (c *common) RegisterOnce(fn func()) {
	c.register.Do(fn)
}

func (c *common) ProcessOnce(fn func() error) (err error) {
	c.process.Do(func() {
		err = fn()
	})
	return err
}
