package file

import (
	"context"
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

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"source": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"content"},
			},

			"content": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"source"},
			},

			"destination": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},

		ApplyFunc:    applyFn,
		ValidateFunc: validateFn,
	}
}

func applyFn(ctx context.Context) error {
	connState := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	data := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)

	// Get a new communicator
	comm, err := communicator.New(connState)
	if err != nil {
		return err
	}

	// Get the source
	src, deleteSource, err := getSrc(data)
	if err != nil {
		return err
	}
	if deleteSource {
		defer os.Remove(src)
	}

	// Begin the file copy
	dst := data.Get("destination").(string)
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- copyFiles(comm, src, dst)
	}()

	// Allow the file copy to complete unless there is an interrupt.
	// If there is an interrupt we make no attempt to cleanly close
	// the connection currently. We just abruptly exit. Because Terraform
	// taints the resource, this is fine.
	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("file transfer interrupted")
	}
}

func validateFn(d *schema.ResourceData) (ws []string, es []error) {
	numSrc := 0
	if _, ok := d.GetOk("source"); ok == true {
		numSrc++
	}
	if _, ok := d.GetOk("content"); ok == true {
		numSrc++
	}
	if numSrc != 1 {
		es = append(es, fmt.Errorf("Must provide one of 'content' or 'source'"))
	}
	return
}

// getSrc returns the file to use as source
func getSrc(data *schema.ResourceData) (string, bool, error) {
	src := data.Get("source").(string)
	if content, ok := data.GetOk("content"); ok {
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
