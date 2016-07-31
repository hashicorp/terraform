package vsphere

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/debug"
	"golang.org/x/net/context"
)

type Config struct {
	User          string
	Password      string
	VSphereServer string
	InsecureFlag  bool
	Debug         bool
	DebugPath     string
	DebugPathRun  string
}

// Client() returns a new client for accessing VMWare vSphere.
func (c *Config) Client() (*govmomi.Client, error) {
	u, err := url.Parse("https://" + c.VSphereServer + "/sdk")
	if err != nil {
		return nil, fmt.Errorf("Error parse url: %s", err)
	}

	u.User = url.UserPassword(c.User, c.Password)

	err = c.EnableDebug()
	if err != nil {
		return nil, fmt.Errorf("Error setting up client debug: %s", err)
	}

	client, err := govmomi.NewClient(context.TODO(), u, c.InsecureFlag)
	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	log.Printf("[INFO] VMWare vSphere Client configured for URL: %s", c.VSphereServer)

	return client, nil
}

func (c *Config) EnableDebug() error {
	if !c.Debug {
		return nil
	}

	// Base path for storing debug logs.
	r := c.DebugPath
	if r == "" {
		r = filepath.Join(os.Getenv("HOME"), ".govmomi")
	}
	r = filepath.Join(r, "debug")

	// Path for this particular run.
	run := c.DebugPathRun
	if run == "" {
		now := time.Now().Format("2006-01-02T15-04-05.999999999")
		r = filepath.Join(r, now)
	} else {
		// reuse the same path
		r = filepath.Join(r, run)
		_ = os.RemoveAll(r)
	}

	err := os.MkdirAll(r, 0700)
	if err != nil {
		log.Printf("[ERROR] Client debug setup failed: %v", err)
		return err
	}

	p := debug.FileProvider{
		Path: r,
	}

	debug.SetProvider(&p)
	return nil
}
