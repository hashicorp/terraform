package file

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

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

	ctx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

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

	if err := copyFiles(ctx, comm, src, dst); err != nil {
		return err
	}
	return nil
}

func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {
	if !c.IsSet("source") && !c.IsSet("content") {
		es = append(es, fmt.Errorf("Must provide one of 'source' or 'content'"))
	}

	return ws, es
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
func copyFiles(ctx context.Context, comm communicator.Communicator, src, dst string) error {
	// Wait and retry until we establish the connection
	err := communicator.Retry(ctx, func() error {
		return comm.Connect(nil)
	})
	if err != nil {
		return err
	}

	// disconnect when the context is canceled, which will close this after
	// Apply as well.
	go func() {
		<-ctx.Done()
		comm.Disconnect()
	}()

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
