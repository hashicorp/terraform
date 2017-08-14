package logging

import (
	"log"
	"net/http"
	"net/http/httputil"
)

type transport struct {
	name      string
	transport http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if IsDebugOrHigher() {
		reqData, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Printf("[DEBUG] "+logReqMsg, t.name, string(reqData))
		} else {
			log.Printf("[ERROR] %s API Request error: %#v", t.name, err)
		}
	}

	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if IsDebugOrHigher() {
		respData, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Printf("[DEBUG] "+logRespMsg, t.name, string(respData))
		} else {
			log.Printf("[ERROR] %s API Response error: %#v", t.name, err)
		}
	}

	return resp, nil
}

func NewTransport(name string, t http.RoundTripper) *transport {
	return &transport{name, t}
}

const logReqMsg = `%s API Request Details:
---[ REQUEST ]---------------------------------------
%s
-----------------------------------------------------`

const logRespMsg = `%s API Response Details:
---[ RESPONSE ]--------------------------------------
%s
-----------------------------------------------------`
