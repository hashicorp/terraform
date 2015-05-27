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
	Certificate    []byte
	ManagementURL  string
}

// NewClient returns a new Azure management client which is created
// using different functions depending on the supplied settings
func (c *Config) NewClient() (management.Client, error) {
	if c.SettingsFile != "" {
		if _, err := os.Stat(c.SettingsFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("Publish Settings file %q does not exist!", c.SettingsFile)
		}

		return management.ClientFromPublishSettingsFile(c.SettingsFile, c.SubscriptionID)
	}

	if c.ManagementURL != "" {
		return management.NewClientFromConfig(
			c.SubscriptionID,
			c.Certificate,
			management.ClientConfig{ManagementURL: c.ManagementURL},
		)
	}

	if c.SubscriptionID != "" && len(c.Certificate) > 0 {
		return management.NewClient(c.SubscriptionID, c.Certificate)
	}

	return nil, fmt.Errorf(
		"Insufficient configuration data. Please specify either a 'settings_file'\n" +
			"or both a 'subscription_id' and 'certificate' with an optional 'management_url'.")
}
