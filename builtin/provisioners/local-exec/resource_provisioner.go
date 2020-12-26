package localexec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/armon/circbuf"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/mitchellh/go-linereader"
	"github.com/zclconf/go-cty/cty"
)

const (
	// maxBufSize limits how much output we collect from a local
	// invocation. This is to prevent TF memory usage from growing
	// to an enormous amount due to a faulty process.
	maxBufSize = 8 * 1024
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
			"command": {
				Type:     cty.String,
				Required: true,
			},
			"interpreter": {
				Type:     cty.List(cty.String),
				Optional: true,
			},
			"working_dir": {
				Type:     cty.String,
				Optional: true,
			},
			"environment": {
				Type:     cty.Map(cty.String),
				Optional: true,
			},
		},
	}

	resp.Provisioner = schema
	return resp
}

func (p *provisioner) ValidateProvisionerConfig(req provisioners.ValidateProvisionerConfigRequest) (resp provisioners.ValidateProvisionerConfigResponse) {
	if _, err := p.GetSchema().Provisioner.CoerceValue(req.Config); err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
	}
	return resp
}

func (p *provisioner) ProvisionResource(req provisioners.ProvisionResourceRequest) (resp provisioners.ProvisionResourceResponse) {
	p.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.mu.Unlock()

	command := req.Config.GetAttr("command").AsString()
	if command == "" {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("local-exec provisioner command must be a non-empty string"))
		return resp
	}

	envVal := req.Config.GetAttr("environment")
	var env []string

	if !envVal.IsNull() {
		for k, v := range envVal.AsValueMap() {
			entry := fmt.Sprintf("%s=%s", k, v.AsString())
			env = append(env, entry)
		}
	}

	// Execute the command using a shell
	intrVal := req.Config.GetAttr("interpreter")

	var cmdargs []string
	if !intrVal.IsNull() && intrVal.LengthInt() > 0 {
		for _, v := range intrVal.AsValueSlice() {
			cmdargs = append(cmdargs, v.AsString())
		}
	} else {
		if runtime.GOOS == "windows" {
			cmdargs = []string{"cmd", "/C"}
		} else {
			cmdargs = []string{"/bin/sh", "-c"}
		}
	}

	cmdargs = append(cmdargs, command)

	workingdir := ""
	if wdVal := req.Config.GetAttr("working_dir"); !wdVal.IsNull() {
		workingdir = wdVal.AsString()
	}

	// Setup the reader that will read the output from the command.
	// We use an os.Pipe so that the *os.File can be passed directly to the
	// process, and not rely on goroutines copying the data which may block.
	// See golang.org/issue/18874
	pr, pw, err := os.Pipe()
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("failed to initialize pipe for output: %s", err))
		return resp
	}

	var cmdEnv []string
	cmdEnv = os.Environ()
	cmdEnv = append(cmdEnv, env...)

	// Setup the command
	cmd := exec.CommandContext(ctx, cmdargs[0], cmdargs[1:]...)
	cmd.Stderr = pw
	cmd.Stdout = pw
	// Dir specifies the working directory of the command.
	// If Dir is the empty string (this is default), runs the command
	// in the calling process's current directory.
	cmd.Dir = workingdir
	// Env specifies the environment of the command.
	// By default will use the calling process's environment
	cmd.Env = cmdEnv

	output, _ := circbuf.NewBuffer(maxBufSize)

	// Write everything we read from the pipe to the output buffer too
	tee := io.TeeReader(pr, output)

	// copy the teed output to the UI output
	copyDoneCh := make(chan struct{})
	go copyUIOutput(req.UIOutput, tee, copyDoneCh)

	// Output what we're about to run
	req.UIOutput.Output(fmt.Sprintf("Executing: %q", cmdargs))

	// Start the command
	err = cmd.Start()
	if err == nil {
		err = cmd.Wait()
	}

	// Close the write-end of the pipe so that the goroutine mirroring output
	// ends properly.
	pw.Close()

	// Cancelling the command may block the pipe reader if the file descriptor
	// was passed to a child process which hasn't closed it. In this case the
	// copyOutput goroutine will just hang out until exit.
	select {
	case <-copyDoneCh:
	case <-ctx.Done():
	}

	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("Error running command '%s': %v. Output: %s",
			command, err, output.Bytes()))
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

func copyUIOutput(o provisioners.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}
