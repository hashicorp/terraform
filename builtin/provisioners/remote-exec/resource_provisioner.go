package remoteexec

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	helper "github.com/hashicorp/terraform/helper/ssh"
	"github.com/hashicorp/terraform/terraform"
)

const (
	// DefaultShebang is added at the top of the script file
	DefaultShebang = "#!/bin/sh"
)

type ResourceProvisioner struct{}

func (p *ResourceProvisioner) Apply(s *terraform.InstanceState,
	c *terraform.ResourceConfig) error {
	// Ensure the connection type is SSH
	if err := helper.VerifySSH(s); err != nil {
		return err
	}

	// Get the SSH configuration
	conf, err := helper.ParseSSHConfig(s)
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
	if err := p.runScripts(conf, scripts); err != nil {
		return err
	}
	return nil
}

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	num := 0
	for name := range c.Raw {
		switch name {
		case "scripts":
			fallthrough
		case "script":
			fallthrough
		case "inline":
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
	lines := []string{DefaultShebang}
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
func (p *ResourceProvisioner) runScripts(conf *helper.SSHConfig, scripts []io.ReadCloser) error {
	// Get the SSH client config
	config, err := helper.PrepareConfig(conf)
	if err != nil {
		return err
	}

	// Wait and retry until we establish the SSH connection
	var comm *helper.SSHCommunicator
	err = retryFunc(conf.TimeoutVal, func() error {
		host := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
		comm, err = helper.New(host, config)
		return err
	})
	if err != nil {
		return err
	}

	for _, script := range scripts {
		var cmd *helper.RemoteCmd
		err := retryFunc(conf.TimeoutVal, func() error {
			if err := comm.Upload(conf.ScriptPath, script); err != nil {
				return fmt.Errorf("Failed to upload script: %v", err)
			}
			cmd = &helper.RemoteCmd{
				Command: fmt.Sprintf("chmod 0777 %s", conf.ScriptPath),
			}
			if err := comm.Start(cmd); err != nil {
				return fmt.Errorf(
					"Error chmodding script file to 0777 in remote "+
						"machine: %s", err)
			}
			cmd.Wait()

			stdOutReader, stdOutWriter := io.Pipe()
			stdErrReader, stdErrWriter := io.Pipe()
			go streamLogs(stdOutReader, "stdout")
			go streamLogs(stdErrReader, "stderr")

			cmd = &helper.RemoteCmd{
				Command: conf.ScriptPath,
				Stdout:  stdOutWriter,
				Stderr:  stdErrWriter,
			}
			if err := comm.Start(cmd); err != nil {
				return fmt.Errorf("Error starting script: %v", err)
			}
			return nil
		})
		if err != nil {
			return err
		}

		cmd.Wait()
		if cmd.ExitStatus != 0 {
			return fmt.Errorf("Script exited with non-zero exit status: %d", cmd.ExitStatus)
		}
	}

	return nil
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

// streamLogs is used to stream lines from stdout/stderr
// of a remote command to log output for users.
func streamLogs(r io.ReadCloser, name string) {
	defer r.Close()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf("remote-exec: %s: %s", name, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return
	}
}
