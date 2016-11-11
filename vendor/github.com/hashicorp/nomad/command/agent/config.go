package agent

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	client "github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad"
	"github.com/hashicorp/nomad/nomad/structs/config"
)

// Config is the configuration for the Nomad agent.
type Config struct {
	// Region is the region this agent is in. Defaults to global.
	Region string `mapstructure:"region"`

	// Datacenter is the datacenter this agent is in. Defaults to dc1
	Datacenter string `mapstructure:"datacenter"`

	// NodeName is the name we register as. Defaults to hostname.
	NodeName string `mapstructure:"name"`

	// DataDir is the directory to store our state in
	DataDir string `mapstructure:"data_dir"`

	// LogLevel is the level of the logs to putout
	LogLevel string `mapstructure:"log_level"`

	// BindAddr is the address on which all of nomad's services will
	// be bound. If not specified, this defaults to 127.0.0.1.
	BindAddr string `mapstructure:"bind_addr"`

	// EnableDebug is used to enable debugging HTTP endpoints
	EnableDebug bool `mapstructure:"enable_debug"`

	// Ports is used to control the network ports we bind to.
	Ports *Ports `mapstructure:"ports"`

	// Addresses is used to override the network addresses we bind to.
	Addresses *Addresses `mapstructure:"addresses"`

	// AdvertiseAddrs is used to control the addresses we advertise.
	AdvertiseAddrs *AdvertiseAddrs `mapstructure:"advertise"`

	// Client has our client related settings
	Client *ClientConfig `mapstructure:"client"`

	// Server has our server related settings
	Server *ServerConfig `mapstructure:"server"`

	// Telemetry is used to configure sending telemetry
	Telemetry *Telemetry `mapstructure:"telemetry"`

	// LeaveOnInt is used to gracefully leave on the interrupt signal
	LeaveOnInt bool `mapstructure:"leave_on_interrupt"`

	// LeaveOnTerm is used to gracefully leave on the terminate signal
	LeaveOnTerm bool `mapstructure:"leave_on_terminate"`

	// EnableSyslog is used to enable sending logs to syslog
	EnableSyslog bool `mapstructure:"enable_syslog"`

	// SyslogFacility is used to control the syslog facility used.
	SyslogFacility string `mapstructure:"syslog_facility"`

	// DisableUpdateCheck is used to disable the periodic update
	// and security bulletin checking.
	DisableUpdateCheck bool `mapstructure:"disable_update_check"`

	// DisableAnonymousSignature is used to disable setting the
	// anonymous signature when doing the update check and looking
	// for security bulletins
	DisableAnonymousSignature bool `mapstructure:"disable_anonymous_signature"`

	// AtlasConfig is used to configure Atlas
	Atlas *AtlasConfig `mapstructure:"atlas"`

	// Consul contains the configuration for the Consul Agent and
	// parameters necessary to register services, their checks, and
	// discover the current Nomad servers.
	Consul *config.ConsulConfig `mapstructure:"consul"`

	// NomadConfig is used to override the default config.
	// This is largly used for testing purposes.
	NomadConfig *nomad.Config `mapstructure:"-" json:"-"`

	// ClientConfig is used to override the default config.
	// This is largly used for testing purposes.
	ClientConfig *client.Config `mapstructure:"-" json:"-"`

	// DevMode is set by the -dev CLI flag.
	DevMode bool `mapstructure:"-"`

	// Version information is set at compilation time
	Revision          string
	Version           string
	VersionPrerelease string

	// List of config files that have been loaded (in order)
	Files []string `mapstructure:"-"`

	// HTTPAPIResponseHeaders allows users to configure the Nomad http agent to
	// set arbritrary headers on API responses
	HTTPAPIResponseHeaders map[string]string `mapstructure:"http_api_response_headers"`
}

// AtlasConfig is used to enable an parameterize the Atlas integration
type AtlasConfig struct {
	// Infrastructure is the name of the infrastructure
	// we belong to. e.g. hashicorp/stage
	Infrastructure string `mapstructure:"infrastructure"`

	// Token is our authentication token from Atlas
	Token string `mapstructure:"token" json:"-"`

	// Join controls if Atlas will attempt to auto-join the node
	// to it's cluster. Requires Atlas integration.
	Join bool `mapstructure:"join"`

	// Endpoint is the SCADA endpoint used for Atlas integration. If
	// empty, the defaults from the provider are used.
	Endpoint string `mapstructure:"endpoint"`
}

// ClientConfig is configuration specific to the client mode
type ClientConfig struct {
	// Enabled controls if we are a client
	Enabled bool `mapstructure:"enabled"`

	// StateDir is the state directory
	StateDir string `mapstructure:"state_dir"`

	// AllocDir is the directory for storing allocation data
	AllocDir string `mapstructure:"alloc_dir"`

	// Servers is a list of known server addresses. These are as "host:port"
	Servers []string `mapstructure:"servers"`

	// NodeClass is used to group the node by class
	NodeClass string `mapstructure:"node_class"`

	// Options is used for configuration of nomad internals,
	// like fingerprinters and drivers. The format is:
	//
	//  namespace.option = value
	Options map[string]string `mapstructure:"options"`

	// Metadata associated with the node
	Meta map[string]string `mapstructure:"meta"`

	// A mapping of directories on the host OS to attempt to embed inside each
	// task's chroot.
	ChrootEnv map[string]string `mapstructure:"chroot_env"`

	// Interface to use for network fingerprinting
	NetworkInterface string `mapstructure:"network_interface"`

	// The network link speed to use if it can not be determined dynamically.
	NetworkSpeed int `mapstructure:"network_speed"`

	// MaxKillTimeout allows capping the user-specifiable KillTimeout.
	MaxKillTimeout string `mapstructure:"max_kill_timeout"`

	// ClientMaxPort is the upper range of the ports that the client uses for
	// communicating with plugin subsystems
	ClientMaxPort int `mapstructure:"client_max_port"`

	// ClientMinPort is the lower range of the ports that the client uses for
	// communicating with plugin subsystems
	ClientMinPort int `mapstructure:"client_min_port"`

	// Reserved is used to reserve resources from being used by Nomad. This can
	// be used to target a certain utilization or to prevent Nomad from using a
	// particular set of ports.
	Reserved *Resources `mapstructure:"reserved"`
}

// ServerConfig is configuration specific to the server mode
type ServerConfig struct {
	// Enabled controls if we are a server
	Enabled bool `mapstructure:"enabled"`

	// BootstrapExpect tries to automatically bootstrap the Consul cluster,
	// by withholding peers until enough servers join.
	BootstrapExpect int `mapstructure:"bootstrap_expect"`

	// DataDir is the directory to store our state in
	DataDir string `mapstructure:"data_dir"`

	// ProtocolVersion is the protocol version to speak. This must be between
	// ProtocolVersionMin and ProtocolVersionMax.
	ProtocolVersion int `mapstructure:"protocol_version"`

	// NumSchedulers is the number of scheduler thread that are run.
	// This can be as many as one per core, or zero to disable this server
	// from doing any scheduling work.
	NumSchedulers int `mapstructure:"num_schedulers"`

	// EnabledSchedulers controls the set of sub-schedulers that are
	// enabled for this server to handle. This will restrict the evaluations
	// that the workers dequeue for processing.
	EnabledSchedulers []string `mapstructure:"enabled_schedulers"`

	// NodeGCThreshold controls how "old" a node must be to be collected by GC.
	NodeGCThreshold string `mapstructure:"node_gc_threshold"`

	// HeartbeatGrace is the grace period beyond the TTL to account for network,
	// processing delays and clock skew before marking a node as "down".
	HeartbeatGrace string `mapstructure:"heartbeat_grace"`

	// StartJoin is a list of addresses to attempt to join when the
	// agent starts. If Serf is unable to communicate with any of these
	// addresses, then the agent will error and exit.
	StartJoin []string `mapstructure:"start_join"`

	// RetryJoin is a list of addresses to join with retry enabled.
	RetryJoin []string `mapstructure:"retry_join"`

	// RetryMaxAttempts specifies the maximum number of times to retry joining a
	// host on startup. This is useful for cases where we know the node will be
	// online eventually.
	RetryMaxAttempts int `mapstructure:"retry_max"`

	// RetryInterval specifies the amount of time to wait in between join
	// attempts on agent start. The minimum allowed value is 1 second and
	// the default is 30s.
	RetryInterval string        `mapstructure:"retry_interval"`
	retryInterval time.Duration `mapstructure:"-"`

	// RejoinAfterLeave controls our interaction with the cluster after leave.
	// When set to false (default), a leave causes Consul to not rejoin
	// the cluster until an explicit join is received. If this is set to
	// true, we ignore the leave, and rejoin the cluster on start.
	RejoinAfterLeave bool `mapstructure:"rejoin_after_leave"`
}

// Telemetry is the telemetry configuration for the server
type Telemetry struct {
	StatsiteAddr             string        `mapstructure:"statsite_address"`
	StatsdAddr               string        `mapstructure:"statsd_address"`
	DisableHostname          bool          `mapstructure:"disable_hostname"`
	CollectionInterval       string        `mapstructure:"collection_interval"`
	collectionInterval       time.Duration `mapstructure:"-"`
	PublishAllocationMetrics bool          `mapstructure:"publish_allocation_metrics"`
	PublishNodeMetrics       bool          `mapstructure:"publish_node_metrics"`

	// Circonus: see https://github.com/circonus-labs/circonus-gometrics
	// for more details on the various configuration options.
	// Valid configuration combinations:
	//    - CirconusAPIToken
	//      metric management enabled (search for existing check or create a new one)
	//    - CirconusSubmissionUrl
	//      metric management disabled (use check with specified submission_url,
	//      broker must be using a public SSL certificate)
	//    - CirconusAPIToken + CirconusCheckSubmissionURL
	//      metric management enabled (use check with specified submission_url)
	//    - CirconusAPIToken + CirconusCheckID
	//      metric management enabled (use check with specified id)

	// CirconusAPIToken is a valid API Token used to create/manage check. If provided,
	// metric management is enabled.
	// Default: none
	CirconusAPIToken string `mapstructure:"circonus_api_token"`
	// CirconusAPIApp is an app name associated with API token.
	// Default: "consul"
	CirconusAPIApp string `mapstructure:"circonus_api_app"`
	// CirconusAPIURL is the base URL to use for contacting the Circonus API.
	// Default: "https://api.circonus.com/v2"
	CirconusAPIURL string `mapstructure:"circonus_api_url"`
	// CirconusSubmissionInterval is the interval at which metrics are submitted to Circonus.
	// Default: 10s
	CirconusSubmissionInterval string `mapstructure:"circonus_submission_interval"`
	// CirconusCheckSubmissionURL is the check.config.submission_url field from a
	// previously created HTTPTRAP check.
	// Default: none
	CirconusCheckSubmissionURL string `mapstructure:"circonus_submission_url"`
	// CirconusCheckID is the check id (not check bundle id) from a previously created
	// HTTPTRAP check. The numeric portion of the check._cid field.
	// Default: none
	CirconusCheckID string `mapstructure:"circonus_check_id"`
	// CirconusCheckForceMetricActivation will force enabling metrics, as they are encountered,
	// if the metric already exists and is NOT active. If check management is enabled, the default
	// behavior is to add new metrics as they are encoutered. If the metric already exists in the
	// check, it will *NOT* be activated. This setting overrides that behavior.
	// Default: "false"
	CirconusCheckForceMetricActivation string `mapstructure:"circonus_check_force_metric_activation"`
	// CirconusCheckInstanceID serves to uniquely identify the metrics comming from this "instance".
	// It can be used to maintain metric continuity with transient or ephemeral instances as
	// they move around within an infrastructure.
	// Default: hostname:app
	CirconusCheckInstanceID string `mapstructure:"circonus_check_instance_id"`
	// CirconusCheckSearchTag is a special tag which, when coupled with the instance id, helps to
	// narrow down the search results when neither a Submission URL or Check ID is provided.
	// Default: service:app (e.g. service:consul)
	CirconusCheckSearchTag string `mapstructure:"circonus_check_search_tag"`
	// CirconusBrokerID is an explicit broker to use when creating a new check. The numeric portion
	// of broker._cid. If metric management is enabled and neither a Submission URL nor Check ID
	// is provided, an attempt will be made to search for an existing check using Instance ID and
	// Search Tag. If one is not found, a new HTTPTRAP check will be created.
	// Default: use Select Tag if provided, otherwise, a random Enterprise Broker associated
	// with the specified API token or the default Circonus Broker.
	// Default: none
	CirconusBrokerID string `mapstructure:"circonus_broker_id"`
	// CirconusBrokerSelectTag is a special tag which will be used to select a broker when
	// a Broker ID is not provided. The best use of this is to as a hint for which broker
	// should be used based on *where* this particular instance is running.
	// (e.g. a specific geo location or datacenter, dc:sfo)
	// Default: none
	CirconusBrokerSelectTag string `mapstructure:"circonus_broker_select_tag"`
}

// Ports is used to encapsulate the various ports we bind to for network
// services. If any are not specified then the defaults are used instead.
type Ports struct {
	HTTP int `mapstructure:"http"`
	RPC  int `mapstructure:"rpc"`
	Serf int `mapstructure:"serf"`
}

// Addresses encapsulates all of the addresses we bind to for various
// network services. Everything is optional and defaults to BindAddr.
type Addresses struct {
	HTTP string `mapstructure:"http"`
	RPC  string `mapstructure:"rpc"`
	Serf string `mapstructure:"serf"`
}

// AdvertiseAddrs is used to control the addresses we advertise out for
// different network services. Not all network services support an
// advertise address. All are optional and default to BindAddr.
type AdvertiseAddrs struct {
	HTTP string `mapstructure:"http"`
	RPC  string `mapstructure:"rpc"`
	Serf string `mapstructure:"serf"`
}

type Resources struct {
	CPU                 int    `mapstructure:"cpu"`
	MemoryMB            int    `mapstructure:"memory"`
	DiskMB              int    `mapstructure:"disk"`
	IOPS                int    `mapstructure:"iops"`
	ReservedPorts       string `mapstructure:"reserved_ports"`
	ParsedReservedPorts []int  `mapstructure:"-"`
}

// ParseReserved expands the ReservedPorts string into a slice of port numbers.
// The supported syntax is comma seperated integers or ranges seperated by
// hyphens. For example, "80,120-150,160"
func (r *Resources) ParseReserved() error {
	parts := strings.Split(r.ReservedPorts, ",")

	// Hot path the empty case
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}

	ports := make(map[int]struct{})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		rangeParts := strings.Split(part, "-")
		l := len(rangeParts)
		switch l {
		case 1:
			if val := rangeParts[0]; val == "" {
				return fmt.Errorf("can't specify empty port")
			} else {
				port, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				ports[port] = struct{}{}
			}
		case 2:
			// We are parsing a range
			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return err
			}

			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return err
			}

			if end < start {
				return fmt.Errorf("invalid range: starting value (%v) less than ending (%v) value", end, start)
			}

			for i := start; i <= end; i++ {
				ports[i] = struct{}{}
			}
		default:
			return fmt.Errorf("can only parse single port numbers or port ranges (ex. 80,100-120,150)")
		}
	}

	for port := range ports {
		r.ParsedReservedPorts = append(r.ParsedReservedPorts, port)
	}

	sort.Ints(r.ParsedReservedPorts)
	return nil
}

// DevConfig is a Config that is used for dev mode of Nomad.
func DevConfig() *Config {
	conf := DefaultConfig()
	conf.LogLevel = "DEBUG"
	conf.Client.Enabled = true
	conf.Server.Enabled = true
	conf.DevMode = true
	conf.EnableDebug = true
	conf.DisableAnonymousSignature = true
	conf.Consul.AutoAdvertise = true
	if runtime.GOOS == "darwin" {
		conf.Client.NetworkInterface = "lo0"
	} else if runtime.GOOS == "linux" {
		conf.Client.NetworkInterface = "lo"
	}
	conf.Client.Options = map[string]string{
		"driver.raw_exec.enable": "true",
	}

	return conf
}

// DefaultConfig is a the baseline configuration for Nomad
func DefaultConfig() *Config {
	return &Config{
		LogLevel:   "INFO",
		Region:     "global",
		Datacenter: "dc1",
		BindAddr:   "127.0.0.1",
		Ports: &Ports{
			HTTP: 4646,
			RPC:  4647,
			Serf: 4648,
		},
		Addresses:      &Addresses{},
		AdvertiseAddrs: &AdvertiseAddrs{},
		Atlas:          &AtlasConfig{},
		Consul:         config.DefaultConsulConfig(),
		Client: &ClientConfig{
			Enabled:        false,
			NetworkSpeed:   100,
			MaxKillTimeout: "30s",
			ClientMinPort:  14000,
			ClientMaxPort:  14512,
			Reserved:       &Resources{},
		},
		Server: &ServerConfig{
			Enabled:          false,
			StartJoin:        []string{},
			RetryJoin:        []string{},
			RetryInterval:    "30s",
			RetryMaxAttempts: 0,
		},
		SyslogFacility: "LOCAL0",
		Telemetry: &Telemetry{
			CollectionInterval: "1s",
			collectionInterval: 1 * time.Second,
		},
	}
}

// Listener can be used to get a new listener using a custom bind address.
// If the bind provided address is empty, the BindAddr is used instead.
func (c *Config) Listener(proto, addr string, port int) (net.Listener, error) {
	if addr == "" {
		addr = c.BindAddr
	}

	// Do our own range check to avoid bugs in package net.
	//
	//   golang.org/issue/11715
	//   golang.org/issue/13447
	//
	// Both of the above bugs were fixed by golang.org/cl/12447 which will be
	// included in Go 1.6. The error returned below is the same as what Go 1.6
	// will return.
	if 0 > port || port > 65535 {
		return nil, &net.OpError{
			Op:  "listen",
			Net: proto,
			Err: &net.AddrError{Err: "invalid port", Addr: fmt.Sprint(port)},
		}
	}
	return net.Listen(proto, fmt.Sprintf("%s:%d", addr, port))
}

// Merge merges two configurations.
func (c *Config) Merge(b *Config) *Config {
	result := *c

	if b.Region != "" {
		result.Region = b.Region
	}
	if b.Datacenter != "" {
		result.Datacenter = b.Datacenter
	}
	if b.NodeName != "" {
		result.NodeName = b.NodeName
	}
	if b.DataDir != "" {
		result.DataDir = b.DataDir
	}
	if b.LogLevel != "" {
		result.LogLevel = b.LogLevel
	}
	if b.BindAddr != "" {
		result.BindAddr = b.BindAddr
	}
	if b.EnableDebug {
		result.EnableDebug = true
	}
	if b.LeaveOnInt {
		result.LeaveOnInt = true
	}
	if b.LeaveOnTerm {
		result.LeaveOnTerm = true
	}
	if b.EnableSyslog {
		result.EnableSyslog = true
	}
	if b.SyslogFacility != "" {
		result.SyslogFacility = b.SyslogFacility
	}
	if b.DisableUpdateCheck {
		result.DisableUpdateCheck = true
	}
	if b.DisableAnonymousSignature {
		result.DisableAnonymousSignature = true
	}

	// Apply the telemetry config
	if result.Telemetry == nil && b.Telemetry != nil {
		telemetry := *b.Telemetry
		result.Telemetry = &telemetry
	} else if b.Telemetry != nil {
		result.Telemetry = result.Telemetry.Merge(b.Telemetry)
	}

	// Apply the client config
	if result.Client == nil && b.Client != nil {
		client := *b.Client
		result.Client = &client
	} else if b.Client != nil {
		result.Client = result.Client.Merge(b.Client)
	}

	// Apply the server config
	if result.Server == nil && b.Server != nil {
		server := *b.Server
		result.Server = &server
	} else if b.Server != nil {
		result.Server = result.Server.Merge(b.Server)
	}

	// Apply the ports config
	if result.Ports == nil && b.Ports != nil {
		ports := *b.Ports
		result.Ports = &ports
	} else if b.Ports != nil {
		result.Ports = result.Ports.Merge(b.Ports)
	}

	// Apply the address config
	if result.Addresses == nil && b.Addresses != nil {
		addrs := *b.Addresses
		result.Addresses = &addrs
	} else if b.Addresses != nil {
		result.Addresses = result.Addresses.Merge(b.Addresses)
	}

	// Apply the advertise addrs config
	if result.AdvertiseAddrs == nil && b.AdvertiseAddrs != nil {
		advertise := *b.AdvertiseAddrs
		result.AdvertiseAddrs = &advertise
	} else if b.AdvertiseAddrs != nil {
		result.AdvertiseAddrs = result.AdvertiseAddrs.Merge(b.AdvertiseAddrs)
	}

	// Apply the Atlas configuration
	if result.Atlas == nil && b.Atlas != nil {
		atlasConfig := *b.Atlas
		result.Atlas = &atlasConfig
	} else if b.Atlas != nil {
		result.Atlas = result.Atlas.Merge(b.Atlas)
	}

	// Apply the Consul Configuration
	if result.Consul == nil && b.Consul != nil {
		consulConfig := *b.Consul
		result.Consul = &consulConfig
	} else if b.Consul != nil {
		result.Consul = result.Consul.Merge(b.Consul)
	}

	// Merge config files lists
	result.Files = append(result.Files, b.Files...)

	// Add the http API response header map values
	if result.HTTPAPIResponseHeaders == nil {
		result.HTTPAPIResponseHeaders = make(map[string]string)
	}
	for k, v := range b.HTTPAPIResponseHeaders {
		result.HTTPAPIResponseHeaders[k] = v
	}

	return &result
}

// Merge is used to merge two server configs together
func (a *ServerConfig) Merge(b *ServerConfig) *ServerConfig {
	result := *a

	if b.Enabled {
		result.Enabled = true
	}
	if b.BootstrapExpect > 0 {
		result.BootstrapExpect = b.BootstrapExpect
	}
	if b.DataDir != "" {
		result.DataDir = b.DataDir
	}
	if b.ProtocolVersion != 0 {
		result.ProtocolVersion = b.ProtocolVersion
	}
	if b.NumSchedulers != 0 {
		result.NumSchedulers = b.NumSchedulers
	}
	if b.NodeGCThreshold != "" {
		result.NodeGCThreshold = b.NodeGCThreshold
	}
	if b.HeartbeatGrace != "" {
		result.HeartbeatGrace = b.HeartbeatGrace
	}
	if b.RetryMaxAttempts != 0 {
		result.RetryMaxAttempts = b.RetryMaxAttempts
	}
	if b.RetryInterval != "" {
		result.RetryInterval = b.RetryInterval
		result.retryInterval = b.retryInterval
	}
	if b.RejoinAfterLeave {
		result.RejoinAfterLeave = true
	}

	// Add the schedulers
	result.EnabledSchedulers = append(result.EnabledSchedulers, b.EnabledSchedulers...)

	// Copy the start join addresses
	result.StartJoin = make([]string, 0, len(a.StartJoin)+len(b.StartJoin))
	result.StartJoin = append(result.StartJoin, a.StartJoin...)
	result.StartJoin = append(result.StartJoin, b.StartJoin...)

	// Copy the retry join addresses
	result.RetryJoin = make([]string, 0, len(a.RetryJoin)+len(b.RetryJoin))
	result.RetryJoin = append(result.RetryJoin, a.RetryJoin...)
	result.RetryJoin = append(result.RetryJoin, b.RetryJoin...)

	return &result
}

// Merge is used to merge two client configs together
func (a *ClientConfig) Merge(b *ClientConfig) *ClientConfig {
	result := *a

	if b.Enabled {
		result.Enabled = true
	}
	if b.StateDir != "" {
		result.StateDir = b.StateDir
	}
	if b.AllocDir != "" {
		result.AllocDir = b.AllocDir
	}
	if b.NodeClass != "" {
		result.NodeClass = b.NodeClass
	}
	if b.NetworkInterface != "" {
		result.NetworkInterface = b.NetworkInterface
	}
	if b.NetworkSpeed != 0 {
		result.NetworkSpeed = b.NetworkSpeed
	}
	if b.MaxKillTimeout != "" {
		result.MaxKillTimeout = b.MaxKillTimeout
	}
	if b.ClientMaxPort != 0 {
		result.ClientMaxPort = b.ClientMaxPort
	}
	if b.ClientMinPort != 0 {
		result.ClientMinPort = b.ClientMinPort
	}
	if b.Reserved != nil {
		result.Reserved = result.Reserved.Merge(b.Reserved)
	}

	// Add the servers
	result.Servers = append(result.Servers, b.Servers...)

	// Add the options map values
	if result.Options == nil {
		result.Options = make(map[string]string)
	}
	for k, v := range b.Options {
		result.Options[k] = v
	}

	// Add the meta map values
	if result.Meta == nil {
		result.Meta = make(map[string]string)
	}
	for k, v := range b.Meta {
		result.Meta[k] = v
	}

	// Add the chroot_env map values
	if result.ChrootEnv == nil {
		result.ChrootEnv = make(map[string]string)
	}
	for k, v := range b.ChrootEnv {
		result.ChrootEnv[k] = v
	}

	return &result
}

// Merge is used to merge two telemetry configs together
func (a *Telemetry) Merge(b *Telemetry) *Telemetry {
	result := *a

	if b.StatsiteAddr != "" {
		result.StatsiteAddr = b.StatsiteAddr
	}
	if b.StatsdAddr != "" {
		result.StatsdAddr = b.StatsdAddr
	}
	if b.DisableHostname {
		result.DisableHostname = true
	}
	if b.CollectionInterval != "" {
		result.CollectionInterval = b.CollectionInterval
	}
	if b.collectionInterval != 0 {
		result.collectionInterval = b.collectionInterval
	}
	if b.CirconusAPIToken != "" {
		result.CirconusAPIToken = b.CirconusAPIToken
	}
	if b.CirconusAPIApp != "" {
		result.CirconusAPIApp = b.CirconusAPIApp
	}
	if b.CirconusAPIURL != "" {
		result.CirconusAPIURL = b.CirconusAPIURL
	}
	if b.CirconusCheckSubmissionURL != "" {
		result.CirconusCheckSubmissionURL = b.CirconusCheckSubmissionURL
	}
	if b.CirconusSubmissionInterval != "" {
		result.CirconusSubmissionInterval = b.CirconusSubmissionInterval
	}
	if b.CirconusCheckID != "" {
		result.CirconusCheckID = b.CirconusCheckID
	}
	if b.CirconusCheckForceMetricActivation != "" {
		result.CirconusCheckForceMetricActivation = b.CirconusCheckForceMetricActivation
	}
	if b.CirconusCheckInstanceID != "" {
		result.CirconusCheckInstanceID = b.CirconusCheckInstanceID
	}
	if b.CirconusCheckSearchTag != "" {
		result.CirconusCheckSearchTag = b.CirconusCheckSearchTag
	}
	if b.CirconusBrokerID != "" {
		result.CirconusBrokerID = b.CirconusBrokerID
	}
	if b.CirconusBrokerSelectTag != "" {
		result.CirconusBrokerSelectTag = b.CirconusBrokerSelectTag
	}
	return &result
}

// Merge is used to merge two port configurations.
func (a *Ports) Merge(b *Ports) *Ports {
	result := *a

	if b.HTTP != 0 {
		result.HTTP = b.HTTP
	}
	if b.RPC != 0 {
		result.RPC = b.RPC
	}
	if b.Serf != 0 {
		result.Serf = b.Serf
	}
	return &result
}

// Merge is used to merge two address configs together.
func (a *Addresses) Merge(b *Addresses) *Addresses {
	result := *a

	if b.HTTP != "" {
		result.HTTP = b.HTTP
	}
	if b.RPC != "" {
		result.RPC = b.RPC
	}
	if b.Serf != "" {
		result.Serf = b.Serf
	}
	return &result
}

// Merge merges two advertise addrs configs together.
func (a *AdvertiseAddrs) Merge(b *AdvertiseAddrs) *AdvertiseAddrs {
	result := *a

	if b.RPC != "" {
		result.RPC = b.RPC
	}
	if b.Serf != "" {
		result.Serf = b.Serf
	}
	if b.HTTP != "" {
		result.HTTP = b.HTTP
	}
	return &result
}

// Merge merges two Atlas configurations together.
func (a *AtlasConfig) Merge(b *AtlasConfig) *AtlasConfig {
	result := *a

	if b.Infrastructure != "" {
		result.Infrastructure = b.Infrastructure
	}
	if b.Token != "" {
		result.Token = b.Token
	}
	if b.Join {
		result.Join = true
	}
	if b.Endpoint != "" {
		result.Endpoint = b.Endpoint
	}
	return &result
}

func (r *Resources) Merge(b *Resources) *Resources {
	result := *r
	if b.CPU != 0 {
		result.CPU = b.CPU
	}
	if b.MemoryMB != 0 {
		result.MemoryMB = b.MemoryMB
	}
	if b.DiskMB != 0 {
		result.DiskMB = b.DiskMB
	}
	if b.IOPS != 0 {
		result.IOPS = b.IOPS
	}
	if b.ReservedPorts != "" {
		result.ReservedPorts = b.ReservedPorts
	}
	if len(b.ParsedReservedPorts) != 0 {
		result.ParsedReservedPorts = b.ParsedReservedPorts
	}
	return &result
}

// LoadConfig loads the configuration at the given path, regardless if
// its a file or directory.
func LoadConfig(path string) (*Config, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return LoadConfigDir(path)
	}

	cleaned := filepath.Clean(path)
	config, err := ParseConfigFile(cleaned)
	if err != nil {
		return nil, fmt.Errorf("Error loading %s: %s", cleaned, err)
	}

	config.Files = append(config.Files, cleaned)
	return config, nil
}

// LoadConfigDir loads all the configurations in the given directory
// in alphabetical order.
func LoadConfigDir(dir string) (*Config, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf(
			"configuration path must be a directory: %s", dir)
	}

	var files []string
	err = nil
	for err != io.EOF {
		var fis []os.FileInfo
		fis, err = f.Readdir(128)
		if err != nil && err != io.EOF {
			return nil, err
		}

		for _, fi := range fis {
			// Ignore directories
			if fi.IsDir() {
				continue
			}

			// Only care about files that are valid to load.
			name := fi.Name()
			skip := true
			if strings.HasSuffix(name, ".hcl") {
				skip = false
			} else if strings.HasSuffix(name, ".json") {
				skip = false
			}
			if skip || isTemporaryFile(name) {
				continue
			}

			path := filepath.Join(dir, name)
			files = append(files, path)
		}
	}

	// Fast-path if we have no files
	if len(files) == 0 {
		return &Config{}, nil
	}

	sort.Strings(files)

	var result *Config
	for _, f := range files {
		config, err := ParseConfigFile(f)
		if err != nil {
			return nil, fmt.Errorf("Error loading %s: %s", f, err)
		}
		config.Files = append(config.Files, f)

		if result == nil {
			result = config
		} else {
			result = result.Merge(config)
		}
	}

	return result, nil
}

// isTemporaryFile returns true or false depending on whether the
// provided file name is a temporary file for the following editors:
// emacs or vim.
func isTemporaryFile(name string) bool {
	return strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, ".#") || // emacs
		(strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#")) // emacs
}
