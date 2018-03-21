package communicator

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
)

// MockCommunicator is an implementation of Communicator that can be used for tests.
type MockCommunicator struct {
	RemoteScriptPath string
	Commands         map[string]bool
	Uploads          map[string]string
	UploadScripts    map[string]string
	UploadDirs       map[string]string
	CommandFunc      func(*remote.Cmd) error
	DisconnectFunc   func() error
	ConnTimeout      time.Duration
}

// Connect implementation of communicator.Communicator interface
func (c *MockCommunicator) Connect(o terraform.UIOutput) error {
	return nil
}

// Disconnect implementation of communicator.Communicator interface
func (c *MockCommunicator) Disconnect() error {
	if c.DisconnectFunc != nil {
		return c.DisconnectFunc()
	}
	return nil
}

// Timeout implementation of communicator.Communicator interface
func (c *MockCommunicator) Timeout() time.Duration {
	if c.ConnTimeout != 0 {
		return c.ConnTimeout
	}
	return time.Duration(5 * time.Second)
}

// ScriptPath implementation of communicator.Communicator interface
func (c *MockCommunicator) ScriptPath() string {
	return c.RemoteScriptPath
}

// Start implementation of communicator.Communicator interface
func (c *MockCommunicator) Start(r *remote.Cmd) error {
	r.Init()

	if c.CommandFunc != nil {
		return c.CommandFunc(r)
	}

	if !c.Commands[r.Command] {
		return fmt.Errorf("Command not found!")
	}

	r.SetExitStatus(0, nil)

	return nil
}

// Upload implementation of communicator.Communicator interface
func (c *MockCommunicator) Upload(path string, input io.Reader) error {
	f, ok := c.Uploads[path]
	if !ok {
		return fmt.Errorf("Path %q not found!", path)
	}

	var buf bytes.Buffer
	buf.ReadFrom(input)
	content := strings.TrimSpace(buf.String())

	f = strings.TrimSpace(f)
	if f != content {
		return fmt.Errorf("expected: %q\n\ngot: %q\n", f, content)
	}

	return nil
}

// UploadScript implementation of communicator.Communicator interface
func (c *MockCommunicator) UploadScript(path string, input io.Reader) error {
	c.Uploads = c.UploadScripts
	return c.Upload(path, input)
}

// UploadDir implementation of communicator.Communicator interface
func (c *MockCommunicator) UploadDir(dst string, src string) error {
	v, ok := c.UploadDirs[src]
	if !ok {
		return fmt.Errorf("Directory not found!")
	}

	if v != dst {
		return fmt.Errorf("expected: %q\n\ngot: %q\n", v, dst)
	}

	return nil
}
