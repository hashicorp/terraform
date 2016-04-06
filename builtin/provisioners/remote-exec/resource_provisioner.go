package remoteexec

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

// ResourceProvisioner represents a remote exec provisioner
type ResourceProvisioner struct{}

// Apply executes the remote exec provisioner
func (p *ResourceProvisioner) Apply(
	o terraform.UIOutput,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) error {
	// Get a new communicator
	comm, err := communicator.New(s)
	if err != nil {
		return err
	}

	// Collect the scripts
	scripts, err := p.collectScripts(c)
	if err != nil {
		return err
	}
	for _, s := range scripts {
		defer s.Close()
	}

	// Copy and execute each script
	if err := p.runScripts(o, comm, scripts); err != nil {
		return err
	}
	return nil
}

// Validate checks if the required arguments are configured
func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	num := 0
	for name := range c.Raw {
		switch name {
		case "scripts", "script", "inline":
			num++
		default:
			es = append(es, fmt.Errorf("Unknown configuration '%s'", name))
		}
	}
	if num != 1 {
		es = append(es, fmt.Errorf("Must provide one of 'scripts', 'script' or 'inline' to remote-exec"))
	}
	return
}

// generateScript takes the configuration and creates a script to be executed
// from the inline configs
func (p *ResourceProvisioner) generateScript(c *terraform.ResourceConfig) (string, error) {
	var lines []string
	command, ok := c.Config["inline"]
	if ok {
		switch cmd := command.(type) {
		case string:
			lines = append(lines, cmd)
		case []string:
			lines = append(lines, cmd...)
		case []interface{}:
			for _, l := range cmd {
				lStr, ok := l.(string)
				if ok {
					lines = append(lines, lStr)
				} else {
					return "", fmt.Errorf("Unsupported 'inline' type! Must be string, or list of strings.")
				}
			}
		default:
			return "", fmt.Errorf("Unsupported 'inline' type! Must be string, or list of strings.")
		}
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n"), nil
}

// collectScripts is used to collect all the scripts we need
// to execute in preparation for copying them.
func (p *ResourceProvisioner) collectScripts(c *terraform.ResourceConfig) ([]io.ReadCloser, error) {
	// Check if inline
	_, ok := c.Config["inline"]
	if ok {
		script, err := p.generateScript(c)
		if err != nil {
			return nil, err
		}
		rc := ioutil.NopCloser(bytes.NewReader([]byte(script)))
		return []io.ReadCloser{rc}, nil
	}

	// Collect scripts
	var scripts []string
	s, ok := c.Config["script"]
	if ok {
		sStr, ok := s.(string)
		if !ok {
			return nil, fmt.Errorf("Unsupported 'script' type! Must be a string.")
		}
		scripts = append(scripts, sStr)
	}

	sl, ok := c.Config["scripts"]
	if ok {
		switch slt := sl.(type) {
		case []string:
			scripts = append(scripts, slt...)
		case []interface{}:
			for _, l := range slt {
				lStr, ok := l.(string)
				if ok {
					scripts = append(scripts, lStr)
				} else {
					return nil, fmt.Errorf("Unsupported 'scripts' type! Must be list of strings.")
				}
			}
		default:
			return nil, fmt.Errorf("Unsupported 'scripts' type! Must be list of strings.")
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
func (p *ResourceProvisioner) runScripts(
	o terraform.UIOutput,
	comm communicator.Communicator,
	scripts []io.ReadCloser) error {
	// Wait and retry until we establish the connection
	err := retryFunc(comm.Timeout(), func() error {
		err := comm.Connect(o)
		return err
	})
	if err != nil {
		return err
	}
	defer comm.Disconnect()

	for _, script := range scripts {
		var cmd *remote.Cmd
		outR, outW := io.Pipe()
		errR, errW := io.Pipe()
		outDoneCh := make(chan struct{})
		errDoneCh := make(chan struct{})
		go p.copyOutput(o, outR, outDoneCh)
		go p.copyOutput(o, errR, errDoneCh)

		remotePath := comm.ScriptPath()
		err = retryFunc(comm.Timeout(), func() error {

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

func (p *ResourceProvisioner) copyOutput(
	o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

// retryFunc is used to retry a function for a given duration
func retryFunc(timeout time.Duration, f func() error) error {
	finish := time.After(timeout)
	for {
		err := f()
		if err == nil {
			return nil
		}
		log.Printf("Retryable error: %v", err)

		select {
		case <-finish:
			return err
		case <-time.After(3 * time.Second):
		}
	}
}
