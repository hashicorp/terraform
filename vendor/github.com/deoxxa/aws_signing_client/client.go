package aws_signing_client

import (
	"net/http"

	"strings"
	"time"

	"io/ioutil"
	"log"

	"bytes"

	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/private/protocol/rest"
)

type (
	// Signer implements the http.RoundTripper interface and houses an optional RoundTripper that will be called between
	// signing and response.
	Signer struct {
		transport http.RoundTripper
		v4        *v4.Signer
		service   string
		region    string
		logger    *log.Logger
	}

	// MissingSignerError is an implementation of the error interface that indicates that no AWS v4.Signer was
	// provided in order to create a client.
	MissingSignerError struct{}

	// MissingServiceError is an implementation of the error interface that indicates that no AWS service was
	// provided in order to create a client.
	MissingServiceError struct{}

	// MissingRegionError is an implementation of the error interface that indicates that no AWS region was
	// provided in order to create a client.
	MissingRegionError struct{}
)

var signer *Signer

// New obtains an HTTP client with a RoundTripper that signs AWS requests for the provided service. An
// existing client can be specified for the `client` value, or--if nil--a new HTTP client will be created.
func New(v4s *v4.Signer, client *http.Client, service string, region string) (*http.Client, error) {
	c := client
	switch {
	case v4s == nil:
		return nil, MissingSignerError{}
	case service == "":
		return nil, MissingServiceError{}
	case region == "":
		return nil, MissingRegionError{}
	case c == nil:
		c = http.DefaultClient
	}
	s := &Signer{
		transport: c.Transport,
		v4:        v4s,
		service:   service,
		region:    region,
		logger:    log.New(ioutil.Discard, "", 0),
	}
	if s.transport == nil {
		s.transport = http.DefaultTransport
	}
	signer = s
	c.Transport = s
	return c, nil
}

// SetDebugLog sets a logger for use in debugging requests and responses.
func SetDebugLog(l *log.Logger) {
	signer.logger = l
}

// RoundTrip implements the http.RoundTripper interface and is used to wrap HTTP requests in order to sign them for AWS
// API calls. The scheme for all requests will be changed to HTTPS.
func (s *Signer) RoundTrip(req *http.Request) (*http.Response, error) {
	if h, ok := req.Header["Authorization"]; ok && len(h) > 0 && strings.HasPrefix(h[0], "AWS4") {
		s.logger.Println("Received request to sign that is already signed. Skipping.")
		return s.transport.RoundTrip(req)
	}
	s.logger.Printf("Receiving request for signing: %+v", req)
	req.URL.Scheme = "https"
	if strings.Contains(req.URL.RawPath, "%2C") {
		s.logger.Printf("Escaping path for URL path '%s'", req.URL.RawPath)
		req.URL.RawPath = rest.EscapePath(req.URL.RawPath, false)
	}
	t := time.Now()
	req.Header.Set("Date", t.Format(time.RFC3339))
	s.logger.Printf("Final request to be signed: %+v", req)
	var err error
	switch req.Body {
	case nil:
		s.logger.Println("Signing request with no body...")
		_, err = s.v4.Sign(req, nil, s.service, s.region, t)
	default:
		d, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(d))
		s.logger.Println("Signing request with body...")
		_, err = s.v4.Sign(req, bytes.NewReader(d), s.service, s.region, t)
	}
	if err != nil {
		s.logger.Printf("Error while attempting to sign request: '%s'", err)
		return nil, err
	}
	s.logger.Printf("Signing succesful. Set header to: '%+v'", req.Header)
	s.logger.Printf("Sending signed request to RoundTripper: %+v", req)
	resp, err := s.transport.RoundTrip(req)
	if err != nil {
		s.logger.Printf("Error from RoundTripper.\n\n\tResponse: %+v\n\n\tError: '%s'", resp, err)
		return resp, err
	}
	respBody := "<nil>"
	if resp.Body != nil {
		defer resp.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		respBody = buf.String()
		resp.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))
	}
	s.logger.Printf("Successful response from RoundTripper: %+v\n\nBody: '%s'\n", resp, respBody)
	return resp, nil
}

// Error implements the error interface.
func (err MissingSignerError) Error() string {
	return "No signer was provided. Cannot create client."
}

// Error implements the error interface.
func (err MissingServiceError) Error() string {
	return "No AWS service abbreviation was provided. Cannot create client."
}

// Error implements the error interface.
func (err MissingRegionError) Error() string {
	return "No AWS region was provided. Cannot create client."
}
