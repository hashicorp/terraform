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
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

// maxBackoffDealy is the maximum delay between retry attempts
var maxBackoffDelay = 10 * time.Second
var initialBackoffDelay = time.Second

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

	// Wait for the context to end and then disconnect
	go func() {
		<-ctx.Done()
		comm.Disconnect()
	}()

	// Wait and retry until we establish the connection
	err := communicator.Retry(ctx, func() error {
		return comm.Connect(o)
	})
	if err != nil {
		return err
	}

	for _, script := range scripts {
		var cmd *remote.Cmd

		outR, outW := io.Pipe()
		errR, errW := io.Pipe()
		defer outW.Close()
		defer errW.Close()

		go copyOutput(o, outR)
		go copyOutput(o, errR)

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

func copyOutput(
	o terraform.UIOutput, r io.Reader) {
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}
