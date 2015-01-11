package azure

import (
	"fmt"
	"log"
	"os"

	azure "github.com/MSOpenTech/azure-sdk-for-go"
)

type Config struct {
	PublishSettingsFile string
}

func (c *Config) loadAndValidate() error {
	if _, err := os.Stat(c.PublishSettingsFile); os.IsNotExist(err) {
		return fmt.Errorf(
			"Error loading Azure Publish Settings file '%s': %s",
			c.PublishSettingsFile,
			err)
	}

	log.Printf("[INFO] Importing Azure Publish Settings file...")
	err := azure.ImportPublishSettingsFile(c.PublishSettingsFile)
	if err != nil {
		return err
	}

	return nil
}
