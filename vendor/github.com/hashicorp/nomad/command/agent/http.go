package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/ugorji/go/codec"
)

const (
	// ErrInvalidMethod is used if the HTTP method is not supported
	ErrInvalidMethod = "Invalid method"

	// scadaHTTPAddr is the address associated with the
	// HTTPServer. When populating an ACL token for a request,
	// this is checked to switch between the ACLToken and
	// AtlasACLToken
	scadaHTTPAddr = "SCADA"
)

var (
	// jsonHandle and jsonHandlePretty are the codec handles to JSON encode
	// structs. The pretty handle will add indents for easier human consumption.
	jsonHandle       = &codec.JsonHandle{}
	jsonHandlePretty = &codec.JsonHandle{Indent: 4}
)

// HTTPServer is used to wrap an Agent and expose it over an HTTP interface
type HTTPServer struct {
	agent    *Agent
	mux      *http.ServeMux
	listener net.Listener
	logger   *log.Logger
	addr     string
}

// NewHTTPServer starts new HTTP server over the agent
func NewHTTPServer(agent *Agent, config *Config, logOutput io.Writer) (*HTTPServer, error) {
	// Start the listener
	ln, err := config.Listener("tcp", config.Addresses.HTTP, config.Ports.HTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to start HTTP listener: %v", err)
	}

	// Create the mux
	mux := http.NewServeMux()

	// Create the server
	srv := &HTTPServer{
		agent:    agent,
		mux:      mux,
		listener: ln,
		logger:   agent.logger,
		addr:     ln.Addr().String(),
	}
	srv.registerHandlers(config.EnableDebug)

	// Start the server
	go http.Serve(ln, gziphandler.GzipHandler(mux))
	return srv, nil
}

// newScadaHttp creates a new HTTP server wrapping the SCADA
// listener such that HTTP calls can be sent from the brokers.
func newScadaHttp(agent *Agent, list net.Listener) *HTTPServer {
	// Create the mux
	mux := http.NewServeMux()

	// Create the server
	srv := &HTTPServer{
		agent:    agent,
		mux:      mux,
		listener: list,
		logger:   agent.logger,
		addr:     scadaHTTPAddr,
	}
	srv.registerHandlers(false) // Never allow debug for SCADA

	// Start the server
	go http.Serve(list, gziphandler.GzipHandler(mux))
	return srv
}

// Shutdown is used to shutdown the HTTP server
func (s *HTTPServer) Shutdown() {
	if s != nil {
		s.logger.Printf("[DEBUG] http: Shutting down http server")
		s.listener.Close()
	}
}

// registerHandlers is used to attach our handlers to the mux
func (s *HTTPServer) registerHandlers(enableDebug bool) {
	s.mux.HandleFunc("/v1/jobs", s.wrap(s.JobsRequest))
	s.mux.HandleFunc("/v1/job/", s.wrap(s.JobSpecificRequest))

	s.mux.HandleFunc("/v1/nodes", s.wrap(s.NodesRequest))
	s.mux.HandleFunc("/v1/node/", s.wrap(s.NodeSpecificRequest))

	s.mux.HandleFunc("/v1/allocations", s.wrap(s.AllocsRequest))
	s.mux.HandleFunc("/v1/allocation/", s.wrap(s.AllocSpecificRequest))

	s.mux.HandleFunc("/v1/evaluations", s.wrap(s.EvalsRequest))
	s.mux.HandleFunc("/v1/evaluation/", s.wrap(s.EvalSpecificRequest))

	s.mux.HandleFunc("/v1/client/fs/", s.wrap(s.FsRequest))
	s.mux.HandleFunc("/v1/client/stats", s.wrap(s.ClientStatsRequest))
	s.mux.HandleFunc("/v1/client/allocation/", s.wrap(s.ClientAllocRequest))

	s.mux.HandleFunc("/v1/agent/self", s.wrap(s.AgentSelfRequest))
	s.mux.HandleFunc("/v1/agent/join", s.wrap(s.AgentJoinRequest))
	s.mux.HandleFunc("/v1/agent/members", s.wrap(s.AgentMembersRequest))
	s.mux.HandleFunc("/v1/agent/force-leave", s.wrap(s.AgentForceLeaveRequest))
	s.mux.HandleFunc("/v1/agent/servers", s.wrap(s.AgentServersRequest))

	s.mux.HandleFunc("/v1/regions", s.wrap(s.RegionListRequest))

	s.mux.HandleFunc("/v1/status/leader", s.wrap(s.StatusLeaderRequest))
	s.mux.HandleFunc("/v1/status/peers", s.wrap(s.StatusPeersRequest))

	s.mux.HandleFunc("/v1/system/gc", s.wrap(s.GarbageCollectRequest))
	s.mux.HandleFunc("/v1/system/reconcile/summaries", s.wrap(s.ReconcileJobSummaries))

	if enableDebug {
		s.mux.HandleFunc("/debug/pprof/", pprof.Index)
		s.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}
}

// HTTPCodedError is used to provide the HTTP error code
type HTTPCodedError interface {
	error
	Code() int
}

func CodedError(c int, s string) HTTPCodedError {
	return &codedError{s, c}
}

type codedError struct {
	s    string
	code int
}

func (e *codedError) Error() string {
	return e.s
}

func (e *codedError) Code() int {
	return e.code
}

// wrap is used to wrap functions to make them more convenient
func (s *HTTPServer) wrap(handler func(resp http.ResponseWriter, req *http.Request) (interface{}, error)) func(resp http.ResponseWriter, req *http.Request) {
	f := func(resp http.ResponseWriter, req *http.Request) {
		setHeaders(resp, s.agent.config.HTTPAPIResponseHeaders)
		// Invoke the handler
		reqURL := req.URL.String()
		start := time.Now()
		defer func() {
			s.logger.Printf("[DEBUG] http: Request %v (%v)", reqURL, time.Now().Sub(start))
		}()
		obj, err := handler(resp, req)

		// Check for an error
	HAS_ERR:
		if err != nil {
			s.logger.Printf("[ERR] http: Request %v, error: %v", reqURL, err)
			code := 500
			if http, ok := err.(HTTPCodedError); ok {
				code = http.Code()
			}
			resp.WriteHeader(code)
			resp.Write([]byte(err.Error()))
			return
		}

		prettyPrint := false
		if v, ok := req.URL.Query()["pretty"]; ok {
			if len(v) > 0 && (len(v[0]) == 0 || v[0] != "0") {
				prettyPrint = true
			}
		}

		// Write out the JSON object
		if obj != nil {
			var buf bytes.Buffer
			if prettyPrint {
				enc := codec.NewEncoder(&buf, jsonHandlePretty)
				err = enc.Encode(obj)
				if err == nil {
					buf.Write([]byte("\n"))
				}
			} else {
				enc := codec.NewEncoder(&buf, jsonHandle)
				err = enc.Encode(obj)
			}
			if err != nil {
				goto HAS_ERR
			}
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(buf.Bytes())
		}
	}
	return f
}

// decodeBody is used to decode a JSON request body
func decodeBody(req *http.Request, out interface{}) error {
	dec := json.NewDecoder(req.Body)
	return dec.Decode(&out)
}

// setIndex is used to set the index response header
func setIndex(resp http.ResponseWriter, index uint64) {
	resp.Header().Set("X-Nomad-Index", strconv.FormatUint(index, 10))
}

// setKnownLeader is used to set the known leader header
func setKnownLeader(resp http.ResponseWriter, known bool) {
	s := "true"
	if !known {
		s = "false"
	}
	resp.Header().Set("X-Nomad-KnownLeader", s)
}

// setLastContact is used to set the last contact header
func setLastContact(resp http.ResponseWriter, last time.Duration) {
	lastMsec := uint64(last / time.Millisecond)
	resp.Header().Set("X-Nomad-LastContact", strconv.FormatUint(lastMsec, 10))
}

// setMeta is used to set the query response meta data
func setMeta(resp http.ResponseWriter, m *structs.QueryMeta) {
	setIndex(resp, m.Index)
	setLastContact(resp, m.LastContact)
	setKnownLeader(resp, m.KnownLeader)
}

// setHeaders is used to set canonical response header fields
func setHeaders(resp http.ResponseWriter, headers map[string]string) {
	for field, value := range headers {
		resp.Header().Set(http.CanonicalHeaderKey(field), value)
	}
}

// parseWait is used to parse the ?wait and ?index query params
// Returns true on error
func parseWait(resp http.ResponseWriter, req *http.Request, b *structs.QueryOptions) bool {
	query := req.URL.Query()
	if wait := query.Get("wait"); wait != "" {
		dur, err := time.ParseDuration(wait)
		if err != nil {
			resp.WriteHeader(400)
			resp.Write([]byte("Invalid wait time"))
			return true
		}
		b.MaxQueryTime = dur
	}
	if idx := query.Get("index"); idx != "" {
		index, err := strconv.ParseUint(idx, 10, 64)
		if err != nil {
			resp.WriteHeader(400)
			resp.Write([]byte("Invalid index"))
			return true
		}
		b.MinQueryIndex = index
	}
	return false
}

// parseConsistency is used to parse the ?stale query params.
func parseConsistency(req *http.Request, b *structs.QueryOptions) {
	query := req.URL.Query()
	if _, ok := query["stale"]; ok {
		b.AllowStale = true
	}
}

// parsePrefix is used to parse the ?prefix query param
func parsePrefix(req *http.Request, b *structs.QueryOptions) {
	query := req.URL.Query()
	if prefix := query.Get("prefix"); prefix != "" {
		b.Prefix = prefix
	}
}

// parseRegion is used to parse the ?region query param
func (s *HTTPServer) parseRegion(req *http.Request, r *string) {
	if other := req.URL.Query().Get("region"); other != "" {
		*r = other
	} else if *r == "" {
		*r = s.agent.config.Region
	}
}

// parse is a convenience method for endpoints that need to parse multiple flags
func (s *HTTPServer) parse(resp http.ResponseWriter, req *http.Request, r *string, b *structs.QueryOptions) bool {
	s.parseRegion(req, r)
	parseConsistency(req, b)
	parsePrefix(req, b)
	return parseWait(resp, req, b)
}
