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
	if c.BaseURL != "" {
		err := client.SetBaseURL(c.BaseURL)
		if err != nil {
			// The BaseURL supplied wasn't valid, bail.
			return nil, err
		}
	}

	// Test the credentials by checking we can get information about the authenticated user.
	_, _, err := client.Users.CurrentUser()
	if err != nil {
		return nil, err
	}

	return client, nil
}
