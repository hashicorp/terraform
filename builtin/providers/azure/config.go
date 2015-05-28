package azure

import (
	"fmt"
	"os"
	"sync"

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

// Client contains all the handles required for managing Azure services.
type Client struct {
	// unfortunately; because of how Azure's network API works; doing networking operations
	// concurrently is very hazardous, and we need a mutex to guard the management.Client.
	mutex      *sync.Mutex
	mgmtClient management.Client
}

// NewClientFromSettingsFile returns a new Azure management
// client created using a publish settings file.
func (c *Config) NewClientFromSettingsFile() (*Client, error) {
	if _, err := os.Stat(c.SettingsFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("Publish Settings file %q does not exist!", c.SettingsFile)
	}

	mc, err := management.ClientFromPublishSettingsFile(c.SettingsFile, c.SubscriptionID)
	if err != nil {
		return nil, nil
	}

	return &Client{
		mutex:      &sync.Mutex{},
		mgmtClient: mc,
	}, nil
}

// NewClient returns a new Azure management client created
// using a subscription ID and certificate.
func (c *Config) NewClient() (*Client, error) {
	mc, err := management.NewClient(c.SubscriptionID, c.Certificate)
	if err != nil {
		return nil, nil
	}

	return &Client{
		mutex:      &sync.Mutex{},
		mgmtClient: mc,
	}, nil
}
