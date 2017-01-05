// Package loggregator_consumer provides a simple, channel-based API for clients to communicate with
// loggregator servers.
package loggregator_consumer

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cloudfoundry/loggregatorlib/logmessage"
	noaa_errors "github.com/cloudfoundry/noaa/errors"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
)

var (
	// KeepAlive sets the interval between keep-alive messages sent by the client to loggregator. This may be mutated by package importers.
	KeepAlive      = 25 * time.Second
	boundaryRegexp = regexp.MustCompile("boundary=(.*)")
	ErrNotFound    = errors.New("/recent path not found or has issues")
	ErrBadResponse = errors.New("bad server response")
	ErrBadRequest  = errors.New("bad client request")
)

const (
	handshakeTimeout = 10 * time.Second
)

/* LoggregatorConsumer represents the actions that can be performed against a loggregator server.
 */
type LoggregatorConsumer interface {

	//	Tail listens indefinitely for log messages. It returns two channels; the first is populated
	//	with log messages, while the second contains errors (e.g. from parsing messages). It returns
	//	immediately. Call Close() to terminate the connection when you are finished listening.
	//
	//	Messages are presented in the order received from the loggregator server. Chronological or
	//	other ordering is not guaranteed. It is the responsibility of the consumer of these channels
	//	to provide any desired sorting mechanism.
	Tail(appGuid string, authToken string) (<-chan *logmessage.LogMessage, error)

	//	Recent connects to loggregator via its 'recent' endpoint and returns a slice of recent messages.
	//	It does not guarantee any order of the messages; they are in the order returned by loggregator.
	//
	//	The SortRecent method is provided to sort the data returned by this method.
	Recent(appGuid string, authToken string) ([]*logmessage.LogMessage, error)

	// Close terminates the websocket connection to loggregator.
	Close() error

	// SetOnConnectCallback sets a callback function to be called with the websocket connection is established.
	SetOnConnectCallback(func())

	// SetDebugPrinter enables logging of the websocket handshake
	SetDebugPrinter(DebugPrinter)
}

type DebugPrinter interface {
	Print(title, dump string)
}

type nullDebugPrinter struct {
}

func (nullDebugPrinter) Print(title, body string) {
}

type consumer struct {
	endpoint     string
	tlsConfig    *tls.Config
	ws           *websocket.Conn
	callback     func()
	proxy        func(*http.Request) (*url.URL, error)
	debugPrinter DebugPrinter
}

/* New creates a new consumer to a loggregator endpoint.
 */
func New(endpoint string, tlsConfig *tls.Config, proxy func(*http.Request) (*url.URL, error)) LoggregatorConsumer {
	return &consumer{endpoint: endpoint, tlsConfig: tlsConfig, proxy: proxy, debugPrinter: nullDebugPrinter{}}
}

/* SetDebugPrinter enables logging of the websocket handshake
 */
func (cnsmr *consumer) SetDebugPrinter(debugPrinter DebugPrinter) {
	cnsmr.debugPrinter = debugPrinter
}

/*
Tail listens indefinitely for log messages. It returns two channels; the first is populated
with log messages, while the second contains errors (e.g. from parsing messages). It returns immediately.
Call Close() to terminate the connection when you are finished listening.

Messages are presented in the order received from the loggregator server. Chronological or other ordering
is not guaranteed. It is the responsibility of the consumer of these channels to provide any desired sorting
mechanism.
*/
func (cnsmr *consumer) Tail(appGuid string, authToken string) (<-chan *logmessage.LogMessage, error) {
	incomingChan := make(chan *logmessage.LogMessage)
	var err error

	tailPath := fmt.Sprintf("/tail/?app=%s", appGuid)
	cnsmr.ws, err = cnsmr.establishWebsocketConnection(tailPath, authToken)

	if err == nil {
		go cnsmr.sendKeepAlive(KeepAlive)

		go func() {
			defer close(incomingChan)
			cnsmr.listenForMessages(incomingChan)
		}()
	}

	return incomingChan, err
}

/*
Recent connects to loggregator via its 'recent' http(s) endpoint and returns a slice of recent messages.
If the new http 'recent' endpoint isn't supported (ie you are connecting to an older loggregator server),
we will fallback to the old Websocket 'dump' endpoint.

It does not guarantee any order of the messages; they are in the order returned by loggregator.

The SortRecent method is provided to sort the data returned by this method.
*/
func (cnsmr *consumer) Recent(appGuid string, authToken string) ([]*logmessage.LogMessage, error) {
	messages, err := cnsmr.httpRecent(appGuid, authToken)
	if err != ErrBadRequest {
		return messages, err
	} else {
		return cnsmr.dump(appGuid, authToken)
	}
}

/*
httpRecent connects to loggregator via its 'recent' http(s) endpoint and returns a slice of recent messages.
It does not guarantee any order of the messages; they are in the order returned by loggregator.
*/
func (cnsmr *consumer) httpRecent(appGuid string, authToken string) ([]*logmessage.LogMessage, error) {
	endpointUrl, err := url.ParseRequestURI(cnsmr.endpoint)
	if err != nil {
		return nil, err
	}

	scheme := "https"

	if endpointUrl.Scheme == "ws" {
		scheme = "http"
	}

	recentPath := fmt.Sprintf("%s://%s/recent?app=%s", scheme, endpointUrl.Host, appGuid)
	transport := &http.Transport{Proxy: cnsmr.proxy, TLSClientConfig: cnsmr.tlsConfig, TLSHandshakeTimeout: handshakeTimeout}
	client := &http.Client{Timeout: handshakeTimeout, Transport: transport}

	req, _ := http.NewRequest("GET", recentPath, nil)
	req.Header.Set("Authorization", authToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error dialing loggregator server: %s.\nPlease ask your Cloud Foundry Operator to check the platform configuration (loggregator endpoint is %s).", err.Error(), cnsmr.endpoint))
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		data, _ := ioutil.ReadAll(resp.Body)
		return nil, noaa_errors.NewUnauthorizedError(string(data))
	}

	if resp.StatusCode == http.StatusBadRequest {
		return nil, ErrBadRequest
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrNotFound
	}

	contentType := resp.Header.Get("Content-Type")

	if len(strings.TrimSpace(contentType)) == 0 {
		return nil, ErrBadResponse
	}

	matches := boundaryRegexp.FindStringSubmatch(contentType)

	if len(matches) != 2 || len(strings.TrimSpace(matches[1])) == 0 {
		return nil, ErrBadResponse
	}

	reader := multipart.NewReader(resp.Body, matches[1])

	var buffer bytes.Buffer
	messages := make([]*logmessage.LogMessage, 0, 200)

	for part, loopErr := reader.NextPart(); loopErr == nil; part, loopErr = reader.NextPart() {
		buffer.Reset()

		msg := new(logmessage.LogMessage)
		_, err := buffer.ReadFrom(part)
		if err != nil {
			break
		}
		proto.Unmarshal(buffer.Bytes(), msg)
		messages = append(messages, msg)
	}

	return messages, err
}

/*
dump connects to loggregator via its 'dump' ws(s) endpoint and returns a slice of recent messages. It does not
guarantee any order of the messages; they are in the order returned by loggregator.

The SortRecent method is provided to sort the data returned by this method.
*/
func (cnsmr *consumer) dump(appGuid string, authToken string) ([]*logmessage.LogMessage, error) {
	var err error

	dumpPath := fmt.Sprintf("/dump/?app=%s", appGuid)
	cnsmr.ws, err = cnsmr.establishWebsocketConnection(dumpPath, authToken)

	if err != nil {
		return nil, err
	}

	messages := []*logmessage.LogMessage{}
	messageChan := make(chan *logmessage.LogMessage)

	go func() {
		err = cnsmr.listenForMessages(messageChan)
		close(messageChan)
	}()

drainLoop:
	for {
		select {
		case msg, ok := <-messageChan:
			if !ok {
				break drainLoop
			}

			messages = append(messages, msg)
		}
	}

	return messages, nil
}

/* Close terminates the websocket connection to loggregator.
 */
func (cnsmr *consumer) Close() error {
	if cnsmr.ws == nil {
		return errors.New("connection does not exist")
	}

	return cnsmr.ws.Close()
}

func (cnsmr *consumer) SetOnConnectCallback(cb func()) {
	cnsmr.callback = cb
}

/*
SortRecent sorts a slice of LogMessages by timestamp. The sort is stable, so
messages with the same timestamp are sorted in the order that they are received.

The input slice is sorted; the return value is simply a pointer to the same slice.
*/
func SortRecent(messages []*logmessage.LogMessage) []*logmessage.LogMessage {
	sort.Stable(logMessageSlice(messages))
	return messages
}

type logMessageSlice []*logmessage.LogMessage

func (lms logMessageSlice) Len() int {
	return len(lms)
}

func (lms logMessageSlice) Less(i, j int) bool {
	return *(lms[i]).Timestamp < *(lms[j]).Timestamp
}

func (lms logMessageSlice) Swap(i, j int) {
	lms[i], lms[j] = lms[j], lms[i]
}

func (cnsmr *consumer) sendKeepAlive(interval time.Duration) {
	for {
		err := cnsmr.ws.WriteMessage(websocket.TextMessage, []byte("I'm alive!"))
		if err != nil {
			return
		}
		time.Sleep(interval)
	}
}

func (cnsmr *consumer) listenForMessages(msgChan chan<- *logmessage.LogMessage) error {
	defer cnsmr.ws.Close()

	for {
		var data []byte

		_, data, err := cnsmr.ws.ReadMessage()
		if err != nil {
			return err
		}

		msg, msgErr := logmessage.ParseMessage(data)
		if msgErr != nil {
			continue
		}

		msgChan <- msg.GetLogMessage()
	}
}

func headersString(header http.Header) string {
	var result string
	for name, values := range header {
		result += name + ": " + strings.Join(values, ", ") + "\n"
	}
	return result
}

func (cnsmr *consumer) establishWebsocketConnection(path string, authToken string) (*websocket.Conn, error) {
	header := http.Header{"Origin": []string{"http://localhost"}, "Authorization": []string{authToken}}

	dialer := websocket.Dialer{HandshakeTimeout: handshakeTimeout, NetDial: cnsmr.proxyDial, TLSClientConfig: cnsmr.tlsConfig}

	url := cnsmr.endpoint + path

	cnsmr.debugPrinter.Print("WEBSOCKET REQUEST:",
		"GET "+path+" HTTP/1.1\n"+
			"Host: "+cnsmr.endpoint+"\n"+
			"Upgrade: websocket\nConnection: Upgrade\nSec-WebSocket-Version: 13\nSec-WebSocket-Key: [HIDDEN]\n"+
			headersString(header))

	ws, resp, err := dialer.Dial(url, header)

	if resp != nil {
		cnsmr.debugPrinter.Print("WEBSOCKET RESPONSE:",
			resp.Proto+" "+resp.Status+"\n"+
				headersString(resp.Header))
	}

	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		bodyData, _ := ioutil.ReadAll(resp.Body)
		err = noaa_errors.NewUnauthorizedError(string(bodyData))
		return ws, err
	}

	if err == nil && cnsmr.callback != nil {
		cnsmr.callback()
	}

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error dialing loggregator server: %s.\nPlease ask your Cloud Foundry Operator to check the platform configuration (loggregator endpoint is %s).", err.Error(), cnsmr.endpoint))
	}

	return ws, err
}

func (cnsmr *consumer) proxyDial(network, addr string) (net.Conn, error) {
	targetUrl, err := url.Parse("http://" + addr)
	if err != nil {
		return nil, err
	}

	proxy := cnsmr.proxy
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

	proxyConn, err := net.Dial(network, proxyUrl.Host)
	if err != nil {
		return nil, err
	}

	connectReq := &http.Request{
		Method: "CONNECT",
		URL:    targetUrl,
		Host:   targetUrl.Host,
		Header: make(http.Header),
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
