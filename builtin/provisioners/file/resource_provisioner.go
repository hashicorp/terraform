package file

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-homedir"
)

// ResourceProvisioner represents a file provisioner
type ResourceProvisioner struct {
	schema.Provisioner
}

func Provisioner() terraform.ResourceProvisioner {
	return &ResourceProvisioner{
		schema.Provisioner{Schema: map[string]*schema.Schema{
			"source": {
				Type:     schema.TypeString,
				Required: true,
			},
			"content": {
				Type:     schema.TypeString,
				Required: true,
			},
			"destination": {
				Type:     schema.TypeString,
				Required: true,
			},
		}},
	}
}

// Apply executes the file provisioner
func (p *ResourceProvisioner) Apply(
	o terraform.UIOutput,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) error {
	// Get a new communicator
	comm, err := communicator.New(s)
	if err != nil {
		return err
	}

	// Get the source
	src, deleteSource, err := p.getSrc(c)
	if err != nil {
		return err
	}
	if deleteSource {
		defer os.Remove(src)
	}

	// Get destination
	dRaw := c.Config["destination"]
	dst, ok := dRaw.(string)
	if !ok {
		return fmt.Errorf("Unsupported 'destination' type! Must be string.")
	}
	return p.copyFiles(comm, src, dst)
}

// Validate checks if the required arguments are configured
func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	numDst := 0
	numSrc := 0
	for name := range c.Raw {
		switch name {
		case "destination":
			numDst++
		case "source", "content":
			numSrc++
		default:
			es = append(es, fmt.Errorf("Unknown configuration '%s'", name))
		}
	}
	if numSrc != 1 || numDst != 1 {
		es = append(es, fmt.Errorf("Must provide one  of 'content' or 'source' and 'destination' to file"))
	}
	return
}

// getSrc returns the file to use as source
func (p *ResourceProvisioner) getSrc(c *terraform.ResourceConfig) (string, bool, error) {
	var src string

	sRaw, ok := c.Config["source"]
	if ok {
		if src, ok = sRaw.(string); !ok {
			return "", false, fmt.Errorf("Unsupported 'source' type! Must be string.")
		}
	}

	content, ok := c.Config["content"]
	if ok {
		file, err := ioutil.TempFile("", "tf-file-content")
		if err != nil {
			return "", true, err
		}

		contentStr, ok := content.(string)
		if !ok {
			return "", true, fmt.Errorf("Unsupported 'content' type! Must be string.")
		}
		if _, err = file.WriteString(contentStr); err != nil {
			return "", true, err
		}

		return file.Name(), true, nil
	}

	expansion, err := homedir.Expand(src)
	return expansion, false, err
}

// copyFiles is used to copy the files from a source to a destination
func (p *ResourceProvisioner) copyFiles(comm communicator.Communicator, src, dst string) error {
	// Wait and retry until we establish the connection
	err := retryFunc(comm.Timeout(), func() error {
		err := comm.Connect(nil)
		return err
	})
	if err != nil {
		return err
	}
	defer comm.Disconnect()

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	// If we're uploading a directory, short circuit and do that
	if info.IsDir() {
		if err := comm.UploadDir(dst, src); err != nil {
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
