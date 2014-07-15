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

	"code.google.com/p/go.crypto/ssh"
	helper "github.com/hashicorp/terraform/helper/ssh"
	"github.com/hashicorp/terraform/terraform"
)

const (
	// DefaultUser is used if there is no default user given
	DefaultUser = "root"

	// DefaultPort is used if there is no port given
	DefaultPort = 22

	// DefaultScriptPath is used as the path to copy the file to
	// for remote execution if not provided otherwise.
	DefaultScriptPath = "/tmp/script.sh"

	// DefaultTimeout is used if there is no timeout given
	DefaultTimeout = 5 * time.Minute

	// DefaultShebang is added at the top of the script file
	DefaultShebang = "#!/bin/sh"
)

type ResourceProvisioner struct{}

func (p *ResourceProvisioner) Apply(s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceState, error) {
	// Ensure the connection type is SSH
	if err := helper.VerifySSH(s); err != nil {
		return s, err
	}

	// Get the SSH configuration
	conf, err := helper.ParseSSHConfig(s)
	if err != nil {
		return s, err
	}

	// Collect the scripts
	scripts, err := p.collectScripts(c)
	if err != nil {
		return s, err
	}
	for _, s := range scripts {
		defer s.Close()
	}

	// Copy and execute each script
	if err := p.runScripts(conf, scripts); err != nil {
		return s, err
	}
	return s, nil
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
// to execute in preperation for copying them.
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
	sshConf := &ssh.ClientConfig{
		User: conf.User,
	}
	if conf.KeyFile != "" {
		key, err := ioutil.ReadFile(conf.KeyFile)
		if err != nil {
			return fmt.Errorf("Failed to read key file '%s': %v", conf.KeyFile, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("Failed to parse key file '%s': %v", conf.KeyFile, err)
		}
		sshConf.Auth = append(sshConf.Auth, ssh.PublicKeys(signer))
	}
	if conf.Password != "" {
		sshConf.Auth = append(sshConf.Auth,
			ssh.Password(conf.Password))
		sshConf.Auth = append(sshConf.Auth,
			ssh.KeyboardInteractive(helper.PasswordKeyboardInteractive(conf.Password)))
	}
	host := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	config := &helper.Config{
		SSHConfig:  sshConf,
		Connection: helper.ConnectFunc("tcp", host),
	}

	// Wait and retry until we establish the SSH connection
	var comm *helper.SSHCommunicator
	err := retryFunc(conf.TimeoutVal, func() error {
		var err error
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

			rPipe1, wPipe1 := io.Pipe()
			rPipe2, wPipe2 := io.Pipe()
			go streamLogs(rPipe1, "stdout")
			go streamLogs(rPipe2, "stderr")

			cmd = &helper.RemoteCmd{
				Command: conf.ScriptPath,
				Stdout:  wPipe1,
				Stderr:  wPipe2,
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
	bufR := bufio.NewReader(r)
	for {
		line, err := bufR.ReadString('\n')
		if err != nil {
			return
		}
		log.Printf("remote-exec: %s: %s", name, line)
	}
}
