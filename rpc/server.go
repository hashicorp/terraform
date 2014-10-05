package rpc

import (
	"io"
	"log"
	"net"
	"net/rpc"

	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/yamux"
)

// Server listens for network connections and then dispenses interface
// implementations for Terraform over net/rpc.
type Server struct {
	ProviderFunc    ProviderFunc
	ProvisionerFunc ProvisionerFunc
}

// ProviderFunc creates terraform.ResourceProviders when they're requested
// from the server.
type ProviderFunc func() terraform.ResourceProvider

// ProvisionerFunc creates terraform.ResourceProvisioners when they're requested
// from the server.
type ProvisionerFunc func() terraform.ResourceProvisioner

// Accept accepts connections on a listener and serves requests for
// each incoming connection. Accept blocks; the caller typically invokes
// it in a go statement.
func (s *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Printf("[ERR] plugin server: %s", err)
			return
		}

		go s.ServeConn(conn)
	}
}

// ServeConn runs a single connection.
//
// ServeConn blocks, serving the connection until the client hangs up.
func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	// First create the yamux server to wrap this connection
	mux, err := yamux.Server(conn, nil)
	if err != nil {
		conn.Close()
		log.Printf("[ERR] plugin: %s", err)
		return
	}

	// Accept the control connection
	control, err := mux.Accept()
	if err != nil {
		mux.Close()
		log.Printf("[ERR] plugin: %s", err)
		return
	}

	// Create the broker and start it up
	broker := newMuxBroker(mux)
	go broker.Run()

	// Use the control connection to build the dispenser and serve the
	// connection.
	server := rpc.NewServer()
	server.RegisterName("Dispenser", &dispenseServer{
		ProviderFunc:    s.ProviderFunc,
		ProvisionerFunc: s.ProvisionerFunc,

		broker: broker,
	})
	server.ServeConn(control)
}

// dispenseServer dispenses variousinterface implementations for Terraform.
type dispenseServer struct {
	ProviderFunc    ProviderFunc
	ProvisionerFunc ProvisionerFunc

	broker *muxBroker
}

func (d *dispenseServer) ResourceProvider(
	args interface{}, response *uint32) error {
	id := d.broker.NextId()
	*response = id

	go func() {
		conn, err := d.broker.Accept(id)
		if err != nil {
			log.Printf("[ERR] Plugin dispense: %s", err)
			return
		}

		serve(conn, "ResourceProvider", &ResourceProviderServer{
			Broker:   d.broker,
			Provider: d.ProviderFunc(),
		})
	}()

	return nil
}

func (d *dispenseServer) ResourceProvisioner(
	args interface{}, response *uint32) error {
	id := d.broker.NextId()
	*response = id

	go func() {
		conn, err := d.broker.Accept(id)
		if err != nil {
			log.Printf("[ERR] Plugin dispense: %s", err)
			return
		}

		serve(conn, "ResourceProvisioner", &ResourceProvisionerServer{
			Broker:      d.broker,
			Provisioner: d.ProvisionerFunc(),
		})
	}()

	return nil
}

func acceptAndServe(mux *muxBroker, id uint32, n string, v interface{}) {
	conn, err := mux.Accept(id)
	if err != nil {
		log.Printf("[ERR] Plugin acceptAndServe: %s", err)
		return
	}

	serve(conn, n, v)
}

func serve(conn io.ReadWriteCloser, name string, v interface{}) {
	server := rpc.NewServer()
	if err := server.RegisterName(name, v); err != nil {
		log.Printf("[ERR] Plugin dispense: %s", err)
		return
	}

	server.ServeConn(conn)
}
