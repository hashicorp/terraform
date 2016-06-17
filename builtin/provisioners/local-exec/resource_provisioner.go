package localexec

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

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

	// Execute the command using a shell
	var shell string
	var params []string
	if runtime.GOOS == "windows" {
		shellEx, ok := c.Config["shell"]
		if !ok {
			shell = "cmd"
			params = append(params, "/C")
			params = append(params, command)
		} else {
			if strings.Contains(strings.ToLower(shellEx.(string)), "powershell") {
				shell = shellEx.(string)
				params = append(params, "-NoProfile")
				params = append(params, "-ExecutionPolicy")
				params = append(params, "Bypass")
				params = append(params, "-Command")
				params = append(params, command)
			} else if strings.Contains(strings.ToLower(shellEx.(string)), "cmd") {
				shell = "cmd"
				params = append(params, "/C")
				params = append(params, command)
			} else {
				return fmt.Errorf("Unsupported shell")
			}
		}
	} else {
		shell = "/bin/sh"
		params = append(params, "-c")
		params = append(params, command)
	}

	// Setup the reader that will read the lines from the command
	pr, pw := io.Pipe()
	copyDoneCh := make(chan struct{})
	go p.copyOutput(o, pr, copyDoneCh)

	// Setup the command
	cmd := exec.Command(shell, params...)
	output, _ := circbuf.NewBuffer(maxBufSize)
	cmd.Stderr = io.MultiWriter(output, pw)
	cmd.Stdout = io.MultiWriter(output, pw)

	// Output what we're about to run
	o.Output(fmt.Sprintf(
		"Executing: %s %v",
		shell, params))

	// Run the command to completion
	err := cmd.Run()

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

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	validator := config.Validator{
		Required: []string{"command"},
		Optional: []string{"shell"},
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
