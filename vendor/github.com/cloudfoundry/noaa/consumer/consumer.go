package consumer

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/cloudfoundry/noaa/consumer/internal"

	noaa_errors "github.com/cloudfoundry/noaa/errors"
	"github.com/gorilla/websocket"
)

var (
	// KeepAlive sets the interval between keep-alive messages sent by the client to loggregator.
	KeepAlive = 25 * time.Second

	boundaryRegexp    = regexp.MustCompile("boundary=(.*)")
	ErrNotOK          = errors.New("unknown issue when making HTTP request to Loggregator")
	ErrNotFound       = ErrNotOK // NotFound isn't an accurate description of how this is used; please use ErrNotOK instead
	ErrBadResponse    = errors.New("bad server response")
	ErrBadRequest     = errors.New("bad client request")
	ErrLostConnection = errors.New("remote server terminated connection unexpectedly")
)

//go:generate hel --type DebugPrinter --output mock_debug_printer_test.go

// DebugPrinter is a type which handles printing debug information.
type DebugPrinter interface {
	Print(title, dump string)
}

type nullDebugPrinter struct {
}

func (nullDebugPrinter) Print(title, body string) {
}

// Consumer represents the actions that can be performed against trafficcontroller.
// See sync.go and async.go for trafficcontroller access methods.
type Consumer struct {
	trafficControllerUrl string
	idleTimeout          time.Duration
	callback             func()
	callbackLock         sync.RWMutex
	proxy                func(*http.Request) (*url.URL, error)
	debugPrinter         DebugPrinter
	client               *http.Client
	dialer               websocket.Dialer

	conns     []*connection
	connsLock sync.Mutex

	refreshTokens  bool
	refresherMutex sync.RWMutex
	tokenRefresher TokenRefresher

	minRetryDelay, maxRetryDelay int64
}

// New creates a new consumer to a trafficcontroller.
func New(trafficControllerUrl string, tlsConfig *tls.Config, proxy func(*http.Request) (*url.URL, error)) *Consumer {
	transport := &http.Transport{Proxy: proxy, TLSClientConfig: tlsConfig, TLSHandshakeTimeout: internal.Timeout, DisableKeepAlives: true}
	consumer := &Consumer{
		trafficControllerUrl: trafficControllerUrl,
		proxy:                proxy,
		debugPrinter:         nullDebugPrinter{},
		client: &http.Client{
			Transport: transport,
			Timeout:   internal.Timeout,
		},
		minRetryDelay: int64(DefaultMinRetryDelay),
		maxRetryDelay: int64(DefaultMaxRetryDelay),
	}
	consumer.dialer = websocket.Dialer{HandshakeTimeout: internal.Timeout, NetDial: consumer.proxyDial, TLSClientConfig: tlsConfig}
	return consumer
}

type httpError struct {
	statusCode int
	error      error
}

func checkForErrors(resp *http.Response) *httpError {
	if resp.StatusCode == http.StatusUnauthorized {
		data, _ := ioutil.ReadAll(resp.Body)
		return &httpError{
			statusCode: resp.StatusCode,
			error:      noaa_errors.NewUnauthorizedError(string(data)),
		}
	}

	if resp.StatusCode == http.StatusBadRequest {
		return &httpError{
			statusCode: resp.StatusCode,
			error:      ErrBadRequest,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &httpError{
			statusCode: resp.StatusCode,
			error:      ErrNotOK,
		}
	}
	return nil
}
