package winrm

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"

	"github.com/masterzen/winrm/soap"
)

type Client struct {
	Parameters
	username  string
	password  string
	useHTTPS  bool
	url       string
	http      HttpPost
	transport http.RoundTripper
}

// NewClient will create a new remote client on url, connecting with user and password
// This function doesn't connect (connection happens only when CreateShell is called)
func NewClient(endpoint *Endpoint, user, password string) (client *Client, err error) {
	params := DefaultParameters()
	client, err = NewClientWithParameters(endpoint, user, password, params)
	return
}

// NewClient will create a new remote client on url, connecting with user and password
// This function doesn't connect (connection happens only when CreateShell is called)
func NewClientWithParameters(endpoint *Endpoint, user, password string, params *Parameters) (client *Client, err error) {
	transport, err := newTransport(endpoint)

	client = &Client{
		Parameters: *params,
		username:   user,
		password:   password,
		url:        endpoint.url(),
		http:       Http_post,
		useHTTPS:   endpoint.HTTPS,
		transport:  transport,
	}

	if params.TransportDecorator != nil {
		client.transport = params.TransportDecorator(transport)
	}
	return
}

// newTransport will create a new HTTP Transport, with options specified within the endpoint configuration
func newTransport(endpoint *Endpoint) (*http.Transport, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: endpoint.Insecure,
		},
	}

	if endpoint.CACert != nil && len(*endpoint.CACert) > 0 {
		certPool, err := readCACerts(endpoint.CACert)
		if err != nil {
			return nil, err
		}

		transport.TLSClientConfig.RootCAs = certPool
	}

	return transport, nil
}

func readCACerts(certs *[]byte) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()

	if !certPool.AppendCertsFromPEM(*certs) {
		return nil, fmt.Errorf("Unable to read certificates")
	}

	return certPool, nil
}

// CreateShell will create a WinRM Shell, which is the prealable for running
// commands.
func (client *Client) CreateShell() (shell *Shell, err error) {
	request := NewOpenShellRequest(client.url, &client.Parameters)
	defer request.Free()

	response, err := client.sendRequest(request)
	if err == nil {
		var shellId string
		if shellId, err = ParseOpenShellResponse(response); err == nil {
			shell = &Shell{client: client, ShellId: shellId}
		}
	}
	return
}

func (client *Client) sendRequest(request *soap.SoapMessage) (response string, err error) {
	return client.http(client, request)
}

// Run will run command on the the remote host, writing the process stdout and stderr to
// the given writers. Note with this method it isn't possible to inject stdin.
func (client *Client) Run(command string, stdout io.Writer, stderr io.Writer) (exitCode int, err error) {
	shell, err := client.CreateShell()
	if err != nil {
		return 0, err
	}
	var cmd *Command
	cmd, err = shell.Execute(command)
	if err != nil {
		return 0, err
	}
	go io.Copy(stdout, cmd.Stdout)
	go io.Copy(stderr, cmd.Stderr)
	cmd.Wait()
	shell.Close()
	return cmd.ExitCode(), cmd.err
}

// Run will run command on the the remote host, returning the process stdout and stderr
// as strings, and using the input stdin string as the process input
func (client *Client) RunWithString(command string, stdin string) (stdout string, stderr string, exitCode int, err error) {
	shell, err := client.CreateShell()
	if err != nil {
		return "", "", 0, err
	}
	defer shell.Close()
	var cmd *Command
	cmd, err = shell.Execute(command)
	if err != nil {
		return "", "", 0, err
	}
	if len(stdin) > 0 {
		cmd.Stdin.Write([]byte(stdin))
	}
	var outWriter, errWriter bytes.Buffer
	go io.Copy(&outWriter, cmd.Stdout)
	go io.Copy(&errWriter, cmd.Stderr)
	cmd.Wait()
	return outWriter.String(), errWriter.String(), cmd.ExitCode(), cmd.err
}

// Run will run command on the the remote host, writing the process stdout and stderr to
// the given writers, and injecting the process stdin with the stdin reader.
// Warning stdin (not stdout/stderr) are bufferized, which means reading only one byte in stdin will
// send a winrm http packet to the remote host. If stdin is a pipe, it might be better for
// performance reasons to buffer it.
func (client *Client) RunWithInput(command string, stdout io.Writer, stderr io.Writer, stdin io.Reader) (exitCode int, err error) {
	shell, err := client.CreateShell()
	if err != nil {
		return 0, err
	}
	defer shell.Close()
	var cmd *Command
	cmd, err = shell.Execute(command)
	if err != nil {
		return 0, err
	}
	go io.Copy(cmd.Stdin, stdin)
	go io.Copy(stdout, cmd.Stdout)
	go io.Copy(stderr, cmd.Stderr)
	cmd.Wait()
	return cmd.ExitCode(), cmd.err
}
