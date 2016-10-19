package gitlab

import (
	"github.com/xanzy/go-gitlab"
)

// Config is per-provider, specifies where to connect to gitlab
type Config struct {
	Token   string
	BaseURL string
}

// Client returns a *gitlab.Client to interact with the configured gitlab instance
func (c *Config) Client() (interface{}, error) {
	client := gitlab.NewClient(nil, c.Token)
	client.SetBaseURL(c.BaseURL)
	return client, nil
}
