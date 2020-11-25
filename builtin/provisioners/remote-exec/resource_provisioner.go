package remoteexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/mitchellh/go-linereader"
	"github.com/zclconf/go-cty/cty"
)

func New() provisioners.Interface {
	return &provisioner{}
}

type provisioner struct {
	// this stored from the running context, so that Stop() can cancel the
	// command
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (p *provisioner) GetSchema() (resp provisioners.GetSchemaResponse) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"inline": {
				Type:     cty.List(cty.String),
				Optional: true,
			},
			"script": {
				Type:     cty.String,
				Optional: true,
			},
			"scripts": {
				Type:     cty.List(cty.String),
				Optional: true,
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
		return resp
	}

	inline := cfg.GetAttr("inline")
	script := cfg.GetAttr("script")
	scripts := cfg.GetAttr("scripts")

	set := 0
	if !inline.IsNull() {
		set++
	}
	if !script.IsNull() {
		set++
	}
	if !scripts.IsNull() {
		set++
	}
	if set != 1 {
		resp.Diagnostics = resp.Diagnostics.Append(errors.New(
			`only one of "inline", "script", or "scripts" must be set`))
	}
	return resp
}

func (p *provisioner) ProvisionResource(req provisioners.ProvisionResourceRequest) (resp provisioners.ProvisionResourceResponse) {
	p.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.mu.Unlock()

	comm, err := communicator.New(req.Connection)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	// Collect the scripts
	scripts, err := collectScripts(req.Config)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	for _, s := range scripts {
		defer s.Close()
	}

	// Copy and execute each script
	if err := runScripts(ctx, req.UIOutput, comm, scripts); err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	return resp
}

func (p *provisioner) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cancel()
	return nil
}

func (p *provisioner) Close() error {
	return nil
}

// generateScripts takes the configuration and creates a script from each inline config
func generateScripts(inline cty.Value) ([]string, error) {
	var lines []string
	for _, l := range inline.AsValueSlice() {
		s := l.AsString()
		if s == "" {
			return nil, errors.New("invalid empty string in 'scripts'")
		}
		lines = append(lines, s)
	}
	lines = append(lines, "")

	return []string{strings.Join(lines, "\n")}, nil
}

// collectScripts is used to collect all the scripts we need
// to execute in preparation for copying them.
func collectScripts(v cty.Value) ([]io.ReadCloser, error) {
	// Check if inline
	if inline := v.GetAttr("inline"); !inline.IsNull() {
		scripts, err := generateScripts(inline)
		if err != nil {
			return nil, err
		}

		var r []io.ReadCloser
		for _, script := range scripts {
			r = append(r, ioutil.NopCloser(bytes.NewReader([]byte(script))))
		}

		return r, nil
	}

	// Collect scripts
	var scripts []string
	if script := v.GetAttr("script"); !script.IsNull() {
		s := script.AsString()
		if s == "" {
			return nil, errors.New("invalid empty string in 'script'")
		}
		scripts = append(scripts, s)
	}

	if scriptList := v.GetAttr("scripts"); !scriptList.IsNull() {
		for _, script := range scriptList.AsValueSlice() {
			s := script.AsString()
			if s == "" {
				return nil, errors.New("invalid empty string in 'script'")
			}
			scripts = append(scripts, script.AsString())
		}
	}

	// Open all the scripts
	var fhs []io.ReadCloser
	for _, s := range scripts {
		fh, err := os.Open(s)
		if err != nil {
			for _, fh := range fhs {
				fh.Close()
			}
			return nil, fmt.Errorf("Failed to open script '%s': %v", s, err)
		}
		fhs = append(fhs, fh)
	}

	// Done, return the file handles
	return fhs, nil
}

// runScripts is used to copy and execute a set of scripts
func runScripts(ctx context.Context, o provisioners.UIOutput, comm communicator.Communicator, scripts []io.ReadCloser) error {
	retryCtx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	// Wait and retry until we establish the connection
	err := communicator.Retry(retryCtx, func() error {
		return comm.Connect(o)
	})
	if err != nil {
		return err
	}

	// Wait for the context to end and then disconnect
	go func() {
		<-ctx.Done()
		comm.Disconnect()
	}()

	for _, script := range scripts {
		var cmd *remote.Cmd

		outR, outW := io.Pipe()
		errR, errW := io.Pipe()
		defer outW.Close()
		defer errW.Close()

		go copyUIOutput(o, outR)
		go copyUIOutput(o, errR)

		remotePath := comm.ScriptPath()

		if err := comm.UploadScript(remotePath, script); err != nil {
			return fmt.Errorf("Failed to upload script: %v", err)
		}

		cmd = &remote.Cmd{
			Command: remotePath,
			Stdout:  outW,
			Stderr:  errW,
		}
		if err := comm.Start(cmd); err != nil {
			return fmt.Errorf("Error starting script: %v", err)
		}

		if err := cmd.Wait(); err != nil {
			return err
		}

		// Upload a blank follow up file in the same path to prevent residual
		// script contents from remaining on remote machine
		empty := bytes.NewReader([]byte(""))
		if err := comm.Upload(remotePath, empty); err != nil {
			// This feature is best-effort.
			log.Printf("[WARN] Failed to upload empty follow up script: %v", err)
		}
	}

	return nil
}

func copyUIOutput(o provisioners.UIOutput, r io.Reader) {
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}
