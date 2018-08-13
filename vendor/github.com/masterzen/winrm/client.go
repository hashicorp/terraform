package winrm

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"io"
	"sync"

	"github.com/masterzen/winrm/soap"
)

// Client struct
type Client struct {
	Parameters
	username string
	password string
	useHTTPS bool
	url      string
	http     Transporter
}

// Transporter does different transporters
// and init a Post request based on them
type Transporter interface {
	// init request baset on the transport configurations
	Post(*Client, *soap.SoapMessage) (string, error)
	Transport(*Endpoint) error
}

// NewClient will create a new remote client on url, connecting with user and password
// This function doesn't connect (connection happens only when CreateShell is called)
func NewClient(endpoint *Endpoint, user, password string) (*Client, error) {
	return NewClientWithParameters(endpoint, user, password, DefaultParameters)
}

// NewClientWithParameters will create a new remote client on url, connecting with user and password
// This function doesn't connect (connection happens only when CreateShell is called)
func NewClientWithParameters(endpoint *Endpoint, user, password string, params *Parameters) (*Client, error) {

	// alloc a new client
	client := &Client{
		Parameters: *params,
		username:   user,
		password:   password,
		url:        endpoint.url(),
		useHTTPS:   endpoint.HTTPS,
		// default transport
		http: &clientRequest{},
	}

	// switch to other transport if provided
	if params.TransportDecorator != nil {
		client.http = params.TransportDecorator()
	}

	// set the transport to some endpoint configuration
	if err := client.http.Transport(endpoint); err != nil {
		return nil, fmt.Errorf("Can't parse this key and certs: %s", err)
	}

	return client, nil
}

func readCACerts(certs []byte) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()

	if !certPool.AppendCertsFromPEM(certs) {
		return nil, fmt.Errorf("Unable to read certificates")
	}

	return certPool, nil
}

// CreateShell will create a WinRM Shell,
// which is the prealable for running commands.
func (c *Client) CreateShell() (*Shell, error) {
	request := NewOpenShellRequest(c.url, &c.Parameters)
	defer request.Free()

	response, err := c.sendRequest(request)
	if err != nil {
		return nil, err
	}

	shellID, err := ParseOpenShellResponse(response)
	if err != nil {
		return nil, err
	}

	return c.NewShell(shellID), nil

}

// NewShell will create a new WinRM Shell for the given shellID
func (c *Client) NewShell(id string) *Shell {
	return &Shell{client: c, id: id}
}

// sendRequest exec the custom http func from the client
func (c *Client) sendRequest(request *soap.SoapMessage) (string, error) {
	return c.http.Post(c, request)
}

// Run will run command on the the remote host, writing the process stdout and stderr to
// the given writers. Note with this method it isn't possible to inject stdin.
func (c *Client) Run(command string, stdout io.Writer, stderr io.Writer) (int, error) {
	shell, err := c.CreateShell()
	if err != nil {
		return 1, err
	}
	defer shell.Close()
	cmd, err := shell.Execute(command)
	if err != nil {
		return 1, err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(stdout, cmd.Stdout)
	}()

	go func() {
		defer wg.Done()
		io.Copy(stderr, cmd.Stderr)
	}()

	cmd.Wait()
	wg.Wait()

	return cmd.ExitCode(), cmd.err
}

// RunWithString will run command on the the remote host, returning the process stdout and stderr
// as strings, and using the input stdin string as the process input
func (c *Client) RunWithString(command string, stdin string) (string, string, int, error) {
	shell, err := c.CreateShell()
	if err != nil {
		return "", "", 1, err
	}
	defer shell.Close()

	cmd, err := shell.Execute(command)
	if err != nil {
		return "", "", 1, err
	}
	if len(stdin) > 0 {
		cmd.Stdin.Write([]byte(stdin))
	}

	var outWriter, errWriter bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(&outWriter, cmd.Stdout)
	}()

	go func() {
		defer wg.Done()
		io.Copy(&errWriter, cmd.Stderr)
	}()

	cmd.Wait()
	wg.Wait()

	return outWriter.String(), errWriter.String(), cmd.ExitCode(), cmd.err
}

// RunWithInput will run command on the the remote host, writing the process stdout and stderr to
// the given writers, and injecting the process stdin with the stdin reader.
// Warning stdin (not stdout/stderr) are bufferized, which means reading only one byte in stdin will
// send a winrm http packet to the remote host. If stdin is a pipe, it might be better for
// performance reasons to buffer it.
func (c Client) RunWithInput(command string, stdout, stderr io.Writer, stdin io.Reader) (int, error) {
	shell, err := c.CreateShell()
	if err != nil {
		return 1, err
	}
	defer shell.Close()
	cmd, err := shell.Execute(command)
	if err != nil {
		return 1, err
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		io.Copy(cmd.Stdin, stdin)
	}()
	go func() {
		defer wg.Done()
		io.Copy(stdout, cmd.Stdout)
	}()
	go func() {
		defer wg.Done()
		io.Copy(stderr, cmd.Stderr)
	}()

	cmd.Wait()
	wg.Wait()

	return cmd.ExitCode(), cmd.err

}
