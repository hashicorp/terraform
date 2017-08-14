package remoteexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"inline": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				PromoteSingle: true,
				Optional:      true,
				ConflictsWith: []string{"script", "scripts"},
			},

			"script": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"inline", "scripts"},
			},

			"scripts": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"script", "inline"},
			},
		},

		ApplyFunc: applyFn,
	}
}

// Apply executes the remote exec provisioner
func applyFn(ctx context.Context) error {
	connState := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	data := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)

	// Get a new communicator
	comm, err := communicator.New(connState)
	if err != nil {
		return err
	}

	// Collect the scripts
	scripts, err := collectScripts(data)
	if err != nil {
		return err
	}
	for _, s := range scripts {
		defer s.Close()
	}

	// Copy and execute each script
	if err := runScripts(ctx, o, comm, scripts); err != nil {
		return err
	}

	return nil
}

// generateScripts takes the configuration and creates a script from each inline config
func generateScripts(d *schema.ResourceData) ([]string, error) {
	var lines []string
	for _, l := range d.Get("inline").([]interface{}) {
		line, ok := l.(string)
		if !ok {
			return nil, fmt.Errorf("Error parsing %v as a string", l)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "")

	return []string{strings.Join(lines, "\n")}, nil
}

// collectScripts is used to collect all the scripts we need
// to execute in preparation for copying them.
func collectScripts(d *schema.ResourceData) ([]io.ReadCloser, error) {
	// Check if inline
	if _, ok := d.GetOk("inline"); ok {
		scripts, err := generateScripts(d)
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
	if script, ok := d.GetOk("script"); ok {
		scr, ok := script.(string)
		if !ok {
			return nil, fmt.Errorf("Error parsing script %v as string", script)
		}
		scripts = append(scripts, scr)
	}

	if scriptList, ok := d.GetOk("scripts"); ok {
		for _, script := range scriptList.([]interface{}) {
			scr, ok := script.(string)
			if !ok {
				return nil, fmt.Errorf("Error parsing script %v as string", script)
			}
			scripts = append(scripts, scr)
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
func runScripts(
	ctx context.Context,
	o terraform.UIOutput,
	comm communicator.Communicator,
	scripts []io.ReadCloser) error {
	// Wrap out context in a cancelation function that we use to
	// kill the connection.
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	// Wait for the context to end and then disconnect
	go func() {
		<-ctx.Done()
		comm.Disconnect()
	}()

	// Wait and retry until we establish the connection
	err := retryFunc(ctx, comm.Timeout(), func() error {
		err := comm.Connect(o)
		return err
	})
	if err != nil {
		return err
	}

	for _, script := range scripts {
		var cmd *remote.Cmd
		outR, outW := io.Pipe()
		errR, errW := io.Pipe()
		outDoneCh := make(chan struct{})
		errDoneCh := make(chan struct{})
		go copyOutput(o, outR, outDoneCh)
		go copyOutput(o, errR, errDoneCh)

		remotePath := comm.ScriptPath()
		err = retryFunc(ctx, comm.Timeout(), func() error {
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

			return nil
		})
		if err == nil {
			cmd.Wait()
			if cmd.ExitStatus != 0 {
				err = fmt.Errorf("Script exited with non-zero exit status: %d", cmd.ExitStatus)
			}
		}

		// If we have an error, end our context so the disconnect happens.
		// This has to happen before the output cleanup below since during
		// an interrupt this will cause the outputs to end.
		if err != nil {
			cancelFunc()
		}

		// Wait for output to clean up
		outW.Close()
		errW.Close()
		<-outDoneCh
		<-errDoneCh

		// Upload a blank follow up file in the same path to prevent residual
		// script contents from remaining on remote machine
		empty := bytes.NewReader([]byte(""))
		if err := comm.Upload(remotePath, empty); err != nil {
			// This feature is best-effort.
			log.Printf("[WARN] Failed to upload empty follow up script: %v", err)
		}

		// If we have an error, return it out now that we've cleaned up
		if err != nil {
			return err
		}
	}

	return nil
}

func copyOutput(
	o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

// retryFunc is used to retry a function for a given duration
// TODO: this should probably backoff too
func retryFunc(ctx context.Context, timeout time.Duration, f func() error) error {
	// Build a new context with the timeout
	ctx, done := context.WithTimeout(ctx, timeout)
	defer done()

	// container for atomic error value
	type errWrap struct {
		E error
	}

	// Try the function in a goroutine
	var errVal atomic.Value
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		for {
			// If our context ended, we want to exit right away.
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Try the function call
			err := f()
			errVal.Store(&errWrap{err})

			if err == nil {
				return
			}

			log.Printf("Retryable error: %v", err)
		}
	}()

	// Wait for completion
	select {
	case <-ctx.Done():
	case <-doneCh:
	}

	// Check if we have a context error to check if we're interrupted or timeout
	switch ctx.Err() {
	case context.Canceled:
		return fmt.Errorf("interrupted")
	case context.DeadlineExceeded:
		return fmt.Errorf("timeout")
	}

	// Check if we got an error executing
	if ev, ok := errVal.Load().(errWrap); ok {
		return ev.E
	}

	return nil
}
