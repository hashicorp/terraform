package stun

import (
	"crypto/md5"
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"strings"
)

var ErrUnsupportedScheme = errors.New("stun: unsupported scheme")
var ErrNoAddressResponse = errors.New("stun: no mapped address")

// Lookup connects to the given STUN URI and makes the STUN binding request.
// Returns the server reflexive transport address (mapped address).
func Lookup(uri, username, password string) (*Addr, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	switch strings.ToLower(u.Scheme) {
	case "stun":
		conn, err = net.Dial("udp", GetServerAddress(u.Opaque, false))
	case "stuns":
		conn, err = tls.Dial("tcp", GetServerAddress(u.Opaque, true), nil)
	default:
		err = ErrUnsupportedScheme
	}
	if err != nil {
		return nil, err
	}

	c := NewClient(conn, &Config{GetAuthKey: LongTermAuthKey(username, password)})
	defer c.Close()

	msg, err := c.RoundTrip(&Message{Method: MethodBinding})
	if err != nil {
		return nil, err
	}
	if addr, ok := msg.Attributes[AttrXorMappedAddress]; ok {
		return addr.(*Addr), nil
	} else if addr, ok := msg.Attributes[AttrMappedAddress]; ok {
		return addr.(*Addr), nil
	}
	return nil, ErrNoAddressResponse
}

// ListenAndServe listens on the network address and calls handler to serve requests.
func ListenAndServe(network, addr string, handler Handler) error {
	srv := &Server{Config: DefaultConfig, Handler: handler}
	return srv.ListenAndServe(network, addr)
}

// ListenAndServeTLS listens on the network address secured by TLS and calls handler to serve requests.
func ListenAndServeTLS(network, addr string, certFile, keyFile string, handler Handler) error {
	srv := &Server{Config: DefaultConfig, Handler: handler}
	return srv.ListenAndServeTLS(network, addr, certFile, keyFile)
}

func LongTermAuthKey(username, password string) func(attrs Attributes) ([]byte, error) {
	return func(attrs Attributes) ([]byte, error) {
		if attrs.Has(AttrRealm) {
			attrs[AttrUsername] = username
			h := md5.New()
			h.Write([]byte(username + ":" + attrs.String(AttrRealm) + ":" + password))
			return h.Sum(nil), nil
		}
		return nil, nil
	}
}
