package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	_AgentSelfACLDatacenter _TypeKey = iota
	_AgentSelfACLDefaultPolicy
	_AgentSelfACLDisableTTL
	_AgentSelfACLDownPolicy
	_AgentSelfACLEnforceVersion8
	_AgentSelfACLTTL
	_AgentSelfAddresses
	_AgentSelfAdvertiseAddr
	_AgentSelfAdvertiseAddrWAN
	_AgentSelfAdvertiseAddrs
	_AgentSelfAtlasJoin
	_AgentSelfBindAddr
	_AgentSelfBootstrap
	_AgentSelfBootstrapExpect
	_AgentSelfCAFile
	_AgentSelfCertFile
	_AgentSelfCheckDeregisterIntervalMin
	_AgentSelfCheckDisableAnonymousSignature
	_AgentSelfCheckDisableRemoteExec
	_AgentSelfCheckReapInterval
	_AgentSelfCheckUpdateInterval
	_AgentSelfClientAddr
	_AgentSelfDNSConfig
	_AgentSelfDNSRecursors
	_AgentSelfDataDir
	_AgentSelfDatacenter
	_AgentSelfDevMode
	_AgentSelfDisableCoordinates
	_AgentSelfDisableUpdateCheck
	_AgentSelfDomain
	_AgentSelfEnableDebug
	_AgentSelfEnableSyslog
	_AgentSelfEnableUI
	_AgentSelfID
	_AgentSelfKeyFile
	_AgentSelfLeaveOnInt
	_AgentSelfLeaveOnTerm
	_AgentSelfLogLevel
	_AgentSelfName
	_AgentSelfPerformance
	_AgentSelfPidFile
	_AgentSelfPorts
	_AgentSelfProtocol
	_AgentSelfReconnectTimeoutLAN
	_AgentSelfReconnectTimeoutWAN
	_AgentSelfRejoinAfterLeave
	_AgentSelfRetryJoin
	_AgentSelfRetryJoinEC2
	_AgentSelfRetryJoinGCE
	_AgentSelfRetryJoinWAN
	_AgentSelfRetryMaxAttempts
	_AgentSelfRetryMaxAttemptsWAN
	_AgentSelfRevision
	_AgentSelfSerfLANBindAddr
	_AgentSelfSerfWANBindAddr
	_AgentSelfServer
	_AgentSelfServerName
	_AgentSelfSessionTTLMin
	_AgentSelfStartJoin
	_AgentSelfStartJoinWAN
	_AgentSelfSyslogFacility
	_AgentSelfTLSMinVersion
	_AgentSelfTaggedAddresses
	_AgentSelfTelemetry
	_AgentSelfTranslateWANAddrs
	_AgentSelfUIDir
	_AgentSelfUnixSockets
	_AgentSelfVerifyIncoming
	_AgentSelfVerifyOutgoing
	_AgentSelfVerifyServerHostname
	_AgentSelfVersion
	_AgentSelfVersionPrerelease
)

const (
	_AgentSelfDNSAllowStale _TypeKey = iota
	_AgentSelfDNSMaxStale
	_AgentSelfRecursorTimeout
	_AgentSelfDNSDisableCompression
	_AgentSelfDNSEnableTruncate
	_AgentSelfDNSNodeTTL
	_AgentSelfDNSOnlyPassing
	_AgentSelfDNSUDPAnswerLimit
	_AgentSelfServiceTTL
)

const (
	_AgentSelfPerformanceRaftMultiplier _TypeKey = iota
)

const (
	_AgentSelfPortsDNS _TypeKey = iota
	_AgentSelfPortsHTTP
	_AgentSelfPortsHTTPS
	_AgentSelfPortsRPC
	_AgentSelfPortsSerfLAN
	_AgentSelfPortsSerfWAN
	_AgentSelfPortsServer
)

const (
	_AgentSelfTaggedAddressesLAN _TypeKey = iota
	_AgentSelfTaggedAddressesWAN
)

const (
	_AgentSelfTelemetryCirconusAPIApp _TypeKey = iota
	_AgentSelfTelemetryCirconusAPIURL
	_AgentSelfTelemetryCirconusBrokerID
	_AgentSelfTelemetryCirconusBrokerSelectTag
	_AgentSelfTelemetryCirconusCheckDisplayName
	_AgentSelfTelemetryCirconusCheckForceMetricActiation
	_AgentSelfTelemetryCirconusCheckID
	_AgentSelfTelemetryCirconusCheckInstanceID
	_AgentSelfTelemetryCirconusCheckSearchTag
	_AgentSelfTelemetryCirconusCheckSubmissionURL
	_AgentSelfTelemetryCirconusCheckTags
	_AgentSelfTelemetryCirconusSubmissionInterval
	_AgentSelfTelemetryDisableHostname
	_AgentSelfTelemetryDogStatsdAddr
	_AgentSelfTelemetryDogStatsdTags
	_AgentSelfTelemetryStatsdAddr
	_AgentSelfTelemetryStatsiteAddr
	_AgentSelfTelemetryStatsitePrefix
)

// Schema for consul's /v1/agent/self endpoint
var _AgentSelfMap = map[_TypeKey]*_TypeEntry{
	_AgentSelfACLDatacenter: {
		APIName:    "ACLDatacenter",
		SchemaName: "acl_datacenter",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfACLDefaultPolicy: {
		APIName:    "ACLDefaultPolicy",
		SchemaName: "acl_default_policy",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfACLDisableTTL: {
		APIName:    "ACLDisabledTTL",
		SchemaName: "acl_disable_ttl",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfACLDownPolicy: {
		APIName:    "ACLDownPolicy",
		SchemaName: "acl_down_policy",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfACLEnforceVersion8: {
		APIName:    "ACLEnforceVersion8",
		SchemaName: "acl_enforce_version_8",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfACLTTL: {
		APIName:    "ACLTTL",
		SchemaName: "acl_ttl",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfAddresses: {
		APIName:    "Addresses",
		SchemaName: "addresses",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
	},
	_AgentSelfAdvertiseAddr: {
		APIName:    "AdvertiseAddr",
		SchemaName: "advertise_addr",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfAdvertiseAddrs: {
		APIName:    "AdvertiseAddrs",
		SchemaName: "advertise_addrs",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
	},
	_AgentSelfAdvertiseAddrWAN: {
		APIName:    "AdvertiseAddrWan",
		SchemaName: "advertise_addr_wan",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	// Omitting the following since they've been depreciated:
	//
	// "AtlasInfrastructure":        "",
	// "AtlasEndpoint":       "",
	_AgentSelfAtlasJoin: {
		APIName:    "AtlasJoin",
		SchemaName: "atlas_join",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfBindAddr: {
		APIName:    "BindAddr",
		SchemaName: "bind_addr",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfBootstrap: {
		APIName:    "Bootstrap",
		SchemaName: "bootstrap",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfBootstrapExpect: {
		APIName:    "BootstrapExpect",
		SchemaName: "bootstrap_expect",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfCAFile: {
		APIName:    "CAFile",
		SchemaName: "ca_file",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfCertFile: {
		APIName:    "CertFile",
		SchemaName: "cert_file",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfCheckDeregisterIntervalMin: {
		APIName:    "CheckDeregisterIntervalMin",
		SchemaName: "check_deregister_interval_min",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfCheckDisableAnonymousSignature: {
		APIName:    "DisableAnonymousSignature",
		SchemaName: "disable_anonymous_signature",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfCheckDisableRemoteExec: {
		APIName:    "DisableRemoteExec",
		SchemaName: "disable_remote_exec",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfCheckReapInterval: {
		APIName:    "CheckReapInterval",
		SchemaName: "check_reap_interval",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfCheckUpdateInterval: {
		APIName:    "CheckUpdateInterval",
		SchemaName: "check_update_interval",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfClientAddr: {
		APIName:    "ClientAddr",
		SchemaName: "client_addr",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfDNSConfig: {
		APIName:    "DNSConfig",
		SchemaName: "dns_config",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[_TypeKey]*_TypeEntry{
			_AgentSelfDNSAllowStale: {
				APIName:    "AllowStale",
				SchemaName: "allow_stale",
				Source:     _SourceAPIResult,
				Type:       schema.TypeBool,
			},
			_AgentSelfDNSDisableCompression: {
				APIName:    "DisableCompression",
				SchemaName: "disable_compression",
				Source:     _SourceAPIResult,
				Type:       schema.TypeBool,
			},
			_AgentSelfDNSEnableTruncate: {
				APIName:    "EnableTruncate",
				SchemaName: "enable_truncate",
				Source:     _SourceAPIResult,
				Type:       schema.TypeBool,
			},
			_AgentSelfDNSMaxStale: {
				APIName:    "MaxStale",
				SchemaName: "max_stale",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfDNSNodeTTL: {
				APIName:    "NodeTTL",
				SchemaName: "node_ttl",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfDNSOnlyPassing: {
				APIName:    "OnlyPassing",
				SchemaName: "only_passing",
				Source:     _SourceAPIResult,
				Type:       schema.TypeBool,
			},
			_AgentSelfRecursorTimeout: {
				APIName:    "RecursorTimeout",
				SchemaName: "recursor_timeout",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfServiceTTL: {
				APIName:    "ServiceTTL",
				SchemaName: "service_ttl",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfDNSUDPAnswerLimit: {
				APIName:    "UDPAnswerLimit",
				SchemaName: "udp_answer_limit",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
		},
	},
	_AgentSelfDataDir: {
		APIName:    "DataDir",
		SchemaName: "data_dir",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfDatacenter: {
		APIName:    "Datacenter",
		SchemaName: "datacenter",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfDevMode: {
		APIName:    "DevMode",
		SchemaName: "dev_mode",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfDisableCoordinates: {
		APIName:    "DisableCoordinates",
		SchemaName: "coordinates",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfDisableUpdateCheck: {
		APIName:    "DisableUpdateCheck",
		SchemaName: "update_check",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfDNSRecursors: {
		APIName:    "DNSRecursors",
		APIAliases: []_APIAttr{"DNSRecursor"},
		SchemaName: "dns_recursors",
		Source:     _SourceAPIResult,
		Type:       schema.TypeList,
	},
	_AgentSelfDomain: {
		APIName:    "Domain",
		SchemaName: "domain",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfEnableDebug: {
		APIName:    "EnableDebug",
		SchemaName: "debug",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfEnableSyslog: {
		APIName:    "EnableSyslog",
		SchemaName: "syslog",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfEnableUI: {
		APIName:    "EnableUi",
		SchemaName: "ui",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	// "HTTPAPIResponseHeaders": nil,
	_AgentSelfID: {
		APIName:    "NodeID",
		SchemaName: "id",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
		ValidateFuncs: []interface{}{
			_ValidateRegexp(`(?i)^[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}$`),
		},
		APITest:    _APITestID,
		APIToState: _APIToStateID,
	},
	_AgentSelfKeyFile: {
		APIName:    "KeyFile",
		SchemaName: "key_file",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfLeaveOnInt: {
		APIName:    "SkipLeaveOnInt",
		SchemaName: "leave_on_int",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
		APITest:    _APITestBool,
		APIToState: _NegateBoolToState(_APIToStateBool),
	},
	_AgentSelfLeaveOnTerm: {
		APIName:    "LeaveOnTerm",
		SchemaName: "leave_on_term",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfLogLevel: {
		APIName:    "LogLevel",
		SchemaName: "log_level",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfName: {
		APIName:    "NodeName",
		SchemaName: "name",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfPerformance: {
		APIName:    "Performance",
		SchemaName: "performance",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[_TypeKey]*_TypeEntry{
			_AgentSelfPerformanceRaftMultiplier: {
				APIName:    "RaftMultiplier",
				SchemaName: "raft_multiplier",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
		},
	},
	_AgentSelfPidFile: {
		APIName:    "PidFile",
		SchemaName: "pid_file",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfPorts: {
		APIName:    "Ports",
		SchemaName: "ports",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[_TypeKey]*_TypeEntry{
			_AgentSelfPortsDNS: {
				APIName:    "DNS",
				SchemaName: "dns",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfPortsHTTP: {
				APIName:    "HTTP",
				SchemaName: "http",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfPortsHTTPS: {
				APIName:    "HTTPS",
				SchemaName: "https",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfPortsRPC: {
				APIName:    "RPC",
				SchemaName: "rpc",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfPortsSerfLAN: {
				APIName:    "SerfLan",
				SchemaName: "serf_lan",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfPortsSerfWAN: {
				APIName:    "SerfWan",
				SchemaName: "serf_wan",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
			_AgentSelfPortsServer: {
				APIName:    "Server",
				SchemaName: "server",
				Source:     _SourceAPIResult,
				Type:       schema.TypeFloat,
			},
		},
	},
	_AgentSelfProtocol: {
		APIName:    "Protocol",
		SchemaName: "protocol",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfReconnectTimeoutLAN: {
		APIName:    "ReconnectTimeoutLan",
		SchemaName: "reconnect_timeout_lan",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfReconnectTimeoutWAN: {
		APIName:    "ReconnectTimeoutWan",
		SchemaName: "reconnect_timeout_wan",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfRejoinAfterLeave: {
		APIName:    "RejoinAfterLeave",
		SchemaName: "rejoin_after_leave",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	// "RetryIntervalWanRaw": "",
	_AgentSelfRetryJoin: {
		APIName:    "RetryJoin",
		SchemaName: "retry_join",
		Source:     _SourceAPIResult,
		Type:       schema.TypeList,
	},
	_AgentSelfRetryJoinWAN: {
		APIName:    "RetryJoinWan",
		SchemaName: "retry_join_wan",
		Source:     _SourceAPIResult,
		Type:       schema.TypeList,
	},
	_AgentSelfRetryMaxAttempts: {
		APIName:    "RetryMaxAttempts",
		SchemaName: "retry_max_attempts",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfRetryMaxAttemptsWAN: {
		APIName:    "RetryMaxAttemptsWan",
		SchemaName: "retry_max_attempts_wan",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfRetryJoinEC2: {
		APIName:    "RetryJoinEC2",
		SchemaName: "retry_join_ec2",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
	},
	_AgentSelfRetryJoinGCE: {
		APIName:    "RetryJoinGCE",
		SchemaName: "retry_join_GCE",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
	},
	_AgentSelfRevision: {
		APIName:    "Revision",
		SchemaName: "revision",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfSerfLANBindAddr: {
		APIName:    "SerfLanBindAddr",
		SchemaName: "serf_lan_bind_addr",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfSerfWANBindAddr: {
		APIName:    "SerfWanBindAddr",
		SchemaName: "serf_wan_bind_addr",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfServer: {
		APIName:    "Server",
		SchemaName: "server",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfServerName: {
		APIName:    "ServerName",
		SchemaName: "server_name",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfSessionTTLMin: {
		APIName:    "SessionTTLMin",
		SchemaName: "session_ttl_min",
		Source:     _SourceAPIResult,
		Type:       schema.TypeFloat,
	},
	_AgentSelfStartJoin: {
		APIName:    "StartJoin",
		SchemaName: "start_join",
		Source:     _SourceAPIResult,
		Type:       schema.TypeList,
	},
	_AgentSelfStartJoinWAN: {
		APIName:    "StartJoinWan",
		SchemaName: "start_join_wan",
		Source:     _SourceAPIResult,
		Type:       schema.TypeList,
	},
	_AgentSelfSyslogFacility: {
		APIName:    "SyslogFacility",
		SchemaName: "syslog_facility",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfTaggedAddresses: {
		APIName:    "TaggedAddresses",
		SchemaName: "tagged_addresses",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[_TypeKey]*_TypeEntry{
			_AgentSelfTaggedAddressesLAN: {
				APIName:    "LAN",
				SchemaName: "lan",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTaggedAddressesWAN: {
				APIName:    "WAN",
				SchemaName: "wan",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
		},
	},
	_AgentSelfTelemetry: {
		APIName:    "Telemetry",
		SchemaName: "telemetry",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[_TypeKey]*_TypeEntry{
			_AgentSelfTelemetryCirconusAPIApp: {
				APIName:    "CirconusAPIApp",
				SchemaName: "circonus_api_app",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusAPIURL: {
				APIName:    "CirconusAPIURL",
				SchemaName: "circonus_api_url",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusBrokerID: {
				APIName:    "CirconusBrokerID",
				SchemaName: "circonus_broker_id",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusBrokerSelectTag: {
				APIName:    "CirconusBrokerSelectTag",
				SchemaName: "circonus_broker_select_tag",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusCheckDisplayName: {
				APIName:    "CirconusCheckDisplayName",
				SchemaName: "circonus_check_display_name",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusCheckForceMetricActiation: {
				APIName:    "CirconusCheckForceMetricActivation",
				SchemaName: "circonus_check_force_metric_activation",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusCheckID: {
				APIName:    "CirconusCheckID",
				SchemaName: "circonus_check_id",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusCheckInstanceID: {
				APIName:    "CirconusCheckInstanceID",
				SchemaName: "circonus_check_instance_id",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusCheckSearchTag: {
				APIName:    "CirconusCheckSearchTag",
				SchemaName: "circonus_check_search_tag",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusCheckSubmissionURL: {
				APIName:    "CirconusCheckSubmissionURL",
				SchemaName: "circonus_check_submission_url",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusCheckTags: {
				APIName:    "CirconusCheckTags",
				SchemaName: "circonus_check_tags",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryCirconusSubmissionInterval: {
				APIName:    "CirconusSubmissionInterval",
				SchemaName: "circonus_submission_interval",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryDisableHostname: {
				APIName:    "DisableHostname",
				SchemaName: "disable_hostname",
				Source:     _SourceAPIResult,
				Type:       schema.TypeBool,
			},
			_AgentSelfTelemetryDogStatsdAddr: {
				APIName:    "DogStatsdAddr",
				SchemaName: "dog_statsd_addr",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryDogStatsdTags: {
				APIName:    "DogStatsdTags",
				SchemaName: "dog_statsd_tags",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryStatsdAddr: {
				APIName:    "StatsdTags",
				SchemaName: "statsd_tags",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryStatsiteAddr: {
				APIName:    "StatsiteAddr",
				SchemaName: "statsite_addr",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
			_AgentSelfTelemetryStatsitePrefix: {
				APIName:    "StatsitePrefix",
				SchemaName: "statsite_prefix",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
			},
		},
	},
	_AgentSelfTLSMinVersion: {
		APIName:    "TLSMinVersion",
		SchemaName: "tls_min_version",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfTranslateWANAddrs: {
		APIName:    "TranslateWanAddrs",
		SchemaName: "translate_wan_addrs",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfUIDir: {
		APIName:    "UiDir",
		SchemaName: "ui_dir",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfUnixSockets: {
		APIName:    "UnixSockets",
		SchemaName: "unix_sockets",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
	},
	_AgentSelfVerifyIncoming: {
		APIName:    "VerifyIncoming",
		SchemaName: "verify_incoming",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfVerifyServerHostname: {
		APIName:    "VerifyServerHostname",
		SchemaName: "verify_server_hostname",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfVerifyOutgoing: {
		APIName:    "VerifyOutgoing",
		SchemaName: "verify_outgoing",
		Source:     _SourceAPIResult,
		Type:       schema.TypeBool,
	},
	_AgentSelfVersion: {
		APIName:    "Version",
		SchemaName: "version",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	_AgentSelfVersionPrerelease: {
		APIName:    "VersionPrerelease",
		SchemaName: "version_prerelease",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
	},
	// "Watches":                nil,
}

func dataSourceConsulAgentSelf() *schema.Resource {
	return &schema.Resource{
		Read:   dataSourceConsulAgentSelfRead,
		Schema: _TypeEntryMapToSchema(_AgentSelfMap),
	}
}

func dataSourceConsulAgentSelfRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	info, err := client.Agent().Self()
	if err != nil {
		return err
	}

	const _APIAgentConfig = "Config"
	cfg, ok := info[_APIAgentConfig]
	if !ok {
		return fmt.Errorf("No %s info available within provider's agent/self endpoint", _APIAgentConfig)
	}

	// TODO(sean@): It'd be nice if this data source had a way of filtering out
	// irrelevant data so only the important bits are persisted in the state file.
	// Something like an attribute mask or even a regexp of matching schema names
	// would be sufficient in the most basic case.  Food for thought.
	dataSourceWriter := _NewStateWriter(d)

	for k, e := range _AgentSelfMap {
		apiTest := e.APITest
		if apiTest == nil {
			apiTest = e.LookupDefaultTypeHandler().APITest
		}
		if apiTest == nil {
			panic(fmt.Sprintf("PROVIDER BUG: %v missing APITest method", k))
		}

		apiToState := e.APIToState
		if apiToState == nil {
			apiToState = e.LookupDefaultTypeHandler().APIToState
		}
		if apiToState == nil {
			panic(fmt.Sprintf("PROVIDER BUG: %v missing APIToState method", k))
		}

		if v, ok := apiTest(e, cfg); ok {
			if err := apiToState(e, v, dataSourceWriter); err != nil {
				return errwrap.Wrapf(fmt.Sprintf("error writing %q's data to state: {{err}}", e.SchemaName), err)
			}
		}
	}

	return nil
}
