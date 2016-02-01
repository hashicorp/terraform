package winrm

import (
	"net/http"
)

type Parameters struct {
	Timeout            string
	Locale             string
	EnvelopeSize       int
	TransportDecorator func(*http.Transport) http.RoundTripper
}

func DefaultParameters() *Parameters {
	return NewParameters("PT60S", "en-US", 153600)
}

func NewParameters(timeout string, locale string, envelopeSize int) *Parameters {
	return &Parameters{Timeout: timeout, Locale: locale, EnvelopeSize: envelopeSize}
}
