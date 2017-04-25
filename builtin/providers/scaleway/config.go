package scaleway

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/scaleway/scaleway-cli/pkg/api"
	"github.com/scaleway/scaleway-cli/pkg/scwversion"
)

// Config contains scaleway configuration values
type Config struct {
	Organization string
	APIKey       string
	Region       string
}

// Client contains scaleway api clients
type Client struct {
	scaleway *api.ScalewayAPI
}

// Client configures and returns a fully initialized Scaleway client
func (c *Config) Client() (*Client, error) {
	api, err := api.NewScalewayAPI(
		c.Organization,
		c.APIKey,
		scwversion.UserAgent(),
		c.Region,
		func(s *api.ScalewayAPI) {
			s.Logger = newTerraformLogger()
		},
	)
	if err != nil {
		return nil, err
	}
	return &Client{api}, nil
}

func newTerraformLogger() api.Logger {
	return &terraformLogger{}
}

type terraformLogger struct {
}

func (l *terraformLogger) LogHTTP(r *http.Request) {
	log.Printf("[DEBUG] %s %s\n", r.Method, r.URL.Path)
}
func (l *terraformLogger) Fatalf(format string, v ...interface{}) {
	log.Printf("[FATAL] %s\n", fmt.Sprintf(format, v))
	os.Exit(1)
}
func (l *terraformLogger) Debugf(format string, v ...interface{}) {
	log.Printf("[DEBUG] %s\n", fmt.Sprintf(format, v))
}
func (l *terraformLogger) Infof(format string, v ...interface{}) {
	log.Printf("[INFO ] %s\n", fmt.Sprintf(format, v))
}
func (l *terraformLogger) Warnf(format string, v ...interface{}) {
	log.Printf("[WARN ] %s\n", fmt.Sprintf(format, v))
}
