package gottyclient

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/creack/goselect"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh/terminal"
)

// GetAuthTokenURL transforms a GoTTY http URL to its AuthToken file URL
func GetAuthTokenURL(httpURL string) (*url.URL, *http.Header, error) {
	header := http.Header{}
	target, err := url.Parse(httpURL)
	if err != nil {
		return nil, nil, err
	}

	target.Path = strings.TrimLeft(target.Path+"auth_token.js", "/")

	if target.User != nil {
		header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(target.User.String())))
		target.User = nil
	}

	return target, &header, nil
}

// GetURLQuery returns url.query
func GetURLQuery(rawurl string) (url.Values, error) {
	target, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	return target.Query(), nil
}

// GetWebsocketURL transforms a GoTTY http URL to its WebSocket URL
func GetWebsocketURL(httpURL string) (*url.URL, *http.Header, error) {
	header := http.Header{}
	target, err := url.Parse(httpURL)
	if err != nil {
		return nil, nil, err
	}

	if target.Scheme == "https" {
		target.Scheme = "wss"
	} else {
		target.Scheme = "ws"
	}

	target.Path = strings.TrimLeft(target.Path+"ws", "/")

	if target.User != nil {
		header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(target.User.String())))
		target.User = nil
	}

	return target, &header, nil
}

type Client struct {
	Dialer          *websocket.Dialer
	Conn            *websocket.Conn
	URL             string
	WriteMutex      *sync.Mutex
	Output          io.Writer
	poison          chan bool
	SkipTLSVerify   bool
	UseProxyFromEnv bool
	Connected       bool
}

type querySingleType struct {
	AuthToken string `json:"AuthToken"`
	Arguments string `json:"Arguments"`
}

func (c *Client) write(data []byte) error {
	c.WriteMutex.Lock()
	defer c.WriteMutex.Unlock()
	return c.Conn.WriteMessage(websocket.TextMessage, data)
}

// GetAuthToken retrieves an Auth Token from dynamic auth_token.js file
func (c *Client) GetAuthToken() (string, error) {
	target, header, err := GetAuthTokenURL(c.URL)
	if err != nil {
		return "", err
	}

	logrus.Debugf("Fetching auth token auth-token: %q", target.String())
	req, err := http.NewRequest("GET", target.String(), nil)
	req.Header = *header
	tr := &http.Transport{}
	if c.SkipTLSVerify {
		conf := &tls.Config{InsecureSkipVerify: true}
		tr.TLSClientConfig = conf
	}
	if c.UseProxyFromEnv {
		tr.Proxy = http.ProxyFromEnvironment
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case 200:
		// Everything is OK
	default:
		return "", fmt.Errorf("unknown status code: %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile("var gotty_auth_token = '(.*)'")
	output := re.FindStringSubmatch(string(body))
	if len(output) == 0 {
		return "", fmt.Errorf("Cannot fetch GoTTY auth-token, please upgrade your GoTTY server.")
	}

	return output[1], nil
}

// Connect tries to dial a websocket server
func (c *Client) Connect() error {
	// Retrieve AuthToken
	authToken, err := c.GetAuthToken()
	if err != nil {
		return err
	}
	logrus.Debugf("Auth-token: %q", authToken)

	// Open WebSocket connection
	target, header, err := GetWebsocketURL(c.URL)
	if err != nil {
		return err
	}
	logrus.Debugf("Connecting to websocket: %q", target.String())
	if c.SkipTLSVerify {
		c.Dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if c.UseProxyFromEnv {
		c.Dialer.Proxy = http.ProxyFromEnvironment
	}
	conn, _, err := c.Dialer.Dial(target.String(), *header)
	if err != nil {
		return err
	}
	c.Conn = conn
	c.Connected = true

	// Pass arguments and auth-token
	query, err := GetURLQuery(c.URL)
	if err != nil {
		return err
	}
	querySingle := querySingleType{
		Arguments: "?" + query.Encode(),
		AuthToken: authToken,
	}
	json, err := json.Marshal(querySingle)
	if err != nil {
		logrus.Errorf("Failed to parse init message %v", err)
		return err
	}
	// Send Json
	logrus.Debugf("Sending arguments and auth-token")
	err = c.write(json)
	if err != nil {
		return err
	}

	go c.pingLoop()

	return nil
}

func (c *Client) pingLoop() {
	for {
		logrus.Debugf("Sending ping")
		c.write([]byte("1"))
		time.Sleep(30 * time.Second)
	}
}

// Close will nicely close the dialer
func (c *Client) Close() {
	c.Conn.Close()
}

// ExitLoop will kill all goroutines launched by c.Loop()
// ExitLoop() -> wait Loop() -> Close()
func (c *Client) ExitLoop() {
	fname := "ExitLoop"
	openPoison(fname, c.poison)
}

// Loop will look indefinitely for new messages
func (c *Client) Loop() error {
	if !c.Connected {
		err := c.Connect()
		if err != nil {
			return err
		}
	}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go c.termsizeLoop(wg)

	wg.Add(1)
	go c.readLoop(wg)

	wg.Add(1)
	go c.writeLoop(wg)

	/* Wait for all of the above goroutines to finish */
	wg.Wait()

	logrus.Debug("Client.Loop() exiting")
	return nil
}

type winsize struct {
	Rows    uint16 `json:"rows"`
	Columns uint16 `json:"columns"`
	// unused
	x uint16
	y uint16
}

type posionReason int

const (
	committedSuicide = iota
	killed
)

func openPoison(fname string, poison chan bool) posionReason {
	logrus.Debug(fname + " suicide")

	/*
	 * The close() may raise panic if multiple goroutines commit suicide at the
	 * same time. Prevent that panic from bubbling up.
	 */
	defer func() {
		if r := recover(); r != nil {
			logrus.Debug("Prevented panic() of simultaneous suicides", r)
		}
	}()

	/* Signal others to die */
	close(poison)
	return committedSuicide
}

func die(fname string, poison chan bool) posionReason {
	logrus.Debug(fname + " died")

	wasOpen := <-poison
	if wasOpen {
		logrus.Error("ERROR: The channel was open when it wasn't suppoed to be")
	}

	return killed
}

func (c *Client) termsizeLoop(wg *sync.WaitGroup) posionReason {

	defer wg.Done()
	fname := "termsizeLoop"

	ch := make(chan os.Signal, 1)
	notifySignalSIGWINCH(ch)
	defer resetSignalSIGWINCH()

	for {
		if b, err := syscallTIOCGWINSZ(); err != nil {
			logrus.Warn(err)
		} else {
			if err = c.write(append([]byte("2"), b...)); err != nil {
				logrus.Warnf("ws.WriteMessage failed: %v", err)
			}
		}
		select {
		case <-c.poison:
			/* Somebody poisoned the well; die */
			return die(fname, c.poison)
		case <-ch:
		}
	}
}

type exposeFd interface {
	Fd() uintptr
}

func (c *Client) writeLoop(wg *sync.WaitGroup) posionReason {

	defer wg.Done()
	fname := "writeLoop"

	buff := make([]byte, 128)
	oldState, err := terminal.MakeRaw(0)
	if err == nil {
		defer terminal.Restore(0, oldState)
	}

	rdfs := &goselect.FDSet{}
	reader := io.Reader(os.Stdin)
	for {
		select {
		case <-c.poison:
			/* Somebody poisoned the well; die */
			return die(fname, c.poison)
		default:
		}

		rdfs.Zero()
		rdfs.Set(reader.(exposeFd).Fd())
		err := goselect.Select(1, rdfs, nil, nil, 50*time.Millisecond)
		if err != nil {
			return openPoison(fname, c.poison)
		}
		if rdfs.IsSet(reader.(exposeFd).Fd()) {
			size, err := reader.Read(buff)

			if err != nil {
				if err == io.EOF {
					// Send EOF to GoTTY

					// Send 'Input' marker, as defined in GoTTY::client_context.go,
					// followed by EOT (a translation of Ctrl-D for terminals)
					err = c.write(append([]byte("0"), byte(4)))

					if err != nil {
						return openPoison(fname, c.poison)
					}
					continue
				} else {
					return openPoison(fname, c.poison)
				}
			}

			if size <= 0 {
				continue
			}

			data := buff[:size]
			err = c.write(append([]byte("0"), data...))
			if err != nil {
				return openPoison(fname, c.poison)
			}
		}
	}
}

func (c *Client) readLoop(wg *sync.WaitGroup) posionReason {

	defer wg.Done()
	fname := "readLoop"

	type MessageNonBlocking struct {
		Data []byte
		Err  error
	}
	msgChan := make(chan MessageNonBlocking)

	for {
		go func() {
			_, data, err := c.Conn.ReadMessage()
			msgChan <- MessageNonBlocking{Data: data, Err: err}
		}()

		select {
		case <-c.poison:
			/* Somebody poisoned the well; die */
			return die(fname, c.poison)
		case msg := <-msgChan:
			if msg.Err != nil {

				if _, ok := msg.Err.(*websocket.CloseError); !ok {
					logrus.Warnf("c.Conn.ReadMessage: %v", msg.Err)
				}
				return openPoison(fname, c.poison)
			}
			if len(msg.Data) == 0 {

				logrus.Warnf("An error has occured")
				return openPoison(fname, c.poison)
			}
			switch msg.Data[0] {
			case '0': // data
				buf, err := base64.StdEncoding.DecodeString(string(msg.Data[1:]))
				if err != nil {
					logrus.Warnf("Invalid base64 content: %q", msg.Data[1:])
					break
				}
				c.Output.Write(buf)
			case '1': // pong
			case '2': // new title
				newTitle := string(msg.Data[1:])
				fmt.Fprintf(c.Output, "\033]0;%s\007", newTitle)
			case '3': // json prefs
				logrus.Debugf("Unhandled protocol message: json pref: %s", string(msg.Data[1:]))
			case '4': // autoreconnect
				logrus.Debugf("Unhandled protocol message: autoreconnect: %s", string(msg.Data))
			default:
				logrus.Warnf("Unhandled protocol message: %s", string(msg.Data))
			}
		}
	}
}

// SetOutput changes the output stream
func (c *Client) SetOutput(w io.Writer) {
	c.Output = w
}

// ParseURL parses an URL which may be incomplete and tries to standardize it
func ParseURL(input string) (string, error) {
	parsed, err := url.Parse(input)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "http", "https":
		// everything is ok
	default:
		return ParseURL(fmt.Sprintf("http://%s", input))
	}
	return parsed.String(), nil
}

// NewClient returns a GoTTY client object
func NewClient(inputURL string) (*Client, error) {
	url, err := ParseURL(inputURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		Dialer:     &websocket.Dialer{},
		URL:        url,
		WriteMutex: &sync.Mutex{},
		Output:     os.Stdout,
		poison:     make(chan bool),
	}, nil
}
