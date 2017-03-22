package testutil

// TestServer is a test helper. It uses a fork/exec model to create
// a test Consul server instance in the background and initialize it
// with some data and/or services. The test server can then be used
// to run a unit test, and offers an easy API to tear itself down
// when the test has completed. The only prerequisite is to have a consul
// binary available on the $PATH.
//
// This package does not use Consul's official API client. This is
// because we use TestServer to test the API client, which would
// otherwise cause an import cycle.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/consul/structs"
	"github.com/hashicorp/go-cleanhttp"
)

// TestPerformanceConfig configures the performance parameters.
type TestPerformanceConfig struct {
	RaftMultiplier uint `json:"raft_multiplier,omitempty"`
}

// TestPortConfig configures the various ports used for services
// provided by the Consul server.
type TestPortConfig struct {
	DNS     int `json:"dns,omitempty"`
	HTTP    int `json:"http,omitempty"`
	RPC     int `json:"rpc,omitempty"`
	SerfLan int `json:"serf_lan,omitempty"`
	SerfWan int `json:"serf_wan,omitempty"`
	Server  int `json:"server,omitempty"`
}

// TestAddressConfig contains the bind addresses for various
// components of the Consul server.
type TestAddressConfig struct {
	HTTP string `json:"http,omitempty"`
}

// TestServerConfig is the main server configuration struct.
type TestServerConfig struct {
	NodeName          string                 `json:"node_name"`
	NodeMeta          map[string]string      `json:"node_meta,omitempty"`
	Performance       *TestPerformanceConfig `json:"performance,omitempty"`
	Bootstrap         bool                   `json:"bootstrap,omitempty"`
	Server            bool                   `json:"server,omitempty"`
	DataDir           string                 `json:"data_dir,omitempty"`
	Datacenter        string                 `json:"datacenter,omitempty"`
	DisableCheckpoint bool                   `json:"disable_update_check"`
	LogLevel          string                 `json:"log_level,omitempty"`
	Bind              string                 `json:"bind_addr,omitempty"`
	Addresses         *TestAddressConfig     `json:"addresses,omitempty"`
	Ports             *TestPortConfig        `json:"ports,omitempty"`
	ACLMasterToken    string                 `json:"acl_master_token,omitempty"`
	ACLDatacenter     string                 `json:"acl_datacenter,omitempty"`
	ACLDefaultPolicy  string                 `json:"acl_default_policy,omitempty"`
	Encrypt           string                 `json:"encrypt,omitempty"`
	Stdout, Stderr    io.Writer              `json:"-"`
	Args              []string               `json:"-"`
}

// ServerConfigCallback is a function interface which can be
// passed to NewTestServerConfig to modify the server config.
type ServerConfigCallback func(c *TestServerConfig)

// defaultServerConfig returns a new TestServerConfig struct
// with all of the listen ports incremented by one.
func defaultServerConfig() *TestServerConfig {
	return &TestServerConfig{
		NodeName:          fmt.Sprintf("node%d", randomPort()),
		DisableCheckpoint: true,
		Performance: &TestPerformanceConfig{
			RaftMultiplier: 1,
		},
		Bootstrap: true,
		Server:    true,
		LogLevel:  "debug",
		Bind:      "127.0.0.1",
		Addresses: &TestAddressConfig{},
		Ports: &TestPortConfig{
			DNS:     randomPort(),
			HTTP:    randomPort(),
			RPC:     randomPort(),
			SerfLan: randomPort(),
			SerfWan: randomPort(),
			Server:  randomPort(),
		},
	}
}

// randomPort asks the kernel for a random port to use.
func randomPort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestService is used to serialize a service definition.
type TestService struct {
	ID      string   `json:",omitempty"`
	Name    string   `json:",omitempty"`
	Tags    []string `json:",omitempty"`
	Address string   `json:",omitempty"`
	Port    int      `json:",omitempty"`
}

// TestCheck is used to serialize a check definition.
type TestCheck struct {
	ID        string `json:",omitempty"`
	Name      string `json:",omitempty"`
	ServiceID string `json:",omitempty"`
	TTL       string `json:",omitempty"`
}

// TestingT is an interface wrapper around TestingT
type TestingT interface {
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatal(args ...interface{})
	Skip(args ...interface{})
}

// TestKVResponse is what we use to decode KV data.
type TestKVResponse struct {
	Value string
}

// TestServer is the main server wrapper struct.
type TestServer struct {
	cmd    *exec.Cmd
	Config *TestServerConfig
	t      TestingT

	HTTPAddr string
	LANAddr  string
	WANAddr  string

	HttpClient *http.Client
}

// NewTestServer is an easy helper method to create a new Consul
// test server with the most basic configuration.
func NewTestServer(t TestingT) *TestServer {
	return NewTestServerConfig(t, nil)
}

// NewTestServerConfig creates a new TestServer, and makes a call to
// an optional callback function to modify the configuration.
func NewTestServerConfig(t TestingT, cb ServerConfigCallback) *TestServer {
	if path, err := exec.LookPath("consul"); err != nil || path == "" {
		t.Fatal("consul not found on $PATH - download and install " +
			"consul or skip this test")
	}

	dataDir, err := ioutil.TempDir("", "consul")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	configFile, err := ioutil.TempFile(dataDir, "config")
	if err != nil {
		defer os.RemoveAll(dataDir)
		t.Fatalf("err: %s", err)
	}

	consulConfig := defaultServerConfig()
	consulConfig.DataDir = dataDir

	if cb != nil {
		cb(consulConfig)
	}

	configContent, err := json.Marshal(consulConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := configFile.Write(configContent); err != nil {
		t.Fatalf("err: %s", err)
	}
	configFile.Close()

	stdout := io.Writer(os.Stdout)
	if consulConfig.Stdout != nil {
		stdout = consulConfig.Stdout
	}

	stderr := io.Writer(os.Stderr)
	if consulConfig.Stderr != nil {
		stderr = consulConfig.Stderr
	}

	// Start the server
	args := []string{"agent", "-config-file", configFile.Name()}
	args = append(args, consulConfig.Args...)
	cmd := exec.Command("consul", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("err: %s", err)
	}

	var httpAddr string
	var client *http.Client
	if strings.HasPrefix(consulConfig.Addresses.HTTP, "unix://") {
		httpAddr = consulConfig.Addresses.HTTP
		trans := cleanhttp.DefaultTransport()
		trans.Dial = func(_, _ string) (net.Conn, error) {
			return net.Dial("unix", httpAddr[7:])
		}
		client = &http.Client{
			Transport: trans,
		}
	} else {
		httpAddr = fmt.Sprintf("127.0.0.1:%d", consulConfig.Ports.HTTP)
		client = cleanhttp.DefaultClient()
	}

	server := &TestServer{
		Config: consulConfig,
		cmd:    cmd,
		t:      t,

		HTTPAddr: httpAddr,
		LANAddr:  fmt.Sprintf("127.0.0.1:%d", consulConfig.Ports.SerfLan),
		WANAddr:  fmt.Sprintf("127.0.0.1:%d", consulConfig.Ports.SerfWan),

		HttpClient: client,
	}

	// Wait for the server to be ready
	if consulConfig.Bootstrap {
		server.waitForLeader()
	} else {
		server.waitForAPI()
	}

	return server
}

// Stop stops the test Consul server, and removes the Consul data
// directory once we are done.
func (s *TestServer) Stop() {
	defer os.RemoveAll(s.Config.DataDir)

	if err := s.cmd.Process.Kill(); err != nil {
		s.t.Errorf("err: %s", err)
	}

	// wait for the process to exit to be sure that the data dir can be
	// deleted on all platforms.
	s.cmd.Wait()
}

// waitForAPI waits for only the agent HTTP endpoint to start
// responding. This is an indication that the agent has started,
// but will likely return before a leader is elected.
func (s *TestServer) waitForAPI() {
	WaitForResult(func() (bool, error) {
		resp, err := s.HttpClient.Get(s.url("/v1/agent/self"))
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()
		if err := s.requireOK(resp); err != nil {
			return false, err
		}
		return true, nil
	}, func(err error) {
		defer s.Stop()
		s.t.Fatalf("err: %s", err)
	})
}

// waitForLeader waits for the Consul server's HTTP API to become
// available, and then waits for a known leader and an index of
// 1 or more to be observed to confirm leader election is done.
// It then waits to ensure the anti-entropy sync has completed.
func (s *TestServer) waitForLeader() {
	var index int64
	WaitForResult(func() (bool, error) {
		// Query the API and check the status code.
		url := s.url(fmt.Sprintf("/v1/catalog/nodes?index=%d&wait=2s", index))
		resp, err := s.HttpClient.Get(url)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()
		if err := s.requireOK(resp); err != nil {
			return false, err
		}

		// Ensure we have a leader and a node registration.
		if leader := resp.Header.Get("X-Consul-KnownLeader"); leader != "true" {
			return false, fmt.Errorf("Consul leader status: %#v", leader)
		}
		index, err = strconv.ParseInt(resp.Header.Get("X-Consul-Index"), 10, 64)
		if err != nil {
			return false, fmt.Errorf("Consul index was bad: %v", err)
		}
		if index == 0 {
			return false, fmt.Errorf("Consul index is 0")
		}

		// Watch for the anti-entropy sync to finish.
		var parsed []map[string]interface{}
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&parsed); err != nil {
			return false, err
		}
		if len(parsed) < 1 {
			return false, fmt.Errorf("No nodes")
		}
		taggedAddresses, ok := parsed[0]["TaggedAddresses"].(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("Missing tagged addresses")
		}
		if _, ok := taggedAddresses["lan"]; !ok {
			return false, fmt.Errorf("No lan tagged addresses")
		}
		return true, nil
	}, func(err error) {
		defer s.Stop()
		s.t.Fatalf("err: %s", err)
	})
}

// url is a helper function which takes a relative URL and
// makes it into a proper URL against the local Consul server.
func (s *TestServer) url(path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", s.Config.Ports.HTTP, path)
}

// requireOK checks the HTTP response code and ensures it is acceptable.
func (s *TestServer) requireOK(resp *http.Response) error {
	if resp.StatusCode != 200 {
		return fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}
	return nil
}

// put performs a new HTTP PUT request.
func (s *TestServer) put(path string, body io.Reader) *http.Response {
	req, err := http.NewRequest("PUT", s.url(path), body)
	if err != nil {
		s.t.Fatalf("err: %s", err)
	}
	resp, err := s.HttpClient.Do(req)
	if err != nil {
		s.t.Fatalf("err: %s", err)
	}
	if err := s.requireOK(resp); err != nil {
		defer resp.Body.Close()
		s.t.Fatal(err)
	}
	return resp
}

// get performs a new HTTP GET request.
func (s *TestServer) get(path string) *http.Response {
	resp, err := s.HttpClient.Get(s.url(path))
	if err != nil {
		s.t.Fatalf("err: %s", err)
	}
	if err := s.requireOK(resp); err != nil {
		defer resp.Body.Close()
		s.t.Fatal(err)
	}
	return resp
}

// encodePayload returns a new io.Reader wrapping the encoded contents
// of the payload, suitable for passing directly to a new request.
func (s *TestServer) encodePayload(payload interface{}) io.Reader {
	var encoded bytes.Buffer
	enc := json.NewEncoder(&encoded)
	if err := enc.Encode(payload); err != nil {
		s.t.Fatalf("err: %s", err)
	}
	return &encoded
}

// JoinLAN is used to join nodes within the same datacenter.
func (s *TestServer) JoinLAN(addr string) {
	resp := s.get("/v1/agent/join/" + addr)
	resp.Body.Close()
}

// JoinWAN is used to join remote datacenters together.
func (s *TestServer) JoinWAN(addr string) {
	resp := s.get("/v1/agent/join/" + addr + "?wan=1")
	resp.Body.Close()
}

// SetKV sets an individual key in the K/V store.
func (s *TestServer) SetKV(key string, val []byte) {
	resp := s.put("/v1/kv/"+key, bytes.NewBuffer(val))
	resp.Body.Close()
}

// GetKV retrieves a single key and returns its value
func (s *TestServer) GetKV(key string) []byte {
	resp := s.get("/v1/kv/" + key)
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.t.Fatalf("err: %s", err)
	}

	var result []*TestKVResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		s.t.Fatalf("err: %s", err)
	}
	if len(result) < 1 {
		s.t.Fatalf("key does not exist: %s", key)
	}

	v, err := base64.StdEncoding.DecodeString(result[0].Value)
	if err != nil {
		s.t.Fatalf("err: %s", err)
	}

	return v
}

// PopulateKV fills the Consul KV with data from a generic map.
func (s *TestServer) PopulateKV(data map[string][]byte) {
	for k, v := range data {
		s.SetKV(k, v)
	}
}

// ListKV returns a list of keys present in the KV store. This will list all
// keys under the given prefix recursively and return them as a slice.
func (s *TestServer) ListKV(prefix string) []string {
	resp := s.get("/v1/kv/" + prefix + "?keys")
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.t.Fatalf("err: %s", err)
	}

	var result []string
	if err := json.Unmarshal(raw, &result); err != nil {
		s.t.Fatalf("err: %s", err)
	}
	return result
}

// AddService adds a new service to the Consul instance. It also
// automatically adds a health check with the given status, which
// can be one of "passing", "warning", or "critical".
func (s *TestServer) AddService(name, status string, tags []string) {
	svc := &TestService{
		Name: name,
		Tags: tags,
	}
	payload := s.encodePayload(svc)
	s.put("/v1/agent/service/register", payload)

	chkName := "service:" + name
	chk := &TestCheck{
		Name:      chkName,
		ServiceID: name,
		TTL:       "10m",
	}
	payload = s.encodePayload(chk)
	s.put("/v1/agent/check/register", payload)

	switch status {
	case structs.HealthPassing:
		s.put("/v1/agent/check/pass/"+chkName, nil)
	case structs.HealthWarning:
		s.put("/v1/agent/check/warn/"+chkName, nil)
	case structs.HealthCritical:
		s.put("/v1/agent/check/fail/"+chkName, nil)
	default:
		s.t.Fatalf("Unrecognized status: %s", status)
	}
}

// AddCheck adds a check to the Consul instance. If the serviceID is
// left empty (""), then the check will be associated with the node.
// The check status may be "passing", "warning", or "critical".
func (s *TestServer) AddCheck(name, serviceID, status string) {
	chk := &TestCheck{
		ID:   name,
		Name: name,
		TTL:  "10m",
	}
	if serviceID != "" {
		chk.ServiceID = serviceID
	}

	payload := s.encodePayload(chk)
	s.put("/v1/agent/check/register", payload)

	switch status {
	case structs.HealthPassing:
		s.put("/v1/agent/check/pass/"+name, nil)
	case structs.HealthWarning:
		s.put("/v1/agent/check/warn/"+name, nil)
	case structs.HealthCritical:
		s.put("/v1/agent/check/fail/"+name, nil)
	default:
		s.t.Fatalf("Unrecognized status: %s", status)
	}
}
