package winrm

import "net"

// Parameters struct defines
// metadata information and http transport config
type Parameters struct {
	Timeout            string
	Locale             string
	EnvelopeSize       int
	TransportDecorator func() Transporter
	Dial               func(network, addr string) (net.Conn, error)
}

// DefaultParameters return constant config
// of type Parameters
var DefaultParameters = NewParameters("PT60S", "en-US", 153600)

// NewParameters return new struct of type Parameters
// this struct makes the configuration for the request, size message, etc.
func NewParameters(timeout, locale string, envelopeSize int) *Parameters {
	return &Parameters{
		Timeout:      timeout,
		Locale:       locale,
		EnvelopeSize: envelopeSize,
	}
}
