package plugin

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync/atomic"
)

// CoreProtocolVersion is the ProtocolVersion of the plugin system itself.
// We will increment this whenever we change any protocol behavior. This
// will invalidate any prior plugins but will at least allow us to iterate
// on the core in a safe way. We will do our best to do this very
// infrequently.
const CoreProtocolVersion = 1

// HandshakeConfig is the configuration used by client and servers to
// handshake before starting a plugin connection. This is embedded by
// both ServeConfig and ClientConfig.
//
// In practice, the plugin host creates a HandshakeConfig that is exported
// and plugins then can easily consume it.
type HandshakeConfig struct {
	// ProtocolVersion is the version that clients must match on to
	// agree they can communicate. This should match the ProtocolVersion
	// set on ClientConfig when using a plugin.
	ProtocolVersion uint

	// MagicCookieKey and value are used as a very basic verification
	// that a plugin is intended to be launched. This is not a security
	// measure, just a UX feature. If the magic cookie doesn't match,
	// we show human-friendly output.
	MagicCookieKey   string
	MagicCookieValue string
}

// ServeConfig configures what sorts of plugins are served.
type ServeConfig struct {
	// HandshakeConfig is the configuration that must match clients.
	HandshakeConfig

	// Plugins are the plugins that are served.
	Plugins map[string]Plugin
}

// Serve serves the plugins given by ServeConfig.
//
// Serve doesn't return until the plugin is done being executed. Any
// errors will be outputted to the log.
//
// This is the method that plugins should call in their main() functions.
func Serve(opts *ServeConfig) {
	// Validate the handshake config
	if opts.MagicCookieKey == "" || opts.MagicCookieValue == "" {
		fmt.Fprintf(os.Stderr,
			"Misconfigured ServeConfig given to serve this plugin: no magic cookie\n"+
				"key or value was set. Please notify the plugin author and report\n"+
				"this as a bug.\n")
		os.Exit(1)
	}

	// First check the cookie
	if os.Getenv(opts.MagicCookieKey) != opts.MagicCookieValue {
		fmt.Fprintf(os.Stderr,
			"This binary is a plugin. These are not meant to be executed directly.\n"+
				"Please execute the program that consumes these plugins, which will\n"+
				"load any plugins automatically\n")
		os.Exit(1)
	}

	// Logging goes to the original stderr
	log.SetOutput(os.Stderr)

	// Create our new stdout, stderr files. These will override our built-in
	// stdout/stderr so that it works across the stream boundary.
	stdout_r, stdout_w, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing plugin: %s\n", err)
		os.Exit(1)
	}
	stderr_r, stderr_w, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing plugin: %s\n", err)
		os.Exit(1)
	}

	// Register a listener so we can accept a connection
	listener, err := serverListener()
	if err != nil {
		log.Printf("[ERR] plugin: plugin init: %s", err)
		return
	}
	defer listener.Close()

	// Create the RPC server to dispense
	server := &RPCServer{
		Plugins: opts.Plugins,
		Stdout:  stdout_r,
		Stderr:  stderr_r,
	}

	// Output the address and service name to stdout so that core can bring it up.
	log.Printf("[DEBUG] plugin: plugin address: %s %s\n",
		listener.Addr().Network(), listener.Addr().String())
	fmt.Printf("%d|%d|%s|%s\n",
		CoreProtocolVersion,
		opts.ProtocolVersion,
		listener.Addr().Network(),
		listener.Addr().String())
	os.Stdout.Sync()

	// Eat the interrupts
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		var count int32 = 0
		for {
			<-ch
			newCount := atomic.AddInt32(&count, 1)
			log.Printf(
				"[DEBUG] plugin: received interrupt signal (count: %d). Ignoring.",
				newCount)
		}
	}()

	// Set our new out, err
	os.Stdout = stdout_w
	os.Stderr = stderr_w

	// Serve
	server.Accept(listener)
}

func serverListener() (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return serverListener_tcp()
	}

	return serverListener_unix()
}

func serverListener_tcp() (net.Listener, error) {
	minPort, err := strconv.ParseInt(os.Getenv("PLUGIN_MIN_PORT"), 10, 32)
	if err != nil {
		return nil, err
	}

	maxPort, err := strconv.ParseInt(os.Getenv("PLUGIN_MAX_PORT"), 10, 32)
	if err != nil {
		return nil, err
	}

	for port := minPort; port <= maxPort; port++ {
		address := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			return listener, nil
		}
	}

	return nil, errors.New("Couldn't bind plugin TCP listener")
}

func serverListener_unix() (net.Listener, error) {
	tf, err := ioutil.TempFile("", "plugin")
	if err != nil {
		return nil, err
	}
	path := tf.Name()

	// Close the file and remove it because it has to not exist for
	// the domain socket.
	if err := tf.Close(); err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}

	return net.Listen("unix", path)
}
