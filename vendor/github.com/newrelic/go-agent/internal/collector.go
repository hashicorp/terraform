package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/newrelic/go-agent/internal/logger"
)

const (
	procotolVersion = "14"
	userAgentPrefix = "NewRelic-Go-Agent/"

	// Methods used in collector communication.
	cmdRedirect     = "get_redirect_host"
	cmdConnect      = "connect"
	cmdMetrics      = "metric_data"
	cmdCustomEvents = "custom_event_data"
	cmdTxnEvents    = "analytic_event_data"
	cmdErrorEvents  = "error_event_data"
	cmdErrorData    = "error_data"
	cmdTxnTraces    = "transaction_sample_data"
	cmdSlowSQLs     = "sql_trace_data"
)

var (
	// ErrPayloadTooLarge is created in response to receiving a 413 response
	// code.
	ErrPayloadTooLarge = errors.New("payload too large")
	// ErrUnsupportedMedia is created in response to receiving a 415
	// response code.
	ErrUnsupportedMedia = errors.New("unsupported media")
)

// RpmCmd contains fields specific to an individual call made to RPM.
type RpmCmd struct {
	Name      string
	Collector string
	RunID     string
	Data      []byte
}

// RpmControls contains fields which will be the same for all calls made
// by the same application.
type RpmControls struct {
	UseTLS       bool
	License      string
	Client       *http.Client
	Logger       logger.Logger
	AgentVersion string
}

func rpmURL(cmd RpmCmd, cs RpmControls) string {
	var u url.URL

	u.Host = cmd.Collector
	u.Path = "agent_listener/invoke_raw_method"

	if cs.UseTLS {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}

	query := url.Values{}
	query.Set("marshal_format", "json")
	query.Set("protocol_version", procotolVersion)
	query.Set("method", cmd.Name)
	query.Set("license_key", cs.License)

	if len(cmd.RunID) > 0 {
		query.Set("run_id", cmd.RunID)
	}

	u.RawQuery = query.Encode()
	return u.String()
}

type unexpectedStatusCodeErr struct {
	code int
}

func (e unexpectedStatusCodeErr) Error() string {
	return fmt.Sprintf("unexpected HTTP status code: %d", e.code)
}

func collectorRequestInternal(url string, data []byte, cs RpmControls) ([]byte, error) {
	deflated, err := compress(data)
	if nil != err {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(deflated))
	if nil != err {
		return nil, err
	}

	req.Header.Add("Accept-Encoding", "identity, deflate")
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add("User-Agent", userAgentPrefix+cs.AgentVersion)
	req.Header.Add("Content-Encoding", "deflate")

	resp, err := cs.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if 413 == resp.StatusCode {
		return nil, ErrPayloadTooLarge
	}

	if 415 == resp.StatusCode {
		return nil, ErrUnsupportedMedia
	}

	// If the response code is not 200, then the collector may not return
	// valid JSON.
	if 200 != resp.StatusCode {
		return nil, unexpectedStatusCodeErr{code: resp.StatusCode}
	}

	b, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return nil, err
	}
	return parseResponse(b)
}

// CollectorRequest makes a request to New Relic.
func CollectorRequest(cmd RpmCmd, cs RpmControls) ([]byte, error) {
	url := rpmURL(cmd, cs)

	if cs.Logger.DebugEnabled() {
		cs.Logger.Debug("rpm request", map[string]interface{}{
			"command": cmd.Name,
			"url":     url,
			"payload": JSONString(cmd.Data),
		})
	}

	resp, err := collectorRequestInternal(url, cmd.Data, cs)
	if err != nil {
		cs.Logger.Debug("rpm failure", map[string]interface{}{
			"command": cmd.Name,
			"url":     url,
			"error":   err.Error(),
		})
	}

	if cs.Logger.DebugEnabled() {
		cs.Logger.Debug("rpm response", map[string]interface{}{
			"command":  cmd.Name,
			"url":      url,
			"response": JSONString(resp),
		})
	}

	return resp, err
}

type rpmException struct {
	Message   string `json:"message"`
	ErrorType string `json:"error_type"`
}

func (e *rpmException) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorType, e.Message)
}

func hasType(e error, expected string) bool {
	rpmErr, ok := e.(*rpmException)
	if !ok {
		return false
	}
	return rpmErr.ErrorType == expected

}

const (
	forceRestartType   = "NewRelic::Agent::ForceRestartException"
	disconnectType     = "NewRelic::Agent::ForceDisconnectException"
	licenseInvalidType = "NewRelic::Agent::LicenseException"
	runtimeType        = "RuntimeError"
)

// IsRestartException indicates if the error was a restart exception.
func IsRestartException(e error) bool { return hasType(e, forceRestartType) }

// IsLicenseException indicates if the error was an invalid exception.
func IsLicenseException(e error) bool { return hasType(e, licenseInvalidType) }

// IsRuntime indicates if the error was a runtime exception.
func IsRuntime(e error) bool { return hasType(e, runtimeType) }

// IsDisconnect indicates if the error was a disconnect exception.
func IsDisconnect(e error) bool { return hasType(e, disconnectType) }

func parseResponse(b []byte) ([]byte, error) {
	var r struct {
		ReturnValue json.RawMessage `json:"return_value"`
		Exception   *rpmException   `json:"exception"`
	}

	err := json.Unmarshal(b, &r)
	if nil != err {
		return nil, err
	}

	if nil != r.Exception {
		return nil, r.Exception
	}

	return r.ReturnValue, nil
}

// ConnectAttempt tries to connect an application.
func ConnectAttempt(js []byte, redirectHost string, cs RpmControls) (*AppRun, error) {
	call := RpmCmd{
		Name:      cmdRedirect,
		Collector: redirectHost,
		Data:      []byte("[]"),
	}

	out, err := CollectorRequest(call, cs)
	if nil != err {
		// err is intentionally unmodified:  We do not want to change
		// the type of these collector errors.
		return nil, err
	}

	var host string
	err = json.Unmarshal(out, &host)
	if nil != err {
		return nil, fmt.Errorf("unable to parse redirect reply: %v", err)
	}

	call.Collector = host
	call.Data = js
	call.Name = cmdConnect

	rawReply, err := CollectorRequest(call, cs)
	if nil != err {
		// err is intentionally unmodified:  We do not want to change
		// the type of these collector errors.
		return nil, err
	}

	reply := ConnectReplyDefaults()
	err = json.Unmarshal(rawReply, reply)
	if nil != err {
		return nil, fmt.Errorf("unable to parse connect reply: %v", err)
	}
	// Note:  This should never happen.  It would mean the collector
	// response is malformed.  This exists merely as extra defensiveness.
	if "" == reply.RunID {
		return nil, errors.New("connect reply missing agent run id")
	}

	return &AppRun{reply, host}, nil
}
