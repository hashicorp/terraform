package localexec

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"

	"github.com/armon/circbuf"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

const (
	// maxBufSize limits how much output we collect from a local
	// invocation. This is to prevent TF memory usage from growing
	// to an enormous amount due to a faulty process.
	maxBufSize = 8 * 1024
)

type ResourceProvisioner struct{}

func (p *ResourceProvisioner) Apply(
	o terraform.UIOutput,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) error {

	// Get the command
	commandRaw, ok := c.Config["command"]
	if !ok {
		return fmt.Errorf("local-exec provisioner missing 'command'")
	}
	command, ok := commandRaw.(string)
	if !ok {
		return fmt.Errorf("local-exec provisioner command must be a string")
	}

	var verify string
	verifyRaw, verifyRawOk := c.Config["verify"]
	if verifyRawOk {
		verify, ok = verifyRaw.(string)
		if !ok {
			return fmt.Errorf("local-exec provisioner verify must be a string")
		}
	}

	// Execute the command
	if err := p.localExec(false, command, o); err != nil {
		return err
	}

	// Verify the command if desired
	if verify != "" {
		if err := p.localExec(true, verify, o); err != nil {
			return err
		}
	}

	return nil
}

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	validator := config.Validator{
		Required: []string{"command"},
		Optional: []string{"verify"},
	}
	return validator.Validate(c)
}

func (p *ResourceProvisioner) copyOutput(
	o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

func (p *ResourceProvisioner) localExec(
	verify bool,
	command string,
	o terraform.UIOutput) error {

	action := "Executing"
	if verify {
		action = "Verifying"
	}
	result := "running"
	if verify {
		result = "verifying"
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
	go p.copyOutput(o, pr, copyDoneCh)

	// Setup the command
	cmd := exec.Command(shell, flag, command)
	output, _ := circbuf.NewBuffer(maxBufSize)
	cmd.Stderr = io.MultiWriter(output, pw)
	cmd.Stdout = io.MultiWriter(output, pw)

	// Output what we're about to run
	o.Output(fmt.Sprintf(
		"%s: %s %s \"%s\"",
		action, shell, flag, command))

	// Run the command to completion
	err := cmd.Run()

	// Close the write-end of the pipe so that the goroutine mirroring output
	// ends properly.
	pw.Close()
	<-copyDoneCh

	if err != nil {
		return fmt.Errorf("Error %s command '%s': %v. Output: %s",
			result, command, err, output.Bytes())
	}

	return nil
}
