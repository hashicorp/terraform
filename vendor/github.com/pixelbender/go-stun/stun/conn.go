package stun

import (
	"bufio"
	"io"
	"net"
	"time"
)

// Config represents a STUN connection configuration.
type Config struct {
	// GetAuthKey returns a key for a MESSAGE-INTEGRITY attribute generation and validation.
	// Key = MD5(username ":" realm ":" SASLprep(password)) for long-term credentials.
	// Key = SASLprep(password) for short-term credentials.
	// SASLprep is defined in RFC 4013.
	// The Username and Password fields are ignored if GetAuthKey is defined.
	GetAuthKey func(attrs Attributes) ([]byte, error)

	// GetAttributeCodec returns STUN attribute codec for the specified attribute type.
	// Using stun.GetAttributeCodec if GetAttributeCodec is nil.
	GetAttributeCodec func(at uint16) AttrCodec

	// Fingerprint controls whether a FINGERPRINT attribute will be generated.
	Fingerprint bool

	// Software is a value for SOFTWARE attribute.
	Software string
}

func (c *Config) getAuthKey(attrs Attributes) ([]byte, error) {
	if c != nil && c.GetAuthKey != nil {
		return c.GetAuthKey(attrs)
	}
	return nil, nil
}

func (c *Config) getAttrCodec(at uint16) AttrCodec {
	if c != nil && c.GetAttributeCodec != nil {
		return c.GetAttributeCodec(at)
	}
	return GetAttributeCodec(at)
}

var DefaultConfig = &Config{
	GetAttributeCodec: GetAttributeCodec,
}

// A Conn represents the STUN connection and implements the STUN protocol over net.Conn interface.
type Conn struct {
	net.Conn
	config   *Config
	dec      *Decoder
	enc      *Encoder
	cr       connReader
	reliable bool
	key      []byte
}

// NewConn creates a Conn connection over the c with specified configuration.
func NewConn(inner net.Conn, config *Config) *Conn {
	if config == nil {
		config = DefaultConfig
	}
	c := &Conn{
		Conn:   inner,
		config: config,
		dec:    NewDecoder(config),
		enc:    NewEncoder(config),
	}
	if _, ok := inner.(net.PacketConn); ok {
		c.cr = newPacketReader(inner)
		c.reliable = false
	} else {
		c.cr = newStreamReader(inner)
		c.reliable = true
	}
	return c
}

// ReadMessage reads STUN messages from the connection.
func (c *Conn) ReadMessage() (*Message, error) {
	b, err := c.cr.PeekMessageBytes()
	if err != nil {
		return nil, err
	}
	msg, err := c.dec.Decode(b, c.key)
	return msg, err
}

// WriteMessage writes the STUN message to the connection.
func (c *Conn) WriteMessage(msg *Message) error {
	b, err := c.enc.Encode(msg)
	if err != nil {
		return err
	}
	if _, err = c.Write(b); err != nil {
		return err
	}
	return nil
}

type connReader interface {
	// PeekMessageBytes returns the bytes, containing a STUN message.
	// The bytes stop being valid at the next read call.
	PeekMessageBytes() ([]byte, error)
}

var bufferSize = 1400

// streamReader reads a STUN message transmitted over a stream-oriented network.
type streamReader struct {
	*bufio.Reader
	r    io.Reader
	skip int
}

func newStreamReader(r io.Reader) *streamReader {
	if tcp, ok := r.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(30 * time.Second)
	}
	return &streamReader{bufio.NewReaderSize(r, bufferSize), r, 0}
}

func (c *streamReader) Read(b []byte) (int, error) {
	c.discard()
	return c.Read(b)
}

func (c *streamReader) PeekMessageBytes() ([]byte, error) {
	c.discard()
	h, err := c.Peek(4)
	if err != nil {
		return nil, err
	}
	if be.Uint16(h)&0xc000 != 0 {
		return nil, ErrFormat
	}
	n := int(be.Uint16(h[2:])) + 20
	b, err := c.Peek(n)
	if err != nil {
		return nil, err
	}
	c.skip = n
	return b, nil
}

func (c *streamReader) discard() {
	if c.skip > 0 {
		c.Discard(c.skip)
		c.skip = 0
	}
}

// packetConn reads a STUN message transmitted over a packet-oriented network.
type packetReader struct {
	io.Reader
	buf []byte
}

func newPacketReader(r io.Reader) *packetReader {
	return &packetReader{r, make([]byte, bufferSize)}
}

func (c *packetReader) PeekMessageBytes() ([]byte, error) {
	n, err := c.Read(c.buf)
	if err != nil {
		return nil, err
	}
	b := c.buf[:n]
	l := int(be.Uint16(b[2:])) + 20
	if n < l {
		return nil, ErrTruncated
	}
	return b, nil
}
