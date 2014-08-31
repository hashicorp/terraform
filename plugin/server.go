package plugin

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync/atomic"

	tfrpc "github.com/hashicorp/terraform/rpc"
)

// The APIVersion is outputted along with the RPC address. The plugin
// client validates this API version and will show an error if it doesn't
// know how to speak it.
const APIVersion = "1"

// The "magic cookie" is used to verify that the user intended to
// actually run this binary. If this cookie isn't present as an
// environmental variable, then we bail out early with an error.
const MagicCookieKey = "TF_PLUGIN_MAGIC_COOKIE"
const MagicCookieValue = "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2"

func Serve(svc interface{}) error {
	// First check the cookie
	if os.Getenv(MagicCookieKey) != MagicCookieValue {
		fmt.Fprintf(os.Stderr,
			"This binary is a Terraform plugin. These are not meant to be\n"+
				"executed directly. Please execute `terraform`, which will load\n"+
				"any plugins automatically.\n")
		os.Exit(1)
	}

	// Create the server to serve our interface
	server := rpc.NewServer()

	// Register the service
	name, err := tfrpc.Register(server, svc)
	if err != nil {
		return err
	}

	// Register a listener so we can accept a connection
	listener, err := serverListener()
	if err != nil {
		return err
	}
	defer listener.Close()

	// Output the address and service name to stdout
	log.Printf("Plugin address: %s %s\n",
		listener.Addr().Network(), listener.Addr().String())
	fmt.Printf("%s|%s|%s|%s\n",
		APIVersion,
		listener.Addr().Network(),
		listener.Addr().String(),
		name)
	os.Stdout.Sync()

	// Accept a connection
	log.Println("Waiting for connection...")
	conn, err := listener.Accept()
	if err != nil {
		log.Printf("Error accepting connection: %s\n", err.Error())
		return err
	}

	// Eat the interrupts
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		var count int32 = 0
		for {
			<-ch
			newCount := atomic.AddInt32(&count, 1)
			log.Printf(
				"Received interrupt signal (count: %d). Ignoring.",
				newCount)
		}
	}()

	// Serve a single connection
	log.Println("Serving a plugin connection...")
	server.ServeConn(conn)
	return nil
}

func serverListener() (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return serverListener_tcp()
	}

	return serverListener_unix()
}

func serverListener_tcp() (net.Listener, error) {
	minPort, err := strconv.ParseInt(os.Getenv("TF_PLUGIN_MIN_PORT"), 10, 32)
	if err != nil {
		return nil, err
	}

	maxPort, err := strconv.ParseInt(os.Getenv("TF_PLUGIN_MAX_PORT"), 10, 32)
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
	tf, err := ioutil.TempFile("", "tf-plugin")
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
