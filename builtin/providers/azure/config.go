package azure

import (
	"fmt"
	"os"

	"github.com/svanharmelen/azure-sdk-for-go/management"
)

// Config is the configuration structure used to instantiate a
// new Azure management client.
type Config struct {
	SettingsFile   string
	SubscriptionID string
}

// NewClient returns a new Azure management client
func (c *Config) NewClient() (management.Client, error) {
	if _, err := os.Stat(c.SettingsFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("Publish Settings file %q does not exist!", c.SettingsFile)
	}

	mc, err := management.ClientFromPublishSettingsFile(c.SettingsFile, c.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("Error creating management client: %s", err)
	}

	return mc, nil
}
