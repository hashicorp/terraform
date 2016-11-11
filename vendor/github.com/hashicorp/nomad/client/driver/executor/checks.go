package executor

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/armon/circbuf"
	docker "github.com/fsouza/go-dockerclient"
	cstructs "github.com/hashicorp/nomad/client/driver/structs"
)

var (
	// We store the client globally to cache the connection to the docker daemon.
	createClient sync.Once
	client       *docker.Client
)

const (
	// The default check timeout
	defaultCheckTimeout = 30 * time.Second
)

// DockerScriptCheck runs nagios compatible scripts in a docker container and
// provides the check result
type DockerScriptCheck struct {
	id          string        // id of the check
	interval    time.Duration // interval of the check
	timeout     time.Duration // timeout of the check
	containerID string        // container id in which the check will be invoked
	logger      *log.Logger
	cmd         string   // check command
	args        []string // check command arguments

	dockerEndpoint string // docker endpoint
	tlsCert        string // path to tls certificate
	tlsCa          string // path to tls ca
	tlsKey         string // path to tls key
}

// dockerClient creates the client to interact with the docker daemon
func (d *DockerScriptCheck) dockerClient() (*docker.Client, error) {
	if client != nil {
		return client, nil
	}

	var err error
	createClient.Do(func() {
		if d.dockerEndpoint != "" {
			if d.tlsCert+d.tlsKey+d.tlsCa != "" {
				d.logger.Printf("[DEBUG] executor.checks: using TLS client connection to %s", d.dockerEndpoint)
				client, err = docker.NewTLSClient(d.dockerEndpoint, d.tlsCert, d.tlsKey, d.tlsCa)
			} else {
				d.logger.Printf("[DEBUG] executor.checks: using standard client connection to %s", d.dockerEndpoint)
				client, err = docker.NewClient(d.dockerEndpoint)
			}
			return
		}

		d.logger.Println("[DEBUG] executor.checks: using client connection initialized from environment")
		client, err = docker.NewClientFromEnv()
	})
	return client, err
}

// Run runs a script check inside a docker container
func (d *DockerScriptCheck) Run() *cstructs.CheckResult {
	var (
		exec    *docker.Exec
		err     error
		execRes *docker.ExecInspect
		time    = time.Now()
	)

	if client, err = d.dockerClient(); err != nil {
		return &cstructs.CheckResult{Err: err}
	}
	client = client
	execOpts := docker.CreateExecOptions{
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		Cmd:          append([]string{d.cmd}, d.args...),
		Container:    d.containerID,
	}
	if exec, err = client.CreateExec(execOpts); err != nil {
		return &cstructs.CheckResult{Err: err}
	}

	output, _ := circbuf.NewBuffer(int64(cstructs.CheckBufSize))
	startOpts := docker.StartExecOptions{
		Detach:       false,
		Tty:          false,
		OutputStream: output,
		ErrorStream:  output,
	}

	if err = client.StartExec(exec.ID, startOpts); err != nil {
		return &cstructs.CheckResult{Err: err}
	}
	if execRes, err = client.InspectExec(exec.ID); err != nil {
		return &cstructs.CheckResult{Err: err}
	}
	return &cstructs.CheckResult{
		ExitCode:  execRes.ExitCode,
		Output:    string(output.Bytes()),
		Timestamp: time,
	}
}

// ID returns the check id
func (d *DockerScriptCheck) ID() string {
	return d.id
}

// Interval returns the interval at which the check has to run
func (d *DockerScriptCheck) Interval() time.Duration {
	return d.interval
}

// Timeout returns the duration after which a check is timed out.
func (d *DockerScriptCheck) Timeout() time.Duration {
	if d.timeout == 0 {
		return defaultCheckTimeout
	}
	return d.timeout
}

// ExecScriptCheck runs a nagios compatible script and returns the check result
type ExecScriptCheck struct {
	id       string        // id of the script check
	interval time.Duration // interval at which the check is invoked
	timeout  time.Duration // timeout duration of the check
	cmd      string        // command of the check
	args     []string      // args passed to the check
	taskDir  string        // the root directory of the check

	FSIsolation bool // indicates whether the check has to be run within a chroot
}

// Run runs an exec script check
func (e *ExecScriptCheck) Run() *cstructs.CheckResult {
	buf, _ := circbuf.NewBuffer(int64(cstructs.CheckBufSize))
	cmd := exec.Command(e.cmd, e.args...)
	cmd.Stdout = buf
	cmd.Stderr = buf
	e.setChroot(cmd)
	ts := time.Now()
	if err := cmd.Start(); err != nil {
		return &cstructs.CheckResult{Err: err}
	}
	errCh := make(chan error, 2)
	go func() {
		errCh <- cmd.Wait()
	}()
	for {
		select {
		case err := <-errCh:
			endTime := time.Now()
			if err == nil {
				return &cstructs.CheckResult{
					ExitCode:  0,
					Output:    string(buf.Bytes()),
					Timestamp: ts,
				}
			}
			exitCode := 1
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				}
			}
			return &cstructs.CheckResult{
				ExitCode:  exitCode,
				Output:    string(buf.Bytes()),
				Timestamp: ts,
				Duration:  endTime.Sub(ts),
			}
		case <-time.After(e.Timeout()):
			errCh <- fmt.Errorf("timed out after waiting 30s")
		}
	}
	return nil
}

// ID returns the check id
func (e *ExecScriptCheck) ID() string {
	return e.id
}

// Interval returns the interval at which the check has to run
func (e *ExecScriptCheck) Interval() time.Duration {
	return e.interval
}

// Timeout returns the duration after which a check is timed out.
func (e *ExecScriptCheck) Timeout() time.Duration {
	if e.timeout == 0 {
		return defaultCheckTimeout
	}
	return e.timeout
}
