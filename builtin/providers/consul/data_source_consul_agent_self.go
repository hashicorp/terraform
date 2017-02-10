package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	agentSelfACLDatacenter typeKey = iota
	agentSelfACLDefaultPolicy
	agentSelfACLDisableTTL
	agentSelfACLDownPolicy
	agentSelfACLEnforceVersion8
	agentSelfACLTTL
	agentSelfAddresses
	agentSelfAdvertiseAddr
	agentSelfAdvertiseAddrWAN
	agentSelfAdvertiseAddrs
	agentSelfAtlasJoin
	agentSelfBindAddr
	agentSelfBootstrap
	agentSelfBootstrapExpect
	agentSelfCAFile
	agentSelfCertFile
	agentSelfCheckDeregisterIntervalMin
	agentSelfCheckDisableAnonymousSignature
	agentSelfCheckDisableRemoteExec
	agentSelfCheckReapInterval
	agentSelfCheckUpdateInterval
	agentSelfClientAddr
	agentSelfDNSConfig
	agentSelfDNSRecursors
	agentSelfDataDir
	agentSelfDatacenter
	agentSelfDevMode
	agentSelfDisableCoordinates
	agentSelfDisableUpdateCheck
	agentSelfDomain
	agentSelfEnableDebug
	agentSelfEnableSyslog
	agentSelfEnableUI
	agentSelfID
	agentSelfKeyFile
	agentSelfLeaveOnInt
	agentSelfLeaveOnTerm
	agentSelfLogLevel
	agentSelfName
	agentSelfPerformance
	agentSelfPidFile
	agentSelfPorts
	agentSelfProtocol
	agentSelfReconnectTimeoutLAN
	agentSelfReconnectTimeoutWAN
	agentSelfRejoinAfterLeave
	agentSelfRetryJoin
	agentSelfRetryJoinEC2
	agentSelfRetryJoinGCE
	agentSelfRetryJoinWAN
	agentSelfRetryMaxAttempts
	agentSelfRetryMaxAttemptsWAN
	agentSelfRevision
	agentSelfSerfLANBindAddr
	agentSelfSerfWANBindAddr
	agentSelfServer
	agentSelfServerName
	agentSelfSessionTTLMin
	agentSelfStartJoin
	agentSelfStartJoinWAN
	agentSelfSyslogFacility
	agentSelfTLSMinVersion
	agentSelfTaggedAddresses
	agentSelfTelemetry
	agentSelfTranslateWANAddrs
	agentSelfUIDir
	agentSelfUnixSockets
	agentSelfVerifyIncoming
	agentSelfVerifyOutgoing
	agentSelfVerifyServerHostname
	agentSelfVersion
	agentSelfVersionPrerelease
)

const (
	agentSelfDNSAllowStale typeKey = iota
	agentSelfDNSMaxStale
	agentSelfRecursorTimeout
	agentSelfDNSDisableCompression
	agentSelfDNSEnableTruncate
	agentSelfDNSNodeTTL
	agentSelfDNSOnlyPassing
	agentSelfDNSUDPAnswerLimit
	agentSelfServiceTTL
)

const (
	agentSelfPerformanceRaftMultiplier typeKey = iota
)

const (
	agentSelfPortsDNS typeKey = iota
	agentSelfPortsHTTP
	agentSelfPortsHTTPS
	agentSelfPortsRPC
	agentSelfPortsSerfLAN
	agentSelfPortsSerfWAN
	agentSelfPortsServer
)

const (
	agentSelfTaggedAddressesLAN typeKey = iota
	agentSelfTaggedAddressesWAN
)

const (
	agentSelfTelemetryCirconusAPIApp typeKey = iota
	agentSelfTelemetryCirconusAPIURL
	agentSelfTelemetryCirconusBrokerID
	agentSelfTelemetryCirconusBrokerSelectTag
	agentSelfTelemetryCirconusCheckDisplayName
	agentSelfTelemetryCirconusCheckForceMetricActiation
	agentSelfTelemetryCirconusCheckID
	agentSelfTelemetryCirconusCheckInstanceID
	agentSelfTelemetryCirconusCheckSearchTag
	agentSelfTelemetryCirconusCheckSubmissionURL
	agentSelfTelemetryCirconusCheckTags
	agentSelfTelemetryCirconusSubmissionInterval
	agentSelfTelemetryDisableHostname
	agentSelfTelemetryDogStatsdAddr
	agentSelfTelemetryDogStatsdTags
	agentSelfTelemetryStatsdAddr
	agentSelfTelemetryStatsiteAddr
	agentSelfTelemetryStatsitePrefix
)

// Schema for consul's /v1/agent/self endpoint
var agentSelfMap = map[typeKey]*typeEntry{
	agentSelfACLDatacenter: {
		APIName:    "ACLDatacenter",
		SchemaName: "acl_datacenter",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfACLDefaultPolicy: {
		APIName:    "ACLDefaultPolicy",
		SchemaName: "acl_default_policy",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfACLDisableTTL: {
		APIName:    "ACLDisabledTTL",
		SchemaName: "acl_disable_ttl",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfACLDownPolicy: {
		APIName:    "ACLDownPolicy",
		SchemaName: "acl_down_policy",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfACLEnforceVersion8: {
		APIName:    "ACLEnforceVersion8",
		SchemaName: "acl_enforce_version_8",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfACLTTL: {
		APIName:    "ACLTTL",
		SchemaName: "acl_ttl",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfAddresses: {
		APIName:    "Addresses",
		SchemaName: "addresses",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
	},
	agentSelfAdvertiseAddr: {
		APIName:    "AdvertiseAddr",
		SchemaName: "advertise_addr",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfAdvertiseAddrs: {
		APIName:    "AdvertiseAddrs",
		SchemaName: "advertise_addrs",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
	},
	agentSelfAdvertiseAddrWAN: {
		APIName:    "AdvertiseAddrWan",
		SchemaName: "advertise_addr_wan",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	// Omitting the following since they've been depreciated:
	//
	// "AtlasInfrastructure":        "",
	// "AtlasEndpoint":       "",
	agentSelfAtlasJoin: {
		APIName:    "AtlasJoin",
		SchemaName: "atlas_join",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfBindAddr: {
		APIName:    "BindAddr",
		SchemaName: "bind_addr",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfBootstrap: {
		APIName:    "Bootstrap",
		SchemaName: "bootstrap",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfBootstrapExpect: {
		APIName:    "BootstrapExpect",
		SchemaName: "bootstrap_expect",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfCAFile: {
		APIName:    "CAFile",
		SchemaName: "ca_file",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfCertFile: {
		APIName:    "CertFile",
		SchemaName: "cert_file",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfCheckDeregisterIntervalMin: {
		APIName:    "CheckDeregisterIntervalMin",
		SchemaName: "check_deregister_interval_min",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfCheckDisableAnonymousSignature: {
		APIName:    "DisableAnonymousSignature",
		SchemaName: "disable_anonymous_signature",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfCheckDisableRemoteExec: {
		APIName:    "DisableRemoteExec",
		SchemaName: "disable_remote_exec",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfCheckReapInterval: {
		APIName:    "CheckReapInterval",
		SchemaName: "check_reap_interval",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfCheckUpdateInterval: {
		APIName:    "CheckUpdateInterval",
		SchemaName: "check_update_interval",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfClientAddr: {
		APIName:    "ClientAddr",
		SchemaName: "client_addr",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfDNSConfig: {
		APIName:    "DNSConfig",
		SchemaName: "dns_config",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[typeKey]*typeEntry{
			agentSelfDNSAllowStale: {
				APIName:    "AllowStale",
				SchemaName: "allow_stale",
				Source:     sourceAPIResult,
				Type:       schema.TypeBool,
			},
			agentSelfDNSDisableCompression: {
				APIName:    "DisableCompression",
				SchemaName: "disable_compression",
				Source:     sourceAPIResult,
				Type:       schema.TypeBool,
			},
			agentSelfDNSEnableTruncate: {
				APIName:    "EnableTruncate",
				SchemaName: "enable_truncate",
				Source:     sourceAPIResult,
				Type:       schema.TypeBool,
			},
			agentSelfDNSMaxStale: {
				APIName:    "MaxStale",
				SchemaName: "max_stale",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfDNSNodeTTL: {
				APIName:    "NodeTTL",
				SchemaName: "node_ttl",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfDNSOnlyPassing: {
				APIName:    "OnlyPassing",
				SchemaName: "only_passing",
				Source:     sourceAPIResult,
				Type:       schema.TypeBool,
			},
			agentSelfRecursorTimeout: {
				APIName:    "RecursorTimeout",
				SchemaName: "recursor_timeout",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfServiceTTL: {
				APIName:    "ServiceTTL",
				SchemaName: "service_ttl",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfDNSUDPAnswerLimit: {
				APIName:    "UDPAnswerLimit",
				SchemaName: "udp_answer_limit",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
		},
	},
	agentSelfDataDir: {
		APIName:    "DataDir",
		SchemaName: "data_dir",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfDatacenter: {
		APIName:    "Datacenter",
		SchemaName: "datacenter",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfDevMode: {
		APIName:    "DevMode",
		SchemaName: "dev_mode",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfDisableCoordinates: {
		APIName:    "DisableCoordinates",
		SchemaName: "coordinates",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfDisableUpdateCheck: {
		APIName:    "DisableUpdateCheck",
		SchemaName: "update_check",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfDNSRecursors: {
		APIName:    "DNSRecursors",
		APIAliases: []apiAttr{"DNSRecursor"},
		SchemaName: "dns_recursors",
		Source:     sourceAPIResult,
		Type:       schema.TypeList,
	},
	agentSelfDomain: {
		APIName:    "Domain",
		SchemaName: "domain",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfEnableDebug: {
		APIName:    "EnableDebug",
		SchemaName: "debug",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfEnableSyslog: {
		APIName:    "EnableSyslog",
		SchemaName: "syslog",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfEnableUI: {
		APIName:    "EnableUi",
		SchemaName: "ui",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	// "HTTPAPIResponseHeaders": nil,
	agentSelfID: {
		APIName:    "NodeID",
		SchemaName: "id",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
		ValidateFuncs: []interface{}{
			validateRegexp(`(?i)^[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}$`),
		},
		APITest:    apiTestID,
		APIToState: apiToStateID,
	},
	agentSelfKeyFile: {
		APIName:    "KeyFile",
		SchemaName: "key_file",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfLeaveOnInt: {
		APIName:    "SkipLeaveOnInt",
		SchemaName: "leave_on_int",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
		APITest:    apiTestBool,
		APIToState: negateBoolToState(apiToStateBool),
	},
	agentSelfLeaveOnTerm: {
		APIName:    "LeaveOnTerm",
		SchemaName: "leave_on_term",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfLogLevel: {
		APIName:    "LogLevel",
		SchemaName: "log_level",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfName: {
		APIName:    "NodeName",
		SchemaName: "name",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfPerformance: {
		APIName:    "Performance",
		SchemaName: "performance",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[typeKey]*typeEntry{
			agentSelfPerformanceRaftMultiplier: {
				APIName:    "RaftMultiplier",
				SchemaName: "raft_multiplier",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
		},
	},
	agentSelfPidFile: {
		APIName:    "PidFile",
		SchemaName: "pid_file",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfPorts: {
		APIName:    "Ports",
		SchemaName: "ports",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[typeKey]*typeEntry{
			agentSelfPortsDNS: {
				APIName:    "DNS",
				SchemaName: "dns",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfPortsHTTP: {
				APIName:    "HTTP",
				SchemaName: "http",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfPortsHTTPS: {
				APIName:    "HTTPS",
				SchemaName: "https",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfPortsRPC: {
				APIName:    "RPC",
				SchemaName: "rpc",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfPortsSerfLAN: {
				APIName:    "SerfLan",
				SchemaName: "serf_lan",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfPortsSerfWAN: {
				APIName:    "SerfWan",
				SchemaName: "serf_wan",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
			agentSelfPortsServer: {
				APIName:    "Server",
				SchemaName: "server",
				Source:     sourceAPIResult,
				Type:       schema.TypeFloat,
			},
		},
	},
	agentSelfProtocol: {
		APIName:    "Protocol",
		SchemaName: "protocol",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfReconnectTimeoutLAN: {
		APIName:    "ReconnectTimeoutLan",
		SchemaName: "reconnect_timeout_lan",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfReconnectTimeoutWAN: {
		APIName:    "ReconnectTimeoutWan",
		SchemaName: "reconnect_timeout_wan",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfRejoinAfterLeave: {
		APIName:    "RejoinAfterLeave",
		SchemaName: "rejoin_after_leave",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	// "RetryIntervalWanRaw": "",
	agentSelfRetryJoin: {
		APIName:    "RetryJoin",
		SchemaName: "retry_join",
		Source:     sourceAPIResult,
		Type:       schema.TypeList,
	},
	agentSelfRetryJoinWAN: {
		APIName:    "RetryJoinWan",
		SchemaName: "retry_join_wan",
		Source:     sourceAPIResult,
		Type:       schema.TypeList,
	},
	agentSelfRetryMaxAttempts: {
		APIName:    "RetryMaxAttempts",
		SchemaName: "retry_max_attempts",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfRetryMaxAttemptsWAN: {
		APIName:    "RetryMaxAttemptsWan",
		SchemaName: "retry_max_attempts_wan",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfRetryJoinEC2: {
		APIName:    "RetryJoinEC2",
		SchemaName: "retry_join_ec2",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
	},
	agentSelfRetryJoinGCE: {
		APIName:    "RetryJoinGCE",
		SchemaName: "retry_join_GCE",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
	},
	agentSelfRevision: {
		APIName:    "Revision",
		SchemaName: "revision",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfSerfLANBindAddr: {
		APIName:    "SerfLanBindAddr",
		SchemaName: "serf_lan_bind_addr",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfSerfWANBindAddr: {
		APIName:    "SerfWanBindAddr",
		SchemaName: "serf_wan_bind_addr",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfServer: {
		APIName:    "Server",
		SchemaName: "server",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfServerName: {
		APIName:    "ServerName",
		SchemaName: "server_name",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfSessionTTLMin: {
		APIName:    "SessionTTLMin",
		SchemaName: "session_ttl_min",
		Source:     sourceAPIResult,
		Type:       schema.TypeFloat,
	},
	agentSelfStartJoin: {
		APIName:    "StartJoin",
		SchemaName: "start_join",
		Source:     sourceAPIResult,
		Type:       schema.TypeList,
	},
	agentSelfStartJoinWAN: {
		APIName:    "StartJoinWan",
		SchemaName: "start_join_wan",
		Source:     sourceAPIResult,
		Type:       schema.TypeList,
	},
	agentSelfSyslogFacility: {
		APIName:    "SyslogFacility",
		SchemaName: "syslog_facility",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfTaggedAddresses: {
		APIName:    "TaggedAddresses",
		SchemaName: "tagged_addresses",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[typeKey]*typeEntry{
			agentSelfTaggedAddressesLAN: {
				APIName:    "LAN",
				SchemaName: "lan",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTaggedAddressesWAN: {
				APIName:    "WAN",
				SchemaName: "wan",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
		},
	},
	agentSelfTelemetry: {
		APIName:    "Telemetry",
		SchemaName: "telemetry",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[typeKey]*typeEntry{
			agentSelfTelemetryCirconusAPIApp: {
				APIName:    "CirconusAPIApp",
				SchemaName: "circonus_api_app",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusAPIURL: {
				APIName:    "CirconusAPIURL",
				SchemaName: "circonus_api_url",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusBrokerID: {
				APIName:    "CirconusBrokerID",
				SchemaName: "circonus_broker_id",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusBrokerSelectTag: {
				APIName:    "CirconusBrokerSelectTag",
				SchemaName: "circonus_broker_select_tag",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusCheckDisplayName: {
				APIName:    "CirconusCheckDisplayName",
				SchemaName: "circonus_check_display_name",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusCheckForceMetricActiation: {
				APIName:    "CirconusCheckForceMetricActivation",
				SchemaName: "circonus_check_force_metric_activation",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusCheckID: {
				APIName:    "CirconusCheckID",
				SchemaName: "circonus_check_id",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusCheckInstanceID: {
				APIName:    "CirconusCheckInstanceID",
				SchemaName: "circonus_check_instance_id",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusCheckSearchTag: {
				APIName:    "CirconusCheckSearchTag",
				SchemaName: "circonus_check_search_tag",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusCheckSubmissionURL: {
				APIName:    "CirconusCheckSubmissionURL",
				SchemaName: "circonus_check_submission_url",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusCheckTags: {
				APIName:    "CirconusCheckTags",
				SchemaName: "circonus_check_tags",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryCirconusSubmissionInterval: {
				APIName:    "CirconusSubmissionInterval",
				SchemaName: "circonus_submission_interval",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryDisableHostname: {
				APIName:    "DisableHostname",
				SchemaName: "disable_hostname",
				Source:     sourceAPIResult,
				Type:       schema.TypeBool,
			},
			agentSelfTelemetryDogStatsdAddr: {
				APIName:    "DogStatsdAddr",
				SchemaName: "dog_statsd_addr",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryDogStatsdTags: {
				APIName:    "DogStatsdTags",
				SchemaName: "dog_statsd_tags",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryStatsdAddr: {
				APIName:    "StatsdTags",
				SchemaName: "statsd_tags",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryStatsiteAddr: {
				APIName:    "StatsiteAddr",
				SchemaName: "statsite_addr",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
			agentSelfTelemetryStatsitePrefix: {
				APIName:    "StatsitePrefix",
				SchemaName: "statsite_prefix",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
			},
		},
	},
	agentSelfTLSMinVersion: {
		APIName:    "TLSMinVersion",
		SchemaName: "tls_min_version",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfTranslateWANAddrs: {
		APIName:    "TranslateWanAddrs",
		SchemaName: "translate_wan_addrs",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfUIDir: {
		APIName:    "UiDir",
		SchemaName: "ui_dir",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfUnixSockets: {
		APIName:    "UnixSockets",
		SchemaName: "unix_sockets",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
	},
	agentSelfVerifyIncoming: {
		APIName:    "VerifyIncoming",
		SchemaName: "verify_incoming",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfVerifyServerHostname: {
		APIName:    "VerifyServerHostname",
		SchemaName: "verify_server_hostname",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfVerifyOutgoing: {
		APIName:    "VerifyOutgoing",
		SchemaName: "verify_outgoing",
		Source:     sourceAPIResult,
		Type:       schema.TypeBool,
	},
	agentSelfVersion: {
		APIName:    "Version",
		SchemaName: "version",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	agentSelfVersionPrerelease: {
		APIName:    "VersionPrerelease",
		SchemaName: "version_prerelease",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
	},
	// "Watches":                nil,
}

func dataSourceConsulAgentSelf() *schema.Resource {
	return &schema.Resource{
		Read:   dataSourceConsulAgentSelfRead,
		Schema: typeEntryMapToSchema(agentSelfMap),
	}
}

func dataSourceConsulAgentSelfRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	info, err := client.Agent().Self()
	if err != nil {
		return err
	}

	const apiAgentConfig = "Config"
	cfg, ok := info[apiAgentConfig]
	if !ok {
		return fmt.Errorf("No %s info available within provider's agent/self endpoint", apiAgentConfig)
	}

	// TODO(sean@): It'd be nice if this data source had a way of filtering out
	// irrelevant data so only the important bits are persisted in the state file.
	// Something like an attribute mask or even a regexp of matching schema names
	// would be sufficient in the most basic case.  Food for thought.
	dataSourceWriter := newStateWriter(d)

	for k, e := range agentSelfMap {
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
