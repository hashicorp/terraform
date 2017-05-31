package aws

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
func sharedClientForRegion(region string) (interface{}, error) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		return nil, fmt.Errorf("empty AWS_ACCESS_KEY_ID")
	}

	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return nil, fmt.Errorf("empty AWS_SECRET_ACCESS_KEY")
	}

	conf := &Config{
		Region: region,
	}

	client, err := conf.Client()
	if err != nil {
		return nil, fmt.Errorf("error getting AWS client")
	}

	return client, nil
}
