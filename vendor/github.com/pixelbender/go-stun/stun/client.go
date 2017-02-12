package stun

import (
	"bytes"
	"crypto/rand"
	"net"
	"time"
)

var retransmissionTimeout = 500 * time.Millisecond
var defaultTimeout = 39500 * time.Millisecond

// Client is a STUN client.
type Client struct {
	*Conn
	Timeout time.Duration
}

func NewClient(c net.Conn, config *Config) *Client {
	return &Client{
		Conn:    NewConn(c, config),
		Timeout: defaultTimeout,
	}
}

// RoundTrip sends STUN request and waits for a STUN response within the transaction.
// Retransmits on unrelied over transport protocol connection.
// Tries to authorize
// No STUN redirect is supported.
// Returns STUN error responses as errors.
func (c *Client) RoundTrip(req *Message) (res *Message, err error) {
	if err = c.WriteMessage(req); err != nil {
		return
	}

	rto := retransmissionTimeout
	deadline := time.Now().Add(c.Timeout)

	if !c.reliable {
		c.SetReadDeadline(time.Now().Add(rto))
	} else if c.Timeout > 0 {
		c.SetReadDeadline(deadline)
	}

	for {
		res, err = c.ReadMessage()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() && !c.reliable && time.Now().Before(deadline) {
				// Retransmit
				if err = c.WriteMessage(req); err != nil {
					return
				}
				rto *= 2
				c.SetReadDeadline(time.Now().Add(rto))
				continue
			}
			return
		}
		if !bytes.Equal(req.Transaction, res.Transaction) {
			if c.reliable {
				return nil, ErrFormat
			}
			continue
		}
		if attr, ok := res.Attributes[AttrErrorCode]; ok {
			code := attr.(*Error)
			if code.Code == CodeUnauthorized && req.Key == nil {
				// TODO: store nonce
				req.Attributes[AttrRealm] = res.Attributes[AttrRealm]
				req.Attributes[AttrNonce] = res.Attributes[AttrNonce]
				req.Key, err = c.config.getAuthKey(req.Attributes)
				if err != nil {
					return
				}
				if req.Key != nil {
					// New transaction
					rand.Read(req.Transaction[4:])
					c.key = req.Key
					if err = c.WriteMessage(req); err != nil {
						return
					}
					continue
				}
			}
			// TODO: handle alternate server
		}
		return res, nil
	}
}

func GetServerAddress(h string, secure bool) string {
	host, port, err := net.SplitHostPort(h)
	if err != nil {
		host = h
	}
	if port == "" {
		if secure {
			port = "5478"
		} else {
			port = "3478"
		}
	}
	return net.JoinHostPort(host, port)
}
