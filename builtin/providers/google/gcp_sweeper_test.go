package google

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

// sharedCredsForRegion returns a common config setup needed for the sweeper
// functions for a given region
func sharedCredsForRegion(region string) (*Config, error) {
	project := os.Getenv("GOOGLE_PROJECT")
	if project == "" {
		return nil, fmt.Errorf("empty GOOGLE_PROJECT")
	}

	creds := os.Getenv("GOOGLE_CREDENTIALS")
	if project == "" {
		return nil, fmt.Errorf("empty GOOGLE_CREDENTIALS")
	}

	conf := &Config{
		Credentials: creds,
		Region:      region,
		Project:     project,
	}

	return conf, nil
}
