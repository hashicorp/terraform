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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

// If this is 1, then we've called CleanupClients. This can be used
// by plugin RPC implementations to change error behavior since you
// can expected network connection errors at this point. This should be
// read by using sync/atomic.
var Killed uint32 = 0

// This is a slice of the "managed" clients which are cleaned up when
// calling Cleanup
var managedClients = make([]*Client, 0, 5)
var managedClientsLock sync.Mutex

// Error types
var (
	// ErrProcessNotFound is returned when a client is instantiated to
	// reattach to an existing process and it isn't found.
	ErrProcessNotFound = errors.New("Reattachment process not found")
)

// Client handles the lifecycle of a plugin application. It launches
// plugins, connects to them, dispenses interface implementations, and handles
// killing the process.
//
// Plugin hosts should use one Client for each plugin executable. To
// dispense a plugin type, use the `Client.Client` function, and then
// cal `Dispense`. This awkward API is mostly historical but is used to split
// the client that deals with subprocess management and the client that
// does RPC management.
//
// See NewClient and ClientConfig for using a Client.
type Client struct {
	config      *ClientConfig
	exited      bool
	doneLogging chan struct{}
	l           sync.Mutex
	address     net.Addr
	process     *os.Process
	client      *RPCClient
}

// ClientConfig is the configuration used to initialize a new
// plugin client. After being used to initialize a plugin client,
// that configuration must not be modified again.
type ClientConfig struct {
	// HandshakeConfig is the configuration that must match servers.
	HandshakeConfig

	// Plugins are the plugins that can be consumed.
	Plugins map[string]Plugin

	// One of the following must be set, but not both.
	//
	// Cmd is the unstarted subprocess for starting the plugin. If this is
	// set, then the Client starts the plugin process on its own and connects
	// to it.
	//
	// Reattach is configuration for reattaching to an existing plugin process
	// that is already running. This isn't common.
	Cmd      *exec.Cmd
	Reattach *ReattachConfig

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
	// (as well as the log). This is the original os.Stderr of the subprocess.
	// This isn't the output of synced stderr.
	Stderr io.Writer

	// SyncStdout, SyncStderr can be set to override the
	// respective os.Std* values in the plugin. Care should be taken to
	// avoid races here. If these are nil, then this will automatically be
	// hooked up to os.Stdin, Stdout, and Stderr, respectively.
	//
	// If the default values (nil) are used, then this package will not
	// sync any of these streams.
	SyncStdout io.Writer
	SyncStderr io.Writer
}

// ReattachConfig is used to configure a client to reattach to an
// already-running plugin process. You can retrieve this information by
// calling ReattachConfig on Client.
type ReattachConfig struct {
	Addr net.Addr
	Pid  int
}

// This makes sure all the managed subprocesses are killed and properly
// logged. This should be called before the parent process running the
// plugins exits.
//
// This must only be called _once_.
func CleanupClients() {
	// Set the killed to true so that we don't get unexpected panics
	atomic.StoreUint32(&Killed, 1)

	// Kill all the managed clients in parallel and use a WaitGroup
	// to wait for them all to finish up.
	var wg sync.WaitGroup
	managedClientsLock.Lock()
	for _, client := range managedClients {
		wg.Add(1)

		go func(client *Client) {
			client.Kill()
			wg.Done()
		}(client)
	}
	managedClientsLock.Unlock()

	log.Println("[DEBUG] plugin: waiting for all plugin processes to complete...")
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

	if config.SyncStdout == nil {
		config.SyncStdout = ioutil.Discard
	}
	if config.SyncStderr == nil {
		config.SyncStderr = ioutil.Discard
	}

	c = &Client{config: config}
	if config.Managed {
		managedClientsLock.Lock()
		managedClients = append(managedClients, c)
		managedClientsLock.Unlock()
	}

	return
}

// Client returns an RPC client for the plugin.
//
// Subsequent calls to this will return the same RPC client.
func (c *Client) Client() (*RPCClient, error) {
	addr, err := c.Start()
	if err != nil {
		return nil, err
	}

	c.l.Lock()
	defer c.l.Unlock()

	if c.client != nil {
		return c.client, nil
	}

	// Connect to the client
	conn, err := net.Dial(addr.Network(), addr.String())
	if err != nil {
		return nil, err
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		// Make sure to set keep alive so that the connection doesn't die
		tcpConn.SetKeepAlive(true)
	}

	// Create the actual RPC client
	c.client, err = NewRPCClient(conn, c.config.Plugins)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Begin the stream syncing so that stdin, out, err work properly
	err = c.client.SyncStreams(
		c.config.SyncStdout,
		c.config.SyncStderr)
	if err != nil {
		c.client.Close()
		c.client = nil
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
	// Grab a lock to read some private fields.
	c.l.Lock()
	process := c.process
	addr := c.address
	doneCh := c.doneLogging
	c.l.Unlock()

	// If there is no process, we never started anything. Nothing to kill.
	if process == nil {
		return
	}

	// We need to check for address here. It is possible that the plugin
	// started (process != nil) but has no address (addr == nil) if the
	// plugin failed at startup. If we do have an address, we need to close
	// the plugin net connections.
	graceful := false
	if addr != nil {
		// Close the client to cleanly exit the process.
		client, err := c.Client()
		if err == nil {
			err = client.Close()

			// If there is no error, then we attempt to wait for a graceful
			// exit. If there was an error, we assume that graceful cleanup
			// won't happen and just force kill.
			graceful = err == nil
			if err != nil {
				// If there was an error just log it. We're going to force
				// kill in a moment anyways.
				log.Printf(
					"[WARN] plugin: error closing client during Kill: %s", err)
			}
		}
	}

	// If we're attempting a graceful exit, then we wait for a short period
	// of time to allow that to happen. To wait for this we just wait on the
	// doneCh which would be closed if the process exits.
	if graceful {
		select {
		case <-doneCh:
			return
		case <-time.After(250 * time.Millisecond):
		}
	}

	// If graceful exiting failed, just kill it
	process.Kill()

	// Wait for the client to finish logging so we have a complete log
	<-doneCh
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

	// If one of cmd or reattach isn't set, then it is an error. We wrap
	// this in a {} for scoping reasons, and hopeful that the escape
	// analysis will pop the stock here.
	{
		cmdSet := c.config.Cmd != nil
		attachSet := c.config.Reattach != nil
		if cmdSet == attachSet {
			return nil, fmt.Errorf("Only one of Cmd or Reattach must be set")
		}
	}

	// Create the logging channel for when we kill
	c.doneLogging = make(chan struct{})

	if c.config.Reattach != nil {
		// Verify the process still exists. If not, then it is an error
		p, err := os.FindProcess(c.config.Reattach.Pid)
		if err != nil {
			return nil, err
		}

		// Attempt to connect to the addr since on Unix systems FindProcess
		// doesn't actually return an error if it can't find the process.
		conn, err := net.Dial(
			c.config.Reattach.Addr.Network(),
			c.config.Reattach.Addr.String())
		if err != nil {
			p.Kill()
			return nil, ErrProcessNotFound
		}
		conn.Close()

		// Goroutine to mark exit status
		go func(pid int) {
			// Wait for the process to die
			pidWait(pid)

			// Log so we can see it
			log.Printf("[DEBUG] plugin: reattached plugin process exited\n")

			// Mark it
			c.l.Lock()
			defer c.l.Unlock()
			c.exited = true

			// Close the logging channel since that doesn't work on reattach
			close(c.doneLogging)
		}(p.Pid)

		// Set the address and process
		c.address = c.config.Reattach.Addr
		c.process = p

		return c.address, nil
	}

	env := []string{
		fmt.Sprintf("%s=%s", c.config.MagicCookieKey, c.config.MagicCookieValue),
		fmt.Sprintf("PLUGIN_MIN_PORT=%d", c.config.MinPort),
		fmt.Sprintf("PLUGIN_MAX_PORT=%d", c.config.MaxPort),
	}

	stdout_r, stdout_w := io.Pipe()
	stderr_r, stderr_w := io.Pipe()

	cmd := c.config.Cmd
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = stderr_w
	cmd.Stdout = stdout_w

	log.Printf("[DEBUG] plugin: starting plugin: %s %#v", cmd.Path, cmd.Args)
	err = cmd.Start()
	if err != nil {
		return
	}

	// Set the process
	c.process = cmd.Process

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
		log.Printf("[DEBUG] plugin: %s: plugin process exited\n", cmd.Path)
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
	log.Printf("[DEBUG] plugin: waiting for RPC address for: %s", cmd.Path)
	select {
	case <-timeout:
		err = errors.New("timeout while waiting for plugin to start")
	case <-exitCh:
		err = errors.New("plugin exited before we could connect")
	case lineBytes := <-linesCh:
		// Trim the line and split by "|" in order to get the parts of
		// the output.
		line := strings.TrimSpace(string(lineBytes))
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			err = fmt.Errorf(
				"Unrecognized remote plugin message: %s\n\n"+
					"This usually means that the plugin is either invalid or simply\n"+
					"needs to be recompiled to support the latest protocol.", line)
			return
		}

		// Check the core protocol. Wrapped in a {} for scoping.
		{
			var coreProtocol int64
			coreProtocol, err = strconv.ParseInt(parts[0], 10, 0)
			if err != nil {
				err = fmt.Errorf("Error parsing core protocol version: %s", err)
				return
			}

			if int(coreProtocol) != CoreProtocolVersion {
				err = fmt.Errorf("Incompatible core API version with plugin. "+
					"Plugin version: %s, Ours: %d\n\n"+
					"To fix this, the plugin usually only needs to be recompiled.\n"+
					"Please report this to the plugin author.", parts[0], CoreProtocolVersion)
				return
			}
		}

		// Parse the protocol version
		var protocol int64
		protocol, err = strconv.ParseInt(parts[1], 10, 0)
		if err != nil {
			err = fmt.Errorf("Error parsing protocol version: %s", err)
			return
		}

		// Test the API version
		if uint(protocol) != c.config.ProtocolVersion {
			err = fmt.Errorf("Incompatible API version with plugin. "+
				"Plugin version: %s, Ours: %d", parts[1], c.config.ProtocolVersion)
			return
		}

		switch parts[2] {
		case "tcp":
			addr, err = net.ResolveTCPAddr("tcp", parts[3])
		case "unix":
			addr, err = net.ResolveUnixAddr("unix", parts[3])
		default:
			err = fmt.Errorf("Unknown address type: %s", parts[3])
		}
	}

	c.address = addr
	return
}

// ReattachConfig returns the information that must be provided to NewClient
// to reattach to the plugin process that this client started. This is
// useful for plugins that detach from their parent process.
//
// If this returns nil then the process hasn't been started yet. Please
// call Start or Client before calling this.
func (c *Client) ReattachConfig() *ReattachConfig {
	c.l.Lock()
	defer c.l.Unlock()

	if c.address == nil {
		return nil
	}

	if c.config.Cmd != nil && c.config.Cmd.Process == nil {
		return nil
	}

	// If we connected via reattach, just return the information as-is
	if c.config.Reattach != nil {
		return c.config.Reattach
	}

	return &ReattachConfig{
		Addr: c.address,
		Pid:  c.config.Cmd.Process.Pid,
	}
}

func (c *Client) logStderr(r io.Reader) {
	bufR := bufio.NewReader(r)
	for {
		line, err := bufR.ReadString('\n')
		if line != "" {
			c.config.Stderr.Write([]byte(line))

			line = strings.TrimRightFunc(line, unicode.IsSpace)
			log.Printf("[DEBUG] plugin: %s: %s", filepath.Base(c.config.Cmd.Path), line)
		}

		if err == io.EOF {
			break
		}
	}

	// Flag that we've completed logging for others
	close(c.doneLogging)
}
