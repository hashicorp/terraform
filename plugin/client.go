package plugin

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	tfrpc "github.com/hashicorp/terraform/rpc"
)

// If this is true, then the "unexpected EOF" panic will not be
// raised throughout the clients.
var Killed = false

// This is a slice of the "managed" clients which are cleaned up when
// calling Cleanup
var managedClients = make([]*Client, 0, 5)

// Client handles the lifecycle of a plugin application, determining its
// RPC address, and returning various types of Terraform interface implementations
// across the multi-process communication layer.
type Client struct {
	config      *ClientConfig
	exited      bool
	doneLogging chan struct{}
	l           sync.Mutex
	address     net.Addr
	client      *tfrpc.Client
}

// ClientConfig is the configuration used to initialize a new
// plugin client. After being used to initialize a plugin client,
// that configuration must not be modified again.
type ClientConfig struct {
	// The unstarted subprocess for starting the plugin.
	Cmd *exec.Cmd

	// Managed represents if the client should be managed by the
	// plugin package or not. If true, then by calling CleanupClients,
	// it will automatically be cleaned up. Otherwise, the client
	// user is fully responsible for making sure to Kill all plugin
	// clients. By default the client is _not_ managed.
	Managed bool

	// The minimum and maximum port to use for communicating with
	// the subprocess. If not set, this defaults to 10,000 and 25,000
	// respectively.
	MinPort, MaxPort uint

	// StartTimeout is the timeout to wait for the plugin to say it
	// has started successfully.
	StartTimeout time.Duration

	// If non-nil, then the stderr of the client will be written to here
	// (as well as the log).
	Stderr io.Writer
}

// This makes sure all the managed subprocesses are killed and properly
// logged. This should be called before the parent process running the
// plugins exits.
//
// This must only be called _once_.
func CleanupClients() {
	// Set the killed to true so that we don't get unexpected panics
	Killed = true

	// Kill all the managed clients in parallel and use a WaitGroup
	// to wait for them all to finish up.
	var wg sync.WaitGroup
	for _, client := range managedClients {
		wg.Add(1)

		go func(client *Client) {
			client.Kill()
			wg.Done()
		}(client)
	}

	log.Println("[DEBUG] waiting for all plugin processes to complete...")
	wg.Wait()
}

// Creates a new plugin client which manages the lifecycle of an external
// plugin and gets the address for the RPC connection.
//
// The client must be cleaned up at some point by calling Kill(). If
// the client is a managed client (created with NewManagedClient) you
// can just call CleanupClients at the end of your program and they will
// be properly cleaned.
func NewClient(config *ClientConfig) (c *Client) {
	if config.MinPort == 0 && config.MaxPort == 0 {
		config.MinPort = 10000
		config.MaxPort = 25000
	}

	if config.StartTimeout == 0 {
		config.StartTimeout = 1 * time.Minute
	}

	if config.Stderr == nil {
		config.Stderr = ioutil.Discard
	}

	c = &Client{config: config}
	if config.Managed {
		managedClients = append(managedClients, c)
	}

	return
}

// Client returns an RPC client for the plugin.
//
// Subsequent calls to this will return the same RPC client.
func (c *Client) Client() (*tfrpc.Client, error) {
	addr, err := c.Start()
	if err != nil {
		return nil, err
	}

	c.l.Lock()
	defer c.l.Unlock()

	if c.client != nil {
		return c.client, nil
	}

	c.client, err = tfrpc.Dial(addr.Network(), addr.String())
	if err != nil {
		return nil, err
	}

	return c.client, nil
}

// Tells whether or not the underlying process has exited.
func (c *Client) Exited() bool {
	c.l.Lock()
	defer c.l.Unlock()
	return c.exited
}

// End the executing subprocess (if it is running) and perform any cleanup
// tasks necessary such as capturing any remaining logs and so on.
//
// This method blocks until the process successfully exits.
//
// This method can safely be called multiple times.
func (c *Client) Kill() {
	cmd := c.config.Cmd

	if cmd.Process == nil {
		return
	}

	cmd.Process.Kill()

	// Wait for the client to finish logging so we have a complete log
	<-c.doneLogging
}

// Starts the underlying subprocess, communicating with it to negotiate
// a port for RPC connections, and returning the address to connect via RPC.
//
// This method is safe to call multiple times. Subsequent calls have no effect.
// Once a client has been started once, it cannot be started again, even if
// it was killed.
func (c *Client) Start() (addr net.Addr, err error) {
	c.l.Lock()
	defer c.l.Unlock()

	if c.address != nil {
		return c.address, nil
	}

	c.doneLogging = make(chan struct{})

	env := []string{
		fmt.Sprintf("%s=%s", MagicCookieKey, MagicCookieValue),
		fmt.Sprintf("TF_PLUGIN_MIN_PORT=%d", c.config.MinPort),
		fmt.Sprintf("TF_PLUGIN_MAX_PORT=%d", c.config.MaxPort),
	}

	stdout_r, stdout_w := io.Pipe()
	stderr_r, stderr_w := io.Pipe()

	cmd := c.config.Cmd
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = stderr_w
	cmd.Stdout = stdout_w

	log.Printf("[DEBUG] Starting plugin: %s %#v", cmd.Path, cmd.Args)
	err = cmd.Start()
	if err != nil {
		return
	}

	// Make sure the command is properly cleaned up if there is an error
	defer func() {
		r := recover()

		if err != nil || r != nil {
			cmd.Process.Kill()
		}

		if r != nil {
			panic(r)
		}
	}()

	// Start goroutine to wait for process to exit
	exitCh := make(chan struct{})
	go func() {
		// Make sure we close the write end of our stderr/stdout so
		// that the readers send EOF properly.
		defer stderr_w.Close()
		defer stdout_w.Close()

		// Wait for the command to end.
		cmd.Wait()

		// Log and make sure to flush the logs write away
		log.Printf("[DEBUG] %s: plugin process exited\n", cmd.Path)
		os.Stderr.Sync()

		// Mark that we exited
		close(exitCh)

		// Set that we exited, which takes a lock
		c.l.Lock()
		defer c.l.Unlock()
		c.exited = true
	}()

	// Start goroutine that logs the stderr
	go c.logStderr(stderr_r)

	// Start a goroutine that is going to be reading the lines
	// out of stdout
	linesCh := make(chan []byte)
	go func() {
		defer close(linesCh)

		buf := bufio.NewReader(stdout_r)
		for {
			line, err := buf.ReadBytes('\n')
			if line != nil {
				linesCh <- line
			}

			if err == io.EOF {
				return
			}
		}
	}()

	// Make sure after we exit we read the lines from stdout forever
	// so they don't block since it is an io.Pipe
	defer func() {
		go func() {
			for _ = range linesCh {
			}
		}()
	}()

	// Some channels for the next step
	timeout := time.After(c.config.StartTimeout)

	// Start looking for the address
	log.Printf("[DEBUG] Waiting for RPC address for: %s", cmd.Path)
	select {
	case <-timeout:
		err = errors.New("timeout while waiting for plugin to start")
	case <-exitCh:
		err = errors.New("plugin exited before we could connect")
	case lineBytes := <-linesCh:
		// Trim the line and split by "|" in order to get the parts of
		// the output.
		line := strings.TrimSpace(string(lineBytes))
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			err = fmt.Errorf("Unrecognized remote plugin message: %s", line)
			return
		}

		// Test the API version
		if parts[0] != APIVersion {
			err = fmt.Errorf("Incompatible API version with plugin. "+
				"Plugin version: %s, Ours: %s", parts[0], APIVersion)
			return
		}

		switch parts[1] {
		case "tcp":
			addr, err = net.ResolveTCPAddr("tcp", parts[2])
		case "unix":
			addr, err = net.ResolveUnixAddr("unix", parts[2])
		default:
			err = fmt.Errorf("Unknown address type: %s", parts[1])
		}
	}

	c.address = addr
	return
}

func (c *Client) logStderr(r io.Reader) {
	bufR := bufio.NewReader(r)
	for {
		line, err := bufR.ReadString('\n')
		if line != "" {
			c.config.Stderr.Write([]byte(line))

			line = strings.TrimRightFunc(line, unicode.IsSpace)
			log.Printf("[DEBUG] %s: %s", filepath.Base(c.config.Cmd.Path), line)
		}

		if err == io.EOF {
			break
		}
	}

	// Flag that we've completed logging for others
	close(c.doneLogging)
}
