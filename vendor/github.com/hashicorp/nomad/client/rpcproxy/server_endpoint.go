package rpcproxy

import (
	"fmt"
	"net"
	"strings"
)

const (
	defaultNomadRPCPort = "4647"
)

// EndpointKey is used in maps and for equality tests.  A key is based on endpoints.
type EndpointKey struct {
	name string
}

// Equal compares two EndpointKey objects
func (k *EndpointKey) Equal(x *EndpointKey) bool {
	return k.name == x.name
}

// ServerEndpoint contains the address information for to connect to a Nomad
// server.
//
// TODO(sean@): Server is stubbed out so that in the future it can hold a
// reference to Node (and ultimately Node.ID).
type ServerEndpoint struct {
	// Name is the unique lookup key for a Server instance
	Name string
	Host string
	Port string
	Addr net.Addr
}

// Key returns the corresponding Key
func (s *ServerEndpoint) Key() *EndpointKey {
	return &EndpointKey{
		name: s.Name,
	}
}

// NewServerEndpoint creates a new Server instance with a resolvable
// endpoint.  `name` can be either an IP address or a DNS name.  If `name` is
// a DNS name, it must be resolvable to an IP address (most inputs are IP
// addresses, not DNS names, but both work equally well when the name is
// resolvable).
func NewServerEndpoint(name string) (*ServerEndpoint, error) {
	s := &ServerEndpoint{
		Name: name,
	}

	var host, port string
	var err error
	host, port, err = net.SplitHostPort(name)
	if err == nil {
		s.Host = host
		s.Port = port
	} else {
		if strings.Contains(err.Error(), "missing port") {
			s.Host = name
			s.Port = defaultNomadRPCPort
		} else {
			return nil, err
		}
	}

	if s.Addr, err = net.ResolveTCPAddr("tcp", net.JoinHostPort(s.Host, s.Port)); err != nil {
		return nil, err
	}

	return s, err
}

// String returns a string representation of Server
func (s *ServerEndpoint) String() string {
	var addrStr, networkStr string
	if s.Addr != nil {
		addrStr = s.Addr.String()
		networkStr = s.Addr.Network()
	}

	return fmt.Sprintf("%s (%s:%s)", s.Name, networkStr, addrStr)
}
