package winrm

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/masterzen/winrm"
	"github.com/packer-community/winrmcp/winrmcp"

	// This import is a bit strange, but it's needed so `make updatedeps` can see and download it
	_ "github.com/dylanmei/winrmtest"
)

// Communicator represents the WinRM communicator
type Communicator struct {
	connInfo *connectionInfo
	client   *winrm.Client
	endpoint *winrm.Endpoint
	rand     *rand.Rand
}

// New creates a new communicator implementation over WinRM.
func New(s *terraform.InstanceState) (*Communicator, error) {
	connInfo, err := parseConnectionInfo(s)
	if err != nil {
		return nil, err
	}

	endpoint := &winrm.Endpoint{
		Host:     connInfo.Host,
		Port:     connInfo.Port,
		HTTPS:    connInfo.HTTPS,
		Insecure: connInfo.Insecure,
	}
	if len(connInfo.CACert) > 0 {
		endpoint.CACert = []byte(connInfo.CACert)
	}

	comm := &Communicator{
		connInfo: connInfo,
		endpoint: endpoint,
		// Seed our own rand source so that script paths are not deterministic
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	return comm, nil
}

// Connect implementation of communicator.Communicator interface
func (c *Communicator) Connect(o terraform.UIOutput) error {
	if c.client != nil {
		return nil
	}

	params := winrm.DefaultParameters
	params.Timeout = formatDuration(c.Timeout())
	if c.connInfo.NTLM == true {
		params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientNTLM{} }
	}

	client, err := winrm.NewClientWithParameters(
		c.endpoint, c.connInfo.User, c.connInfo.Password, params)
	if err != nil {
		return err
	}

	if o != nil {
		o.Output(fmt.Sprintf(
			"Connecting to remote host via WinRM...\n"+
				"  Host: %s\n"+
				"  Port: %d\n"+
				"  User: %s\n"+
				"  Password: %t\n"+
				"  HTTPS: %t\n"+
				"  Insecure: %t\n"+
				"  NTLM: %t\n"+
				"  CACert: %t",
			c.connInfo.Host,
			c.connInfo.Port,
			c.connInfo.User,
			c.connInfo.Password != "",
			c.connInfo.HTTPS,
			c.connInfo.Insecure,
			c.connInfo.NTLM,
			c.connInfo.CACert != "",
		))
	}

	log.Printf("[DEBUG] connecting to remote shell using WinRM")
	shell, err := client.CreateShell()
	if err != nil {
		log.Printf("[ERROR] connection error: %s", err)
		return err
	}

	err = shell.Close()
	if err != nil {
		log.Printf("[ERROR] error closing connection: %s", err)
		return err
	}

	if o != nil {
		o.Output("Connected!")
	}

	c.client = client

	return nil
}

// Disconnect implementation of communicator.Communicator interface
func (c *Communicator) Disconnect() error {
	c.client = nil
	return nil
}

// Timeout implementation of communicator.Communicator interface
func (c *Communicator) Timeout() time.Duration {
	return c.connInfo.TimeoutVal
}

// ScriptPath implementation of communicator.Communicator interface
func (c *Communicator) ScriptPath() string {
	return strings.Replace(
		c.connInfo.ScriptPath, "%RAND%",
		strconv.FormatInt(int64(c.rand.Int31()), 10), -1)
}

// Start implementation of communicator.Communicator interface
func (c *Communicator) Start(rc *remote.Cmd) error {
	rc.Init()
	log.Printf("[DEBUG] starting remote command: %s", rc.Command)

	if c.client == nil {
		return errors.New("winrm client is not connected")
	}

	status, err := c.client.Run(rc.Command, rc.Stdout, rc.Stderr)
	rc.SetExitStatus(status, err)

	return nil
}

// Upload implementation of communicator.Communicator interface
func (c *Communicator) Upload(path string, input io.Reader) error {
	wcp, err := c.newCopyClient()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Uploading file to '%s'", path)
	return wcp.Write(path, input)
}

// UploadScript implementation of communicator.Communicator interface
func (c *Communicator) UploadScript(path string, input io.Reader) error {
	return c.Upload(path, input)
}

// UploadDir implementation of communicator.Communicator interface
func (c *Communicator) UploadDir(dst string, src string) error {
	log.Printf("[DEBUG] Uploading dir '%s' to '%s'", src, dst)
	wcp, err := c.newCopyClient()
	if err != nil {
		return err
	}
	return wcp.Copy(src, dst)
}

func (c *Communicator) newCopyClient() (*winrmcp.Winrmcp, error) {
	addr := fmt.Sprintf("%s:%d", c.endpoint.Host, c.endpoint.Port)

	config := winrmcp.Config{
		Auth: winrmcp.Auth{
			User:     c.connInfo.User,
			Password: c.connInfo.Password,
		},
		Https:                 c.connInfo.HTTPS,
		Insecure:              c.connInfo.Insecure,
		OperationTimeout:      c.Timeout(),
		MaxOperationsPerShell: 15, // lowest common denominator
	}

	if c.connInfo.NTLM == true {
		config.TransportDecorator = func() winrm.Transporter { return &winrm.ClientNTLM{} }
	}

	if c.connInfo.CACert != "" {
		config.CACertBytes = []byte(c.connInfo.CACert)
	}

	return winrmcp.New(addr, &config)
}
