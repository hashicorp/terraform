package cloudfoundry

import (
	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
)

// Config -
type Config struct {
	endpoint          string
	User              string
	Password          string
	UaaClientID       string
	UaaClientSecret   string
	CACert            string
	SkipSslValidation bool
}

// Client - Terraform providor client initialization
func (c *Config) Client() (*cfapi.Session, error) {
	return cfapi.NewSession(c.endpoint, c.User, c.Password, c.UaaClientID, c.UaaClientSecret, c.CACert, c.SkipSslValidation)
}
