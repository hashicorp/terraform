package fastly

import (
	"fmt"

	gofastly "github.com/sethvargo/go-fastly"
)

type Config struct {
	ApiKey string
}

type FastlyClient struct {
	conn *gofastly.Client
}

func (c *Config) Client() (interface{}, error) {
	var client FastlyClient

	if c.ApiKey == "" {
		return nil, fmt.Errorf("[Err] No API key for Fastly")
	}

	fconn, err := gofastly.NewClient(c.ApiKey)
	if err != nil {
		return nil, err
	}

	client.conn = fconn
	return &client, nil
}
