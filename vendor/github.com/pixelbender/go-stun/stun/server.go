package stun

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"time"
)

// A Handler handles a STUN message.
type Handler interface {
	ServeSTUN(rw ResponseWriter, r *Message)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as STUN handlers.
type HandlerFunc func(rw ResponseWriter, r *Message)

// ServeSTUN calls f(rw, r).
func (f HandlerFunc) ServeSTUN(rw ResponseWriter, r *Message) {
	f(rw, r)
}

type ResponseWriter interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	// WriteResponse writes STUN response within the transaction.
	WriteMessage(msg *Message) error
}

// Server represents a STUN server.
type Server struct {
	*Config
	Realm   string
	Handler Handler
}

func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig
	}
	return &Server{Config: config}
}

// ListenAndServe listens on the network address and calls handler to serve requests.
// Accepted connections are configured to enable TCP keep-alives.
func (srv *Server) ListenAndServe(network, addr string) error {
	switch network {
	case "tcp", "tcp4", "tcp6":
		l, err := net.Listen(network, addr)
		if err != nil {
			return err
		}
		return srv.Serve(l)
	case "udp", "udp4", "udp6":
		l, err := net.ListenPacket(network, addr)
		if err != nil {
			return err
		}
		return srv.ServePacket(l)
	}
	return fmt.Errorf("stun: listen unsupported network %v", network)
}

// ListenAndServeTLS listens on the network address secured by TLS and calls handler to serve requests.
// Accepted connections are configured to enable TCP keep-alives.
func (srv *Server) ListenAndServeTLS(network, addr, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	l, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	l = tls.NewListener(l, config)
	return srv.Serve(l)
}

// ServePacket receives incoming packets on the packet-oriented network listener and calls handler to serve STUN requests.
// Multiple goroutines may invoke ServePacket on the same PacketConn simultaneously.
func (srv *Server) ServePacket(l net.PacketConn) error {
	enc := NewEncoder(srv.Config)
	dec := NewDecoder(srv.Config)
	buf := make([]byte, bufferSize)

	for {
		n, addr, err := l.ReadFrom(buf)
		if err != nil {
			return err
		}
		msg, err := dec.Decode(buf[:n], nil)
		rw := &packetResponseWriter{l, msg, addr, enc}
		srv.serve(rw, msg, err)
	}
}

// Serve accepts incoming connection on the listener and calls handler to serve STUN requests.
// Multiple goroutines may invoke Serve on the same Listener simultaneously.
func (srv *Server) Serve(l net.Listener) error {
	for {
		c, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(time.Millisecond)
				continue
			}
			return err
		}
		go srv.serveConn(c)
	}
}

func (srv *Server) serveConn(conn net.Conn) error {
	c := NewConn(conn, srv.Config)
	defer c.Close()
	for {
		msg, err := c.ReadMessage()
		if err != nil {
			return err
		}
		srv.serve(&connResponseWriter{c, msg}, msg, err)
	}
}

func (srv *Server) serve(rw ResponseWriter, r *Message, err error) error {
	if r.IsType(TypeRequest) || r.IsType(TypeIndication) {
		if srv.GetAuthKey != nil {
			if !r.Attributes.Has(AttrMessageIntegrity) || !r.Attributes.Has(AttrMessageIntegrity) {
				err = ErrUnauthorized
			}
		}
	}
	if err != nil {
		switch err {
		case ErrUnauthorized, ErrIncorrectFingerprint:
			// TODO: store nonce
			nonce := make([]byte, 8)
			rand.Read(nonce)
			return rw.WriteMessage(&Message{
				Method: r.Method | TypeError,
				Attributes: Attributes{
					AttrErrorCode: NewError(CodeUnauthorized),
					AttrRealm:     srv.Realm,
					AttrNonce:     hex.EncodeToString(nonce),
				},
			})
		}
		if unk, ok := err.(ErrUnknownAttrs); ok {
			return rw.WriteMessage(&Message{
				Method: r.Method | TypeError,
				Attributes: Attributes{
					AttrErrorCode:         NewError(CodeUnknownAttribute),
					AttrUnknownAttributes: unk,
				},
			})
		}
		// TODO: log error
		return nil
	}
	if h := srv.Handler; h != nil {
		h.ServeSTUN(rw, r)
	} else {
		srv.ServeSTUN(rw, r)
	}
	return nil
}

// ServeSTUN responds to the simple STUN binding request.
func (srv *Server) ServeSTUN(rw ResponseWriter, r *Message) {
	switch r.Method {
	case MethodBinding:
		rw.WriteMessage(&Message{
			Method: r.Method | TypeResponse,
			Attributes: Attributes{
				AttrXorMappedAddress: rw.RemoteAddr(),
				AttrMappedAddress:    rw.RemoteAddr(),
				AttrResponseOrigin:   rw.LocalAddr(),
				// TODO: add other address
				// TODO: handle change request
			},
		})
	}
}

type connResponseWriter struct {
	*Conn
	msg *Message
}

func (w *connResponseWriter) WriteMessage(msg *Message) error {
	if msg.Transaction == nil {
		msg.Transaction = w.msg.Transaction
	}
	if msg.Key == nil {
		msg.Key = w.msg.Key
	}
	return w.Conn.WriteMessage(msg)
}

type packetResponseWriter struct {
	net.PacketConn
	msg  *Message
	addr net.Addr
	enc  *Encoder
}

func (w *packetResponseWriter) RemoteAddr() net.Addr {
	return w.addr
}

func (w *packetResponseWriter) WriteMessage(msg *Message) error {
	if msg.Transaction == nil {
		msg.Transaction = w.msg.Transaction
	}
	if msg.Key == nil {
		msg.Key = w.msg.Key
	}
	b, err := w.enc.Encode(msg)
	if err != nil {
		return err
	}
	if _, err = w.WriteTo(b, w.addr); err != nil {
		return err
	}
	return nil
}
