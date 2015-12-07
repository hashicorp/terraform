package vsphere

import (
	"fmt"
	"log"
	"net/url"

	"github.com/vmware/govmomi"
	"golang.org/x/net/context"
)

type Config struct {
	User          string
	Password      string
	VSphereServer string
	InsecureFlag  bool
}

// Client() returns a new client for accessing VMWare vSphere.
func (c *Config) Client() (*govmomi.Client, error) {
	u, err := url.Parse("https://" + c.VSphereServer + "/sdk")
	if err != nil {
		return nil, fmt.Errorf("Error parse url: %s", err)
	}

	u.User = url.UserPassword(c.User, c.Password)

	client, err := govmomi.NewClient(context.TODO(), u, c.InsecureFlag)
	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	log.Printf("[INFO] VMWare vSphere Client configured for URL: %s", u)

	return client, nil
}
