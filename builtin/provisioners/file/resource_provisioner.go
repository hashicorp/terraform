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

func ResourceProvisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"source": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"content"},
			},
			"content": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"source"},
			},
			"destination": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
		ApplyFunc: Apply,
		//ValidateFunc: Validate,
	}
}

// Apply executes the file provisioner
func Apply(
	o terraform.UIOutput,
	d *schema.ResourceData) error {
	// Get a new communicator
	comm, err := communicator.New(d.State())
	if err != nil {
		return err
	}

	// Get the source
	src, deleteSource, err := getSrc(d)
	if err != nil {
		return err
	}
	if deleteSource {
		defer os.Remove(src)
	}

	// Get destination
	dst := d.Get("destination").(string)
	return copyFiles(comm, src, dst)
}

// Validate checks if the required arguments are configured
func Validate(d *schema.ResourceData) (ws []string, es []error) {
	numSrc := 0
	if _, ok := d.GetOk("source"); ok == true {
		numSrc++
	}
	if _, ok := d.GetOk("content"); ok == true {
		numSrc++
	}
	if numSrc != 1 {
		es = append(es, fmt.Errorf("Must provide one of 'content' or 'source' and 'destination' to file"))
	}
	return
}

// getSrc returns the file to use as source
func getSrc(d *schema.ResourceData) (string, bool, error) {
	var src string

	source, ok := d.GetOk("source")
	if ok {
		src = source.(string)
	}

	content, ok := d.GetOk("content")
	if ok {
		file, err := ioutil.TempFile("", "tf-file-content")
		if err != nil {
			return "", true, err
		}

		if _, err = file.WriteString(content.(string)); err != nil {
			return "", true, err
		}

		return file.Name(), true, nil
	}

	expansion, err := homedir.Expand(src)
	return expansion, false, err
}

// copyFiles is used to copy the files from a source to a destination
func copyFiles(comm communicator.Communicator, src, dst string) error {
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
