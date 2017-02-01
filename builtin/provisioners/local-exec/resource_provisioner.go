package localexec

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"

	"github.com/armon/circbuf"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

const (
	// maxBufSize limits how much output we collect from a local
	// invocation. This is to prevent TF memory usage from growing
	// to an enormous amount due to a faulty process.
	maxBufSize = 8 * 1024
)

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"command": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},

		ApplyFunc: applyFn,
	}
}

func applyFn(ctx context.Context) error {
	data := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)

	command := data.Get("command").(string)
	if command == "" {
		return fmt.Errorf("local-exec provisioner command must be a non-empty string")
	}

	// Execute the command using a shell
	var shell, flag string
	if runtime.GOOS == "windows" {
		shell = "cmd"
		flag = "/C"
	} else {
		shell = "/bin/sh"
		flag = "-c"
	}

	// Setup the reader that will read the lines from the command
	pr, pw := io.Pipe()
	copyDoneCh := make(chan struct{})
	go copyOutput(o, pr, copyDoneCh)

	// Setup the command
	cmd := exec.Command(shell, flag, command)
	output, _ := circbuf.NewBuffer(maxBufSize)
	cmd.Stderr = io.MultiWriter(output, pw)
	cmd.Stdout = io.MultiWriter(output, pw)

	// Output what we're about to run
	o.Output(fmt.Sprintf(
		"Executing: %s %s \"%s\"",
		shell, flag, command))

	// Start the command
	err := cmd.Start()
	if err == nil {
		// Wait for the command to complete in a goroutine
		doneCh := make(chan error, 1)
		go func() {
			doneCh <- cmd.Wait()
		}()

		// Wait for the command to finish or for us to be interrupted
		select {
		case err = <-doneCh:
		case <-ctx.Done():
			cmd.Process.Kill()
			err = cmd.Wait()
		}
	}

	// Close the write-end of the pipe so that the goroutine mirroring output
	// ends properly.
	pw.Close()
	<-copyDoneCh

	if err != nil {
		return fmt.Errorf("Error running command '%s': %v. Output: %s",
			command, err, output.Bytes())
	}

	return nil
}

func copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}
