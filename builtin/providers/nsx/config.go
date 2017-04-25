package nsx

import (
	"github.com/sky-uk/gonsx"
	"log"
)

// Config is a struct for containing the provider parameters.
type Config struct {
	Debug       bool
	Insecure    bool
	NSXUser     string
	NSXPassword string
	NSXServer   string
}

// Client returns a new client for accessing VMWare vSphere.
func (c *Config) Client() (*gonsx.NSXClient, error) {
	log.Printf("[INFO] VMWare NSX Client configured for URL: %s", c.NSXServer)
	nsxclient := gonsx.NewNSXClient("https://"+c.NSXServer, c.NSXUser, c.NSXPassword, c.Insecure, c.Debug)
	return nsxclient, nil
}
