package cobbler

import (
	"net/http"

	cobbler "github.com/jtopjian/cobblerclient"
)

type Config struct {
	Url      string
	Username string
	Password string

	cobblerClient cobbler.Client
}

func (c *Config) loadAndValidate() error {
	config := cobbler.ClientConfig{
		Url:      c.Url,
		Username: c.Username,
		Password: c.Password,
	}

	client := cobbler.NewClient(http.DefaultClient, config)
	_, err := client.Login()
	if err != nil {
		return err
	}

	c.cobblerClient = client

	return nil
}
