package file

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/terraform/helper/config"
	helper "github.com/hashicorp/terraform/helper/ssh"
	"github.com/hashicorp/terraform/terraform"
)

type ResourceProvisioner struct{}

func (p *ResourceProvisioner) Apply(
	o terraform.UIOutput,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) error {
	// Ensure the connection type is SSH
	if err := helper.VerifySSH(s); err != nil {
		return err
	}

	// Get the SSH configuration
	conf, err := helper.ParseSSHConfig(s)
	if err != nil {
		return err
	}

	// Get the source and destination
	sRaw := c.Config["source"]
	src, ok := sRaw.(string)
	if !ok {
		return fmt.Errorf("Unsupported 'source' type! Must be string.")
	}

	dRaw := c.Config["destination"]
	dst, ok := dRaw.(string)
	if !ok {
		return fmt.Errorf("Unsupported 'destination' type! Must be string.")
	}
	return p.copyFiles(conf, src, dst)
}

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	v := &config.Validator{
		Required: []string{
			"source",
			"destination",
		},
	}
	return v.Validate(c)
}

// copyFiles is used to copy the files from a source to a destination
func (p *ResourceProvisioner) copyFiles(conf *helper.SSHConfig, src, dst string) error {
	// Get the SSH client config
	config, err := helper.PrepareConfig(conf)
	if err != nil {
		return err
	}

	// Wait and retry until we establish the SSH connection
	var comm *helper.SSHCommunicator
	err = retryFunc(conf.TimeoutVal, func() error {
		host := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
		comm, err = helper.New(host, config)
		return err
	})
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	// If we're uploading a directory, short circuit and do that
	if info.IsDir() {
		if err := comm.UploadDir(dst, src, nil); err != nil {
			return fmt.Errorf("Upload failed: %v", err)
		}
		return nil
	}

	// We're uploading a file...
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	err = comm.Upload(dst, f)
	if err != nil {
		return fmt.Errorf("Upload failed: %v", err)
	}
	return err
}

// retryFunc is used to retry a function for a given duration
func retryFunc(timeout time.Duration, f func() error) error {
	finish := time.After(timeout)
	for {
		err := f()
		if err == nil {
			return nil
		}
		log.Printf("Retryable error: %v", err)

		select {
		case <-finish:
			return err
		case <-time.After(3 * time.Second):
		}
	}
}
