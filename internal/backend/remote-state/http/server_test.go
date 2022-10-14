package http

//go:generate go run github.com/golang/mock/mockgen -package $GOPACKAGE -source $GOFILE -destination mock_$GOFILE

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

const sampleState = `
{
    "version": 4,
    "serial": 0,
    "lineage": "666f9301-7e65-4b19-ae23-71184bb19b03",
    "remote": {
        "type": "http",
        "config": {
            "path": "local-state.tfstate"
        }
    }
}
`

type (
	HttpServerCallback interface {
		StateGET(req *http.Request)
		StatePOST(req *http.Request)
		StateDELETE(req *http.Request)
		StateLOCK(req *http.Request)
		StateUNLOCK(req *http.Request)
	}
	httpServer struct {
		r     *http.ServeMux
		data  map[string]string
		locks map[string]string
		lock  sync.RWMutex

		httpServerCallback HttpServerCallback
	}
	httpServerOpt func(*httpServer)
)

func withHttpServerCallback(callback HttpServerCallback) httpServerOpt {
	return func(s *httpServer) {
		s.httpServerCallback = callback
	}
}

func newHttpServer(opts ...httpServerOpt) *httpServer {
	r := http.NewServeMux()
	s := &httpServer{
		r:     r,
		data:  make(map[string]string),
		locks: make(map[string]string),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.data["sample"] = sampleState
	r.HandleFunc("/state/", s.handleState)
	return s
}

func (h *httpServer) getResource(req *http.Request) string {
	switch pathParts := strings.SplitN(req.URL.Path, string(filepath.Separator), 3); len(pathParts) {
	case 3:
		return pathParts[2]
	default:
		return ""
	}
}

func (h *httpServer) handleState(writer http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		h.handleStateGET(writer, req)
	case "POST":
		h.handleStatePOST(writer, req)
	case "DELETE":
		h.handleStateDELETE(writer, req)
	case "LOCK":
		h.handleStateLOCK(writer, req)
	case "UNLOCK":
		h.handleStateUNLOCK(writer, req)
	}
}

func (h *httpServer) handleStateGET(writer http.ResponseWriter, req *http.Request) {
	if h.httpServerCallback != nil {
		defer h.httpServerCallback.StateGET(req)
	}
	resource := h.getResource(req)

	h.lock.RLock()
	defer h.lock.RUnlock()

	if state, ok := h.data[resource]; ok {
		_, _ = io.WriteString(writer, state)
	} else {
		writer.WriteHeader(http.StatusNotFound)
	}
}

func (h *httpServer) handleStatePOST(writer http.ResponseWriter, req *http.Request) {
	if h.httpServerCallback != nil {
		defer h.httpServerCallback.StatePOST(req)
	}
	defer req.Body.Close()
	resource := h.getResource(req)

	data, err := io.ReadAll(req.Body)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	h.data[resource] = string(data)
	writer.WriteHeader(http.StatusOK)
}

func (h *httpServer) handleStateDELETE(writer http.ResponseWriter, req *http.Request) {
	if h.httpServerCallback != nil {
		defer h.httpServerCallback.StateDELETE(req)
	}
	resource := h.getResource(req)

	h.lock.Lock()
	defer h.lock.Unlock()

	delete(h.data, resource)
	writer.WriteHeader(http.StatusOK)
}

func (h *httpServer) handleStateLOCK(writer http.ResponseWriter, req *http.Request) {
	if h.httpServerCallback != nil {
		defer h.httpServerCallback.StateLOCK(req)
	}
	defer req.Body.Close()
	resource := h.getResource(req)

	data, err := io.ReadAll(req.Body)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	if existingLock, ok := h.locks[resource]; ok {
		writer.WriteHeader(http.StatusLocked)
		_, _ = io.WriteString(writer, existingLock)
	} else {
		h.locks[resource] = string(data)
		_, _ = io.WriteString(writer, existingLock)
	}
}

func (h *httpServer) handleStateUNLOCK(writer http.ResponseWriter, req *http.Request) {
	if h.httpServerCallback != nil {
		defer h.httpServerCallback.StateUNLOCK(req)
	}
	defer req.Body.Close()
	resource := h.getResource(req)

	data, err := io.ReadAll(req.Body)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	var lockInfo map[string]interface{}
	if err = json.Unmarshal(data, &lockInfo); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	if existingLock, ok := h.locks[resource]; ok {
		var existingLockInfo map[string]interface{}
		if err = json.Unmarshal([]byte(existingLock), &existingLockInfo); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		lockID := lockInfo["ID"].(string)
		existingID := existingLockInfo["ID"].(string)
		if lockID != existingID {
			writer.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(writer, existingLock)
		} else {
			delete(h.locks, resource)
			_, _ = io.WriteString(writer, existingLock)
		}
	} else {
		writer.WriteHeader(http.StatusConflict)
	}
}

func (h *httpServer) handler() http.Handler {
	return h.r
}

func NewHttpTestServer(opts ...httpServerOpt) (*httptest.Server, error) {
	clientCAData, err := os.ReadFile("testdata/certs/ca.cert.pem")
	if err != nil {
		return nil, err
	}
	clientCAs := x509.NewCertPool()
	clientCAs.AppendCertsFromPEM(clientCAData)

	cert, err := tls.LoadX509KeyPair("testdata/certs/server.crt", "testdata/certs/server.key")
	if err != nil {
		return nil, err
	}

	h := newHttpServer(opts...)
	s := httptest.NewUnstartedServer(h.handler())
	s.TLS = &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
		Certificates: []tls.Certificate{cert},
	}

	s.StartTLS()
	return s, nil
}

func TestMTLSServer_NoCertFails(t *testing.T) {
	// Ensure that no calls are made to the server - everything is blocked by the tls.RequireAndVerifyClientCert
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCallback := NewMockHttpServerCallback(ctrl)

	// Fire up a test server
	ts, err := NewHttpTestServer(withHttpServerCallback(mockCallback))
	require.NoError(t, err)
	defer ts.Close()

	// Configure the backend to the pre-populated sample state
	url := ts.URL + "/state/sample"
	conf := map[string]cty.Value{
		"address":                cty.StringVal(url),
		"skip_cert_verification": cty.BoolVal(true),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	require.NotNil(t, b, "nil backend")

	// Now get a state manager and check that it fails to refresh the state
	sm, err := b.StateMgr(backend.DefaultStateName)
	require.NoError(t, err)
	err = sm.RefreshState()
	require.Error(t, err)
}

func TestMTLSServer_WithCertPasses(t *testing.T) {
	// Ensure that the expected amount of calls is made to the server
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCallback := NewMockHttpServerCallback(ctrl)

	// Two or three (not testing the caching here) calls to GET
	mockCallback.EXPECT().
		StateGET(gomock.Any()).
		MinTimes(2).
		MaxTimes(3)
	// One call to the POST to write the data
	mockCallback.EXPECT().
		StatePOST(gomock.Any())

	// Fire up a test server
	ts, err := NewHttpTestServer(withHttpServerCallback(mockCallback))
	require.NoError(t, err)
	defer ts.Close()

	// Configure the backend to the pre-populated sample state, and with all the test certs lined up
	url := ts.URL + "/state/sample"
	conf := map[string]cty.Value{
		"address":                   cty.StringVal(url),
		"lock_address":              cty.StringVal(url),
		"unlock_address":            cty.StringVal(url),
		"client_ca_certificate_pem": cty.StringVal("testdata/certs/ca.cert.pem"),
		"client_certificate_pem":    cty.StringVal("testdata/certs/client.crt"),
		"client_private_key_pem":    cty.StringVal("testdata/certs/client.key"),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	require.NotNil(t, b, "nil backend")

	// Now get a state manager, fetch the state, and ensure that the "foo" output is not set
	sm, err := b.StateMgr(backend.DefaultStateName)
	require.NoError(t, err)
	require.NoError(t, sm.RefreshState())
	state := sm.State()
	require.NotNil(t, state, "nil state")
	stateFoo := state.OutputValue(addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance))
	assert.Nil(t, stateFoo)

	// Create a new state that has "foo" set to "bar" and ensure that state is as expected
	state = states.BuildState(func(ss *states.SyncState) {
		ss.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"),
			false)
	})
	stateFoo = state.OutputValue(addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance))
	require.NotNil(t, stateFoo)
	assert.Equal(t, "bar", stateFoo.Value.AsString())

	// Ensure the change hasn't altered the current state manager state by checking "foo" and comparing states
	curState := sm.State()
	curStateFoo := curState.OutputValue(addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance))
	assert.Nil(t, curStateFoo)
	assert.NotEqual(t, state, curState)

	// Write the new state, persist, and refresh
	assert.NoError(t, sm.WriteState(state))
	assert.NoError(t, sm.PersistState(nil))
	assert.NoError(t, sm.RefreshState())

	// Get the state again and verify that is now the same as state and has the "foo" value set to "bar"
	curState = sm.State()
	assert.Equal(t, state, curState)
	curStateFoo = curState.OutputValue(addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance))
	require.NotNil(t, curStateFoo)
	assert.Equal(t, "bar", curStateFoo.Value.AsString())
}

// TestRunServer allows running the server for local debugging; it runs until ctl-c is received
func TestRunServer(t *testing.T) {
	if _, ok := os.LookupEnv("TEST_RUN_SERVER"); !ok {
		t.Skip("TEST_RUN_SERVER not set")
	}
	s, err := NewHttpTestServer()
	require.NoError(t, err)
	defer s.Close()

	t.Log(s.URL)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	// wait until signal
	<-ctx.Done()
}
