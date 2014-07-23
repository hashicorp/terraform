package localexec

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/armon/circbuf"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/terraform"
)

const (
	// maxBufSize limits how much output we collect from a local
	// invocation. This is to prevent TF memory usage from growing
	// to an enormous amount due to a faulty process.
	maxBufSize = 8 * 1024
)

type ResourceProvisioner struct{}

func (p *ResourceProvisioner) Apply(
	s *terraform.ResourceState,
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
	var shell, flag string
	if runtime.GOOS == "windows" {
		shell = "cmd"
		flag = "/C"
	} else {
		shell = "/bin/sh"
		flag = "-c"
	}

	// Setup the command
	cmd := exec.Command(shell, flag, command)
	output, _ := circbuf.NewBuffer(maxBufSize)
	cmd.Stderr = output
	cmd.Stdout = output

	// Run the command to completion
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error running command '%s': %v. Output: %s",
			command, err, output.Bytes())
	}
	return nil
}

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	validator := config.Validator{
		Required: []string{"command"},
	}
	return validator.Validate(c)
}
