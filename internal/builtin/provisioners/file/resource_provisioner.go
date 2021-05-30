package file

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/terraform/internal/communicator"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/mitchellh/go-homedir"
	"github.com/zclconf/go-cty/cty"
)

func New() provisioners.Interface {
	ctx, cancel := context.WithCancel(context.Background())
	return &provisioner{
		ctx:    ctx,
		cancel: cancel,
	}
}

type provisioner struct {
	// We store a context here tied to the lifetime of the provisioner.
	// This allows the Stop method to cancel any in-flight requests.
	ctx    context.Context
	cancel context.CancelFunc
}

func (p *provisioner) GetSchema() (resp provisioners.GetSchemaResponse) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"source": {
				Type:     cty.String,
				Optional: true,
			},

			"content": {
				Type:     cty.String,
				Optional: true,
			},

			"destination": {
				Type:     cty.String,
				Required: true,
			},
		},
	}
	resp.Provisioner = schema
	return resp
}

func (p *provisioner) ValidateProvisionerConfig(req provisioners.ValidateProvisionerConfigRequest) (resp provisioners.ValidateProvisionerConfigResponse) {
	cfg, err := p.GetSchema().Provisioner.CoerceValue(req.Config)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
	}

	source := cfg.GetAttr("source")
	content := cfg.GetAttr("content")

	switch {
	case !source.IsNull() && !content.IsNull():
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("Cannot set both 'source' and 'content'"))
		return resp
	case source.IsNull() && content.IsNull():
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("Must provide one of 'source' or 'content'"))
		return resp
	}

	return resp
}

func (p *provisioner) ProvisionResource(req provisioners.ProvisionResourceRequest) (resp provisioners.ProvisionResourceResponse) {
	if req.Connection.IsNull() {
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"file provisioner error",
			"Missing connection configuration for provisioner.",
		))
		return resp
	}

	comm, err := communicator.New(req.Connection)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"file provisioner error",
			err.Error(),
		))
		return resp
	}

	// Get the source
	src, deleteSource, err := getSrc(req.Config)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"file provisioner error",
			err.Error(),
		))
		return resp
	}
	if deleteSource {
		defer os.Remove(src)
	}

	// Begin the file copy
	dst := req.Config.GetAttr("destination").AsString()
	if err := copyFiles(p.ctx, comm, src, dst); err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.WholeContainingBody(
			tfdiags.Error,
			"file provisioner error",
			err.Error(),
		))
		return resp
	}

	return resp
}

// getSrc returns the file to use as source
func getSrc(v cty.Value) (string, bool, error) {
	content := v.GetAttr("content")
	src := v.GetAttr("source")

	switch {
	case !content.IsNull():
		file, err := ioutil.TempFile("", "tf-file-content")
		if err != nil {
			return "", true, err
		}

		if _, err = file.WriteString(content.AsString()); err != nil {
			return "", true, err
		}

		return file.Name(), true, nil

	case !src.IsNull():
		expansion, err := homedir.Expand(src.AsString())
		return expansion, false, err

	default:
		panic("source and content cannot both be null")
	}
}

// copyFiles is used to copy the files from a source to a destination
func copyFiles(ctx context.Context, comm communicator.Communicator, src, dst string) error {
	retryCtx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	// Wait and retry until we establish the connection
	err := communicator.Retry(retryCtx, func() error {
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

func (p *provisioner) Stop() error {
	p.cancel()
	return nil
}

func (p *provisioner) Close() error {
	return nil
}
