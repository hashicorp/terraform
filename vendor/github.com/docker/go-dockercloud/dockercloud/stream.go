package dockercloud

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	WRITE_WAIT = 5 * time.Second
	// Time allowed to read the next pong message from the peer.
	PONG_WAIT = 10 * time.Second
	// Send pings to client with this period. Must be less than PONG_WAIT.
	PING_PERIOD = PONG_WAIT / 2
)

func dial() (*websocket.Conn, *http.Response, error) {
	if os.Getenv("DOCKERCLOUD_STREAM_HOST") != "" {
		u, _ := url.Parse(os.Getenv("DOCKERCLOUD_STREAM_HOST"))
		_, port, _ := net.SplitHostPort(u.Host)
		if port == "" {
			u.Host = u.Host + ":443"
		}
		StreamUrl = u.Scheme + "://" + u.Host + "/"
	} else if os.Getenv("DOCKERCLOUD_STREAM_URL") != "" {
		u, _ := url.Parse(os.Getenv("DOCKERCLOUD_STREAM_URL"))
		_, port, _ := net.SplitHostPort(u.Host)
		if port == "" {
			u.Host = u.Host + ":443"
		}
		StreamUrl = u.Scheme + "://" + u.Host + "/"
	}

	Url := ""
	if Namespace != "" {
		Url = StreamUrl + "api/audit/" + auditSubsystemVersion + "/" + Namespace + "/events/"
	} else {
		Url = StreamUrl + "api/audit/" + auditSubsystemVersion + "/events/"
	}

	header := http.Header{}
	header.Add("Authorization", AuthHeader)
	header.Add("User-Agent", customUserAgent)

	var Dialer websocket.Dialer
	return Dialer.Dial(Url, header)
}

func dialHandler(e chan error) (*websocket.Conn, error) {
	if !IsAuthenticated() {
		err := LoadAuth()
		if err != nil {
			e <- err
			return nil, err
		}
	}

	tries := 0
	for {
		ws, resp, err := dial()
		if err != nil {
			tries++
			time.Sleep(3 * time.Second)
			if resp.StatusCode == 401 {
				return nil, HttpError{Status: resp.Status, StatusCode: resp.StatusCode}
			}
			if tries > 3 {
				log.Println("[DIAL ERROR]: " + err.Error())
				e <- err
			}
		} else {
			return ws, nil
		}
	}
}

func messagesHandler(ws *websocket.Conn, ticker *time.Ticker, msg Event, c chan Event, e chan error, e2 chan error) {
	defer func() {
		close(c)
		close(e)
		close(e2)
	}()
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(PONG_WAIT))
		return nil
	})
	for {
		err := ws.ReadJSON(&msg)
		if err != nil {
			e <- err
			e2 <- err
			time.Sleep(4 * time.Second)
		} else {
			if reflect.TypeOf(msg).String() == "dockercloud.Event" {
				c <- msg
			}
		}
	}
}

func Events(c chan Event, e chan error) {
	var msg Event
	ticker := time.NewTicker(PING_PERIOD)
	ws, err := dialHandler(e)
	if err != nil {
		e <- err
		return
	}
	e2 := make(chan error)

	defer func() {
		ws.Close()
	}()
	go messagesHandler(ws, ticker, msg, c, e, e2)

Loop:
	for {
		select {
		case <-ticker.C:
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				ticker.Stop()
				log.Println("Ping Timeout")
				e <- err
				break Loop
			}
		case <-e2:
			ticker.Stop()
			break Loop
		}
	}
}
