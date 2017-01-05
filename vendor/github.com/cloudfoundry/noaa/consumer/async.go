package consumer

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	noaa_errors "github.com/cloudfoundry/noaa/errors"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
)

const (
	DefaultMinRetryDelay = 500 * time.Millisecond
	DefaultMaxRetryDelay = time.Minute
)

// SetMinRetryDelay sets the duration that automatically reconnecting methods
// on c (e.g. Firehose, Stream, TailingLogs) will sleep for after receiving
// an error from the traffic controller.
//
// Successive errors will double the sleep time, up to c's max retry delay,
// set by c.SetMaxRetryDelay.
//
// Defaults to DefaultMinRetryDelay.
func (c *Consumer) SetMinRetryDelay(d time.Duration) {
	atomic.StoreInt64(&c.minRetryDelay, int64(d))
}

// SetMaxRetryDelay sets the maximum duration that automatically reconnecting
// methods on c (e.g. Firehose, Stream, TailingLogs) will sleep for after
// receiving many successive errors from the traffic controller.
//
// Defaults to DefaultMaxRetryDelay.
func (c *Consumer) SetMaxRetryDelay(d time.Duration) {
	atomic.StoreInt64(&c.maxRetryDelay, int64(d))
}

// TailingLogs listens indefinitely for log messages only; other event types
// are dropped.
// Whenever an error is encountered, the error will be sent down the error
// channel and TailingLogs will attempt to reconnect up to 5 times.  After
// five failed reconnection attempts, TailingLogs will give up and close the
// error and LogMessage channels.
//
// If c is closed, the returned channels will both be closed.
//
// Errors must be drained from the returned error channel for it to continue
// retrying; if they are not drained, the connection attempts will hang.
func (c *Consumer) TailingLogs(appGuid, authToken string) (<-chan *events.LogMessage, <-chan error) {
	return c.tailingLogs(appGuid, authToken, true)
}

// TailingLogsWithoutReconnect functions identically to TailingLogs but without
// any reconnect attempts when errors occur.
func (c *Consumer) TailingLogsWithoutReconnect(appGuid string, authToken string) (<-chan *events.LogMessage, <-chan error) {
	return c.tailingLogs(appGuid, authToken, false)
}

// Stream listens indefinitely for all log and event messages.
//
// Messages are presented in the order received from the loggregator server.
// Chronological or other ordering is not guaranteed. It is the responsibility
// of the consumer of these channels to provide any desired sorting mechanism.
//
// Whenever an error is encountered, the error will be sent down the error
// channel and Stream will attempt to reconnect indefinitely.
func (c *Consumer) Stream(appGuid string, authToken string) (outputChan <-chan *events.Envelope, errorChan <-chan error) {
	return c.runStream(appGuid, authToken, true)
}

// StreamWithoutReconnect functions identically to Stream but without any
// reconnect attempts when errors occur.
func (c *Consumer) StreamWithoutReconnect(appGuid string, authToken string) (<-chan *events.Envelope, <-chan error) {
	return c.runStream(appGuid, authToken, false)
}

// Firehose streams all data. All clients with the same subscriptionId will
// receive a proportionate share of the message stream.  Each pool of clients
// will receive the entire stream.
//
// Messages are presented in the order received from the loggregator server.
// Chronological or other ordering is not guaranteed. It is the responsibility
// of the consumer of these channels to provide any desired sorting mechanism.
//
// Whenever an error is encountered, the error will be sent down the error
// channel and Firehose will attempt to reconnect indefinitely.
func (c *Consumer) Firehose(subscriptionId string, authToken string) (<-chan *events.Envelope, <-chan error) {
	return c.firehose(subscriptionId, authToken, true)
}

// FirehoseWithoutReconnect functions identically to Firehose but without any
// reconnect attempts when errors occur.
func (c *Consumer) FirehoseWithoutReconnect(subscriptionId string, authToken string) (<-chan *events.Envelope, <-chan error) {
	return c.firehose(subscriptionId, authToken, false)
}

// SetDebugPrinter sets the websocket connection to write debug information to
// debugPrinter.
func (c *Consumer) SetDebugPrinter(debugPrinter DebugPrinter) {
	c.debugPrinter = debugPrinter
}

// SetOnConnectCallback sets a callback function to be called with the
// websocket connection is established.
func (c *Consumer) SetOnConnectCallback(cb func()) {
	c.callbackLock.Lock()
	defer c.callbackLock.Unlock()
	c.callback = cb
}

// Close terminates all previously opened websocket connections to the traffic
// controller.  It will return an error if there are no open connections, or
// if it has problems closing any connection.
func (c *Consumer) Close() error {
	c.connsLock.Lock()
	defer c.connsLock.Unlock()
	if len(c.conns) == 0 {
		return errors.New("connection does not exist")
	}
	for len(c.conns) > 0 {
		if err := c.conns[0].close(); err != nil {
			return err
		}
		c.conns = c.conns[1:]
	}
	return nil
}

func (c *Consumer) SetIdleTimeout(idleTimeout time.Duration) {
	c.idleTimeout = idleTimeout
}

func (c *Consumer) onConnectCallback() func() {
	c.callbackLock.RLock()
	defer c.callbackLock.RUnlock()
	return c.callback
}

func (c *Consumer) tailingLogs(appGuid, authToken string, retry bool) (<-chan *events.LogMessage, <-chan error) {
	outputs := make(chan *events.LogMessage)
	errors := make(chan error, 1)
	callback := func(env *events.Envelope) {
		if env.GetEventType() == events.Envelope_LogMessage {
			outputs <- env.GetLogMessage()
		}
	}

	conn := c.newConn()
	go func() {
		defer close(errors)
		defer close(outputs)
		c.streamAppDataTo(conn, appGuid, authToken, callback, errors, retry)
	}()
	return outputs, errors
}

func (c *Consumer) runStream(appGuid, authToken string, retry bool) (<-chan *events.Envelope, <-chan error) {
	outputs := make(chan *events.Envelope)
	errors := make(chan error, 1)

	callback := func(env *events.Envelope) {
		outputs <- env
	}

	conn := c.newConn()
	go func() {
		defer close(errors)
		defer close(outputs)
		c.streamAppDataTo(conn, appGuid, authToken, callback, errors, retry)
	}()
	return outputs, errors
}

func (c *Consumer) streamAppDataTo(conn *connection, appGuid, authToken string, callback func(*events.Envelope), errors chan<- error, retry bool) {
	streamPath := fmt.Sprintf("/apps/%s/stream", appGuid)
	if retry {
		c.retryAction(c.listenAction(conn, streamPath, authToken, callback), errors)
		return
	}
	err, _ := c.listenAction(conn, streamPath, authToken, callback)()
	errors <- err
}

func (c *Consumer) firehose(subID, authToken string, retry bool) (<-chan *events.Envelope, <-chan error) {
	outputs := make(chan *events.Envelope)
	errors := make(chan error, 1)
	callback := func(env *events.Envelope) {
		outputs <- env
	}

	streamPath := "/firehose/" + subID
	conn := c.newConn()
	go func() {
		defer close(errors)
		defer close(outputs)
		if retry {
			c.retryAction(c.listenAction(conn, streamPath, authToken, callback), errors)
			return
		}
		err, _ := c.listenAction(conn, streamPath, authToken, callback)()
		errors <- err
	}()
	return outputs, errors
}

func (c *Consumer) listenForMessages(conn *connection, callback func(*events.Envelope)) error {
	if conn.closed() {
		return nil
	}
	ws := conn.websocket()
	for {
		if c.idleTimeout != 0 {
			ws.SetReadDeadline(time.Now().Add(c.idleTimeout))
		}
		_, data, err := ws.ReadMessage()

		// If the connection was closed (i.e. if conn.Close() was called), we
		// will have a non-nil error, but we want to return a nil error.
		if conn.closed() {
			return nil
		}

		if c.isTimeoutErr(err) {
			return noaa_errors.NewRetryError(err)
		}

		if err != nil {
			return err
		}

		envelope := &events.Envelope{}
		err = proto.Unmarshal(data, envelope)
		if err != nil {
			continue
		}

		callback(envelope)
	}
}

func (c *Consumer) listenAction(conn *connection, streamPath, authToken string, callback func(*events.Envelope)) func() (err error, done bool) {
	return func() (error, bool) {
		if conn.closed() {
			return nil, true
		}
		ws, err := c.establishWebsocketConnection(streamPath, authToken)
		if err != nil {
			return err, false
		}
		conn.setWebsocket(ws)
		return c.listenForMessages(conn, callback), false
	}
}

func (c *Consumer) retryAction(action func() (err error, done bool), errors chan<- error) {
	oldConnectCallback := c.onConnectCallback()
	defer c.SetOnConnectCallback(oldConnectCallback)
	nextSleep := atomic.LoadInt64(&c.minRetryDelay)

	c.SetOnConnectCallback(func() {
		atomic.StoreInt64(&nextSleep, atomic.LoadInt64(&c.minRetryDelay))
		if oldConnectCallback != nil {
			oldConnectCallback()
		}
	})

	for {
		err, done := action()
		if done {
			return
		}

		if _, ok := err.(noaa_errors.NonRetryError); ok {
			c.debugPrinter.Print("WEBSOCKET ERROR", fmt.Sprintf("%s. Retrying...", err.Error()))
			errors <- err
			return
		}

		if err != nil {
			c.debugPrinter.Print("WEBSOCKET ERROR", fmt.Sprintf("%s. Retrying...", err.Error()))
			err = noaa_errors.NewRetryError(err)
		}

		errors <- err

		ns := atomic.LoadInt64(&nextSleep)
		time.Sleep(time.Duration(ns))
		ns = atomic.AddInt64(&nextSleep, ns)
		max := atomic.LoadInt64(&c.maxRetryDelay)
		if ns > max {
			atomic.StoreInt64(&nextSleep, max)
		}
	}
}

func (c *Consumer) isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}

	// This is an unfortunate way to validate this,
	// however the error type is `*websocket.netError`
	// which is not exported
	return strings.Contains(err.Error(), "i/o timeout")
}

func (c *Consumer) newConn() *connection {
	conn := &connection{}
	c.connsLock.Lock()
	defer c.connsLock.Unlock()
	c.conns = append(c.conns, conn)
	return conn
}

func (c *Consumer) websocketConn(path, authToken string) (*websocket.Conn, error) {
	if authToken == "" && c.refreshTokens {
		return c.websocketConnNewToken(path)
	}

	URL, err := url.Parse(c.trafficControllerUrl + path)
	if err != nil {
		return nil, noaa_errors.NewNonRetryError(err)
	}

	if URL.Scheme != "wss" && URL.Scheme != "ws" {
		return nil, noaa_errors.NewNonRetryError(fmt.Errorf("Invalid scheme '%s'", URL.Scheme))
	}

	ws, httpErr := c.tryWebsocketConnection(path, authToken)
	if httpErr != nil {
		err = httpErr.error
		if httpErr.statusCode == http.StatusUnauthorized && c.refreshTokens {
			ws, err = c.websocketConnNewToken(path)
		}
	}
	return ws, err
}

func (c *Consumer) websocketConnNewToken(path string) (*websocket.Conn, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}
	ws, httpErr := c.tryWebsocketConnection(path, token)
	if httpErr != nil {
		return nil, httpErr.error
	}
	return ws, nil
}

func (c *Consumer) establishWebsocketConnection(path, authToken string) (*websocket.Conn, error) {
	ws, err := c.websocketConn(path, authToken)
	if err != nil {
		return nil, err
	}

	callback := c.onConnectCallback()
	if err == nil && callback != nil {
		callback()
	}

	return ws, nil
}

func (c *Consumer) tryWebsocketConnection(path, token string) (*websocket.Conn, *httpError) {
	header := http.Header{"Origin": []string{"http://localhost"}, "Authorization": []string{token}}
	URL := c.trafficControllerUrl + path

	c.debugPrinter.Print("WEBSOCKET REQUEST:",
		"GET "+path+" HTTP/1.1\n"+
			"Host: "+c.trafficControllerUrl+"\n"+
			"Upgrade: websocket\nConnection: Upgrade\nSec-WebSocket-Version: 13\nSec-WebSocket-Key: [HIDDEN]\n"+
			headersString(header))

	ws, resp, err := c.dialer.Dial(URL, header)
	if resp != nil {
		c.debugPrinter.Print("WEBSOCKET RESPONSE:",
			resp.Proto+" "+resp.Status+"\n"+
				headersString(resp.Header))
	}

	httpErr := &httpError{}
	if resp != nil {
		if resp.StatusCode == http.StatusUnauthorized {
			bodyData, _ := ioutil.ReadAll(resp.Body)
			err = noaa_errors.NewUnauthorizedError(string(bodyData))
		}
		httpErr.statusCode = resp.StatusCode
	}
	if err != nil {
		errMsg := "Error dialing trafficcontroller server: %s.\n" +
			"Please ask your Cloud Foundry Operator to check the platform configuration (trafficcontroller is %s)."
		httpErr.error = fmt.Errorf(errMsg, err.Error(), c.trafficControllerUrl)
		return nil, httpErr
	}
	return ws, nil
}

func (c *Consumer) proxyDial(network, addr string) (net.Conn, error) {
	targetUrl, err := url.Parse("http://" + addr)
	if err != nil {
		return nil, err
	}

	proxy := c.proxy
	if proxy == nil {
		proxy = http.ProxyFromEnvironment
	}

	proxyUrl, err := proxy(&http.Request{URL: targetUrl})
	if err != nil {
		return nil, err
	}
	if proxyUrl == nil {
		return net.Dial(network, addr)
	}

	connectHeader := make(http.Header)
	if user := proxyUrl.User; user != nil {
		proxyUser := user.Username()
		if proxyPassword, passwordSet := user.Password(); passwordSet {
			credential := base64.StdEncoding.EncodeToString([]byte(proxyUser + ":" + proxyPassword))
			connectHeader.Set("Proxy-Authorization", "Basic "+credential)
		}
	}

	proxyConn, err := net.Dial(network, proxyUrl.Host)
	if err != nil {
		return nil, err
	}

	connectReq := &http.Request{
		Method: "CONNECT",
		URL:    targetUrl,
		Host:   targetUrl.Host,
		Header: connectHeader,
	}
	connectReq.Write(proxyConn)

	connectResp, err := http.ReadResponse(bufio.NewReader(proxyConn), connectReq)
	if err != nil {
		proxyConn.Close()
		return nil, err
	}
	if connectResp.StatusCode != http.StatusOK {
		f := strings.SplitN(connectResp.Status, " ", 2)
		proxyConn.Close()
		return nil, errors.New(f[1])
	}

	return proxyConn, nil
}

func headersString(header http.Header) string {
	var result string
	for name, values := range header {
		result += name + ": " + strings.Join(values, ", ") + "\n"
	}
	return result
}

type connection struct {
	ws       *websocket.Conn
	isClosed bool
	lock     sync.Mutex
}

func (c *connection) websocket() *websocket.Conn {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.ws
}

func (c *connection) setWebsocket(ws *websocket.Conn) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.isClosed {
		return
	}
	c.ws = ws
}

func (c *connection) close() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.isClosed = true
	if c.ws == nil {
		return nil
	}
	err := c.ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Time{})
	if err != nil {
		return err
	}
	return c.ws.Close()
}

func (c *connection) closed() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.isClosed
}
