package consul

import (
	"fmt"
	"strconv"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	agentSelfACLDatacenter              = "acl_datacenter"
	agentSelfACLDefaultPolicy           = "acl_default_policy"
	agentSelfACLDisabledTTL             = "acl_disabled_ttl"
	agentSelfACLDownPolicy              = "acl_down_policy"
	agentSelfACLEnforceVersion8         = "acl_enforce_0_8_semantics"
	agentSelfACLTTL                     = "acl_ttl"
	agentSelfAddresses                  = "addresses"
	agentSelfAdvertiseAddr              = "advertise_addr"
	agentSelfAdvertiseAddrWAN           = "advertise_addr_wan"
	agentSelfAdvertiseAddrs             = "advertise_addrs"
	agentSelfAtlasJoin                  = "atlas_join"
	agentSelfBindAddr                   = "bind_addr"
	agentSelfBootstrapExpect            = "bootstrap_expect"
	agentSelfBootstrapMode              = "bootstrap_mode"
	agentSelfCheckDeregisterIntervalMin = "check_deregister_interval_min"
	agentSelfCheckReapInterval          = "check_reap_interval"
	agentSelfCheckUpdateInterval        = "check_update_interval"
	agentSelfClientAddr                 = "client_addr"
	agentSelfDNSConfig                  = "dns"
	agentSelfDNSRecursors               = "dns_recursors"
	agentSelfDataDir                    = "data_dir"
	agentSelfDatacenter                 = "datacenter"
	agentSelfDevMode                    = "dev_mode"
	agentSelfDomain                     = "domain"
	agentSelfEnableAnonymousSignature   = "enable_anonymous_signature"
	agentSelfEnableCoordinates          = "enable_coordinates"
	agentSelfEnableDebug                = "enable_debug"
	agentSelfEnableRemoteExec           = "enable_remote_exec"
	agentSelfEnableSyslog               = "enable_syslog"
	agentSelfEnableUI                   = "enable_ui"
	agentSelfEnableUpdateCheck          = "enable_update_check"
	agentSelfID                         = "id"
	agentSelfLeaveOnInt                 = "leave_on_int"
	agentSelfLeaveOnTerm                = "leave_on_term"
	agentSelfLogLevel                   = "log_level"
	agentSelfName                       = "name"
	agentSelfPerformance                = "performance"
	agentSelfPidFile                    = "pid_file"
	agentSelfPorts                      = "ports"
	agentSelfProtocol                   = "protocol_version"
	agentSelfReconnectTimeoutLAN        = "reconnect_timeout_lan"
	agentSelfReconnectTimeoutWAN        = "reconnect_timeout_wan"
	agentSelfRejoinAfterLeave           = "rejoin_after_leave"
	agentSelfRetryJoin                  = "retry_join"
	agentSelfRetryJoinEC2               = "retry_join_ec2"
	agentSelfRetryJoinGCE               = "retry_join_gce"
	agentSelfRetryJoinWAN               = "retry_join_wan"
	agentSelfRetryMaxAttempts           = "retry_max_attempts"
	agentSelfRetryMaxAttemptsWAN        = "retry_max_attempts_wan"
	agentSelfSerfLANBindAddr            = "serf_lan_bind_addr"
	agentSelfSerfWANBindAddr            = "serf_wan_bind_addr"
	agentSelfServerMode                 = "server_mode"
	agentSelfServerName                 = "server_name"
	agentSelfSessionTTLMin              = "session_ttl_min"
	agentSelfStartJoin                  = "start_join"
	agentSelfStartJoinWAN               = "start_join_wan"
	agentSelfSyslogFacility             = "syslog_facility"
	agentSelfTLSCAFile                  = "tls_ca_file"
	agentSelfTLSCertFile                = "tls_cert_file"
	agentSelfTLSKeyFile                 = "tls_key_file"
	agentSelfTLSMinVersion              = "tls_min_version"
	agentSelfTLSVerifyIncoming          = "tls_verify_incoming"
	agentSelfTLSVerifyOutgoing          = "tls_verify_outgoing"
	agentSelfTLSVerifyServerHostname    = "tls_verify_server_hostname"
	agentSelfTaggedAddresses            = "tagged_addresses"
	agentSelfTelemetry                  = "telemetry"
	agentSelfTranslateWANAddrs          = "translate_wan_addrs"
	agentSelfUIDir                      = "ui_dir"
	agentSelfUnixSockets                = "unix_sockets"
	agentSelfVersion                    = "version"
	agentSelfVersionPrerelease          = "version_prerelease"
	agentSelfVersionRevision            = "version_revision"
)

const (
	agentSelfRetryJoinAWSRegion   = "region"
	agentSelfRetryJoinAWSTagKey   = "tag_key"
	agentSelfRetryJoinAWSTagValue = "tag_value"
)

const (
	agentSelfRetryJoinGCECredentialsFile = "credentials_file"
	agentSelfRetryJoinGCEProjectName     = "project_name"
	agentSelfRetryJoinGCETagValue        = "tag_value"
	agentSelfRetryJoinGCEZonePattern     = "zone_pattern"
)

const (
	agentSelfDNSAllowStale        = "allow_stale"
	agentSelfDNSEnableCompression = "enable_compression"
	agentSelfDNSEnableTruncate    = "enable_truncate"
	agentSelfDNSMaxStale          = "max_stale"
	agentSelfDNSNodeTTL           = "node_ttl"
	agentSelfDNSOnlyPassing       = "only_passing"
	agentSelfDNSRecursorTimeout   = "recursor_timeout"
	agentSelfDNSServiceTTL        = "service_ttl"
	agentSelfDNSUDPAnswerLimit    = "udp_answer_limit"
)

const (
	agentSelfPerformanceRaftMultiplier = "raft_multiplier"
)

const (
	agentSelfAPIPortsDNS     = "dns"
	agentSelfAPIPortsHTTP    = "http"
	agentSelfAPIPortsHTTPS   = "https"
	agentSelfAPIPortsRPC     = "rpc"
	agentSelfAPIPortsSerfLAN = "serf_lan"
	agentSelfAPIPortsSerfWAN = "serf_wan"
	agentSelfAPIPortsServer  = "server"

	agentSelfSchemaPortsDNS     = "dns"
	agentSelfSchemaPortsHTTP    = "http"
	agentSelfSchemaPortsHTTPS   = "https"
	agentSelfSchemaPortsRPC     = "rpc"
	agentSelfSchemaPortsSerfLAN = "serf_lan"
	agentSelfSchemaPortsSerfWAN = "serf_wan"
	agentSelfSchemaPortsServer  = "server"
)

const (
	agentSelfTaggedAddressesLAN = "lan"
	agentSelfTaggedAddressesWAN = "wan"
)

const (
	agentSelfTelemetryCirconusAPIApp                    = "circonus_api_app"
	agentSelfTelemetryCirconusAPIToken                  = "circonus_api_token"
	agentSelfTelemetryCirconusAPIURL                    = "circonus_api_url"
	agentSelfTelemetryCirconusBrokerID                  = "circonus_broker_id"
	agentSelfTelemetryCirconusBrokerSelectTag           = "circonus_select_tag"
	agentSelfTelemetryCirconusCheckDisplayName          = "circonus_display_name"
	agentSelfTelemetryCirconusCheckForceMetricActiation = "circonus_force_metric_activation"
	agentSelfTelemetryCirconusCheckID                   = "circonus_check_id"
	agentSelfTelemetryCirconusCheckInstanceID           = "circonus_instance_id"
	agentSelfTelemetryCirconusCheckSearchTag            = "circonus_search_tag"
	agentSelfTelemetryCirconusCheckSubmissionURL        = "circonus_submission_url"
	agentSelfTelemetryCirconusCheckTags                 = "circonus_check_tags"
	agentSelfTelemetryCirconusSubmissionInterval        = "circonus_submission_interval"

	agentSelfTelemetryDogStatsdAddr  = "dogstatsd_addr"
	agentSelfTelemetryDogStatsdTags  = "dogstatsd_tags"
	agentSelfTelemetryEnableHostname = "enable_hostname"
	agentSelfTelemetryStatsdAddr     = "statsd_addr"
	agentSelfTelemetryStatsiteAddr   = "statsite_addr"
	agentSelfTelemetryStatsitePrefix = "statsite_prefix"
)

const (
	agentSelfUnixSocketGroup = "group"
	agentSelfUnixSocketMode  = "mode"
	agentSelfUnixSocketUser  = "user"
)

func dataSourceConsulAgentSelf() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceConsulAgentSelfRead,
		Schema: map[string]*schema.Schema{
			agentSelfACLDatacenter: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfACLDefaultPolicy: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfACLDisabledTTL: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfACLDownPolicy: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfACLEnforceVersion8: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfACLTTL: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfAddresses: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfSchemaPortsDNS: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfSchemaPortsHTTP: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfSchemaPortsHTTPS: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfSchemaPortsRPC: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			agentSelfAdvertiseAddr: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfAdvertiseAddrs: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfSchemaPortsSerfLAN: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfSchemaPortsSerfWAN: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfSchemaPortsRPC: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			agentSelfAdvertiseAddrWAN: {
				Computed: true,
				Type:     schema.TypeString,
			},
			// Omitting the following since they've been depreciated:
			//
			// "AtlasInfrastructure":        "",
			// "AtlasEndpoint":       "",
			agentSelfAtlasJoin: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfBindAddr: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfBootstrapMode: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfBootstrapExpect: {
				Computed: true,
				Type:     schema.TypeInt,
			},
			agentSelfCheckDeregisterIntervalMin: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfCheckReapInterval: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfCheckUpdateInterval: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfClientAddr: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfDNSConfig: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfDNSAllowStale: {
							Computed: true,
							Type:     schema.TypeBool,
						},
						agentSelfDNSEnableCompression: {
							Computed: true,
							Type:     schema.TypeBool,
						},
						agentSelfDNSEnableTruncate: {
							Computed: true,
							Type:     schema.TypeBool,
						},
						agentSelfDNSMaxStale: {
							Computed: true,
							Type:     schema.TypeString,
						},
						agentSelfDNSNodeTTL: {
							Computed: true,
							Type:     schema.TypeString,
						},
						agentSelfDNSOnlyPassing: {
							Computed: true,
							Type:     schema.TypeBool,
						},
						agentSelfDNSRecursorTimeout: {
							Computed: true,
							Type:     schema.TypeString,
						},
						agentSelfDNSServiceTTL: {
							Computed: true,
							Type:     schema.TypeString,
						},
						agentSelfDNSUDPAnswerLimit: {
							Computed: true,
							Type:     schema.TypeInt,
						},
					},
				},
			},
			agentSelfDataDir: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfDatacenter: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfDevMode: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfEnableAnonymousSignature: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfEnableCoordinates: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfEnableRemoteExec: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfEnableUpdateCheck: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfDNSRecursors: {
				Computed: true,
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			agentSelfDomain: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfEnableDebug: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfEnableSyslog: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfEnableUI: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			// "HTTPAPIResponseHeaders": nil, // TODO(sean@)
			agentSelfID: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfLeaveOnInt: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfLeaveOnTerm: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfLogLevel: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfName: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfPerformance: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfPerformanceRaftMultiplier: {
							Computed: true,
							Type:     schema.TypeString, // FIXME(sean@): should be schema.TypeInt
						},
					},
				},
			},
			agentSelfPidFile: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfPorts: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfSchemaPortsDNS: {
							Computed: true,
							Type:     schema.TypeInt,
						},
						agentSelfSchemaPortsHTTP: {
							Computed: true,
							Type:     schema.TypeInt,
						},
						agentSelfSchemaPortsHTTPS: {
							Computed: true,
							Type:     schema.TypeInt,
						},
						agentSelfSchemaPortsRPC: {
							Computed: true,
							Type:     schema.TypeInt,
						},
						agentSelfSchemaPortsSerfLAN: {
							Computed: true,
							Type:     schema.TypeInt,
						},
						agentSelfSchemaPortsSerfWAN: {
							Computed: true,
							Type:     schema.TypeInt,
						},
						agentSelfSchemaPortsServer: {
							Computed: true,
							Type:     schema.TypeInt,
						},
					},
				},
			},
			agentSelfProtocol: {
				Computed: true,
				Type:     schema.TypeInt,
			},
			agentSelfReconnectTimeoutLAN: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfReconnectTimeoutWAN: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfRejoinAfterLeave: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfRetryJoin: {
				Computed: true,
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			agentSelfRetryJoinWAN: {
				Computed: true,
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			agentSelfRetryMaxAttempts: {
				Computed: true,
				Type:     schema.TypeInt,
			},
			agentSelfRetryMaxAttemptsWAN: {
				Computed: true,
				Type:     schema.TypeInt,
			},
			agentSelfRetryJoinEC2: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfRetryJoinAWSRegion: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfRetryJoinAWSTagKey: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfRetryJoinAWSTagValue: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			agentSelfRetryJoinGCE: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfRetryJoinGCEProjectName: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfRetryJoinGCEZonePattern: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfRetryJoinGCETagValue: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfRetryJoinGCECredentialsFile: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			agentSelfSerfLANBindAddr: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfSerfWANBindAddr: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfServerMode: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfServerName: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfSessionTTLMin: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfStartJoin: {
				Computed: true,
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			agentSelfStartJoinWAN: {
				Computed: true,
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			agentSelfSyslogFacility: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfTaggedAddresses: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfTaggedAddressesLAN: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTaggedAddressesWAN: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			agentSelfTelemetry: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfTelemetryCirconusAPIApp: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusAPIToken: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusAPIURL: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusBrokerID: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusBrokerSelectTag: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusCheckDisplayName: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusCheckID: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusCheckInstanceID: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusCheckSearchTag: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusCheckSubmissionURL: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusCheckTags: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryCirconusCheckForceMetricActiation: &schema.Schema{
							Type:     schema.TypeBool,
							Computed: true,
						},
						agentSelfTelemetryCirconusSubmissionInterval: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryEnableHostname: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryDogStatsdAddr: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryDogStatsdTags: &schema.Schema{
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						agentSelfTelemetryStatsdAddr: {
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryStatsiteAddr: {
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfTelemetryStatsitePrefix: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			agentSelfTLSCAFile: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfTLSCertFile: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfTLSKeyFile: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfTLSMinVersion: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfTLSVerifyIncoming: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfTLSVerifyServerHostname: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfTLSVerifyOutgoing: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfTranslateWANAddrs: {
				Computed: true,
				Type:     schema.TypeBool,
			},
			agentSelfUIDir: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfUnixSockets: {
				Computed: true,
				Type:     schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						agentSelfUnixSocketUser: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfUnixSocketGroup: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						agentSelfUnixSocketMode: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			agentSelfVersion: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfVersionPrerelease: {
				Computed: true,
				Type:     schema.TypeString,
			},
			agentSelfVersionRevision: {
				Computed: true,
				Type:     schema.TypeString,
			},
			// "Watches":                nil,
		},
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

	// Pull the datacenter first because we use it when setting the ID
	var dc string
	if v, found := cfg["Datacenter"]; found {
		dc = v.(string)
	}

	const idKeyFmt = "agent-self-%s"
	d.SetId(fmt.Sprintf(idKeyFmt, dc))

	if v, found := cfg["ACLDatacenter"]; found {
		d.Set(agentSelfACLDatacenter, v.(string))
	}

	if v, found := cfg["ACLDefaultPolicy"]; found {
		d.Set(agentSelfACLDefaultPolicy, v.(string))
	}

	if v, found := cfg["ACLDisabledTTL"]; found {
		dur := time.Duration(int64(v.(float64)))
		d.Set(agentSelfACLDisabledTTL, dur.String())
	}

	if v, found := cfg["ACLDownPolicy"]; found {
		d.Set(agentSelfACLDownPolicy, v.(string))
	}

	if v, found := cfg["ACLEnforceVersion8"]; found {
		d.Set(agentSelfACLEnforceVersion8, v.(bool))
	}

	if v, found := cfg["ACLTTL"]; found {
		dur := time.Duration(int64(v.(float64)))
		d.Set(agentSelfACLTTL, dur.String())
	}

	if v, found := cfg["Addresses"]; found {
		addrs := v.(map[string]interface{})

		m := make(map[string]interface{}, len(addrs))

		if v, found := addrs["DNS"]; found {
			m[agentSelfSchemaPortsDNS] = v.(string)
		}

		if v, found := addrs["HTTP"]; found {
			m[agentSelfSchemaPortsHTTP] = v.(string)
		}

		if v, found := addrs["HTTPS"]; found {
			m[agentSelfSchemaPortsHTTPS] = v.(string)
		}

		if v, found := addrs["RPC"]; found {
			m[agentSelfSchemaPortsRPC] = v.(string)
		}

		if err := d.Set(agentSelfAddresses, m); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfAddresses), err)
		}
	}

	if v, found := cfg["AdvertiseAddr"]; found {
		d.Set(agentSelfAdvertiseAddr, v.(string))
	}

	if v, found := cfg["AdvertiseAddrs"]; found {
		addrs := v.(map[string]interface{})

		m := make(map[string]interface{}, len(addrs))

		if v, found := addrs["SerfLan"]; found && v != nil {
			m[agentSelfSchemaPortsSerfLAN] = v.(string)
		}

		if v, found := addrs["SerfWan"]; found && v != nil {
			m[agentSelfSchemaPortsSerfWAN] = v.(string)
		}

		if v, found := addrs["RPC"]; found && v != nil {
			m[agentSelfSchemaPortsRPC] = v.(string)
		}

		if err := d.Set(agentSelfAdvertiseAddrs, m); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfAdvertiseAddrs), err)
		}
	}

	if v, found := cfg["AtlasJoin"]; found {
		d.Set(agentSelfAtlasJoin, v.(bool))
	}

	if v, found := cfg["BindAddr"]; found {
		d.Set(agentSelfBindAddr, v.(string))
	}

	if v, found := cfg["Bootstrap"]; found {
		d.Set(agentSelfBootstrapMode, v.(bool))
	}

	if v, found := cfg["BootstrapExpect"]; found {
		d.Set(agentSelfBootstrapExpect, int(v.(float64)))
	}

	if v, found := cfg["CheckDeregisterIntervalMin"]; found {
		dur := time.Duration(int64(v.(float64)))
		d.Set(agentSelfCheckDeregisterIntervalMin, dur.String())
	}

	if v, found := cfg["CheckReapInterval"]; found {
		dur := time.Duration(int64(v.(float64)))
		d.Set(agentSelfCheckReapInterval, dur.String())
	}

	if v, found := cfg["CheckUpdateInterval"]; found {
		dur := time.Duration(int64(v.(float64)))
		d.Set(agentSelfCheckUpdateInterval, dur.String())
	}

	if v, found := cfg["ClientAddr"]; found {
		d.Set(agentSelfClientAddr, v.(string))
	}

	if v, found := cfg["DNS"]; found {
		dnsOpts := v.(map[string]interface{})

		m := make(map[string]interface{}, len(dnsOpts))

		if v, found := dnsOpts["AllowStale"]; found {
			m[agentSelfDNSAllowStale] = v.(bool)
		}

		if v, found := dnsOpts["DisableCompression"]; found {
			m[agentSelfDNSEnableCompression] = !v.(bool)
		}

		if v, found := dnsOpts["EnableTruncate"]; found {
			m[agentSelfDNSEnableTruncate] = v.(bool)
		}

		if v, found := dnsOpts["MaxStale"]; found {
			dur := time.Duration(int64(v.(float64)))
			m[agentSelfDNSMaxStale] = dur.String()
		}

		if v, found := dnsOpts["NodeTTL"]; found {
			dur := time.Duration(int64(v.(float64)))
			m[agentSelfDNSNodeTTL] = dur.String()
		}

		if v, found := dnsOpts["OnlyPassing"]; found {
			m[agentSelfDNSOnlyPassing] = v.(bool)
		}

		if v, found := dnsOpts["RecursorTimeout"]; found {
			dur := time.Duration(int64(v.(float64)))
			m[agentSelfDNSRecursorTimeout] = dur.String()
		}

		if v, found := dnsOpts["ServiceTTL"]; found {
			dur := time.Duration(int64(v.(float64)))
			m[agentSelfDNSServiceTTL] = dur.String()
		}

		if v, found := dnsOpts["UDPAnswerLimit"]; found {
			m[agentSelfDNSServiceTTL] = v.(int)
		}

		if err := d.Set(agentSelfDNSConfig, m); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfDNSConfig), err)
		}
	}

	{
		var l []interface{}

		if v, found := cfg["DNSRecursors"]; found {
			l = make([]interface{}, 0, len(v.([]interface{}))+1)
			l = append(l, v.([]interface{})...)
		}

		if v, found := cfg["DNSRecursor"]; found {
			l = append([]interface{}{v.(string)}, l...)
		}

		if len(l) > 0 {
			if err := d.Set(agentSelfDNSRecursors, l); err != nil {
				return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfDNSRecursors), err)
			}
		}
	}

	if v, found := cfg["DataDir"]; found {
		d.Set(agentSelfDataDir, v.(string))
	}

	if len(dc) > 0 {
		d.Set(agentSelfDatacenter, dc)
	}

	if v, found := cfg["DevMode"]; found {
		d.Set(agentSelfDevMode, v.(bool))
	}

	if v, found := cfg["DisableAnonymousSignature"]; found {
		d.Set(agentSelfEnableAnonymousSignature, !v.(bool))
	}

	if v, found := cfg["DisableCoordinates"]; found {
		d.Set(agentSelfEnableCoordinates, !v.(bool))
	}

	if v, found := cfg["DisableRemoteExec"]; found {
		d.Set(agentSelfEnableRemoteExec, !v.(bool))
	}

	if v, found := cfg["DisableUpdateCheck"]; found {
		d.Set(agentSelfEnableUpdateCheck, !v.(bool))
	}

	if v, found := cfg["Domain"]; found {
		d.Set(agentSelfDomain, v.(string))
	}

	if v, found := cfg["EnableDebug"]; found {
		d.Set(agentSelfEnableDebug, v.(bool))
	}

	if v, found := cfg["EnableSyslog"]; found {
		d.Set(agentSelfEnableSyslog, v.(bool))
	}

	if v, found := cfg["EnableUi"]; found {
		d.Set(agentSelfEnableUI, v.(bool))
	}

	if v, found := cfg["id"]; found {
		d.Set(agentSelfID, v.(string))
	}

	if v, found := cfg["SkipLeaveOnInt"]; found {
		d.Set(agentSelfLeaveOnInt, !v.(bool))
	}

	if v, found := cfg["LeaveOnTerm"]; found {
		d.Set(agentSelfLeaveOnTerm, v.(bool))
	}

	if v, found := cfg["LogLevel"]; found {
		d.Set(agentSelfLogLevel, v.(string))
	}

	if v, found := cfg["NodeName"]; found {
		d.Set(agentSelfName, v.(string))
	}

	if v, found := cfg["Performance"]; found {
		cfgs := v.(map[string]interface{})

		m := make(map[string]interface{}, len(cfgs))

		if v, found := cfgs["RaftMultiplier"]; found {
			m[agentSelfPerformanceRaftMultiplier] = strconv.FormatFloat(v.(float64), 'g', -1, 64)
		}

		if err := d.Set(agentSelfPerformance, m); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfPerformance), err)
		}
	}

	if v, found := cfg["PidFile"]; found {
		d.Set(agentSelfPidFile, v.(string))
	}

	if v, found := cfg["Ports"]; found {
		cfgs := v.(map[string]interface{})

		m := make(map[string]interface{}, len(cfgs))

		if v, found := cfgs[agentSelfAPIPortsDNS]; found {
			m[agentSelfSchemaPortsDNS] = int(v.(float64))
		}

		if v, found := cfgs[agentSelfAPIPortsHTTP]; found {
			m[agentSelfSchemaPortsHTTP] = int(v.(float64))
		}

		if v, found := cfgs[agentSelfAPIPortsHTTPS]; found {
			m[agentSelfSchemaPortsHTTPS] = int(v.(float64))
		}

		if v, found := cfgs[agentSelfAPIPortsRPC]; found {
			m[agentSelfSchemaPortsRPC] = int(v.(float64))
		}

		if v, found := cfgs[agentSelfAPIPortsSerfLAN]; found {
			m[agentSelfSchemaPortsSerfLAN] = int(v.(float64))
		}

		if v, found := cfgs[agentSelfAPIPortsSerfWAN]; found {
			m[agentSelfSchemaPortsSerfWAN] = int(v.(float64))
		}

		if v, found := cfgs[agentSelfAPIPortsServer]; found {
			m[agentSelfSchemaPortsServer] = int(v.(float64))
		}

		if err := d.Set(agentSelfPorts, m); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfPorts), err)
		}
	}

	if v, found := cfg["Protocol"]; found {
		d.Set(agentSelfProtocol, int(v.(float64)))
	}

	if v, found := cfg["ReconnectTimeoutLan"]; found {
		dur := time.Duration(int64(v.(float64)))
		d.Set(agentSelfReconnectTimeoutLAN, dur.String())
	}

	if v, found := cfg["ReconnectTimeoutWan"]; found {
		dur := time.Duration(int64(v.(float64)))
		d.Set(agentSelfReconnectTimeoutWAN, dur.String())
	}

	if v, found := cfg["RejoinAfterLeave"]; found {
		d.Set(agentSelfRejoinAfterLeave, v.(bool))
	}

	if v, found := cfg["RetryJoin"]; found {
		l := make([]string, 0, len(v.([]interface{})))
		for _, e := range v.([]interface{}) {
			l = append(l, e.(string))
		}

		if err := d.Set(agentSelfRetryJoin, l); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfRetryJoin), err)
		}
	}

	if v, found := cfg["RetryJoinEC2"]; found {
		ec2Config := v.(map[string]interface{})

		m := make(map[string]interface{}, len(ec2Config))

		if v, found := ec2Config["Region"]; found {
			m[agentSelfRetryJoinAWSRegion] = v.(string)
		}

		if v, found := ec2Config["TagKey"]; found {
			m[agentSelfRetryJoinAWSTagKey] = v.(string)
		}

		if v, found := ec2Config["TagValue"]; found {
			m[agentSelfRetryJoinAWSTagValue] = v.(string)
		}

		if err := d.Set(agentSelfRetryJoinEC2, m); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfRetryJoinEC2), err)
		}
	}

	if v, found := cfg["RetryJoinWan"]; found {
		l := make([]string, 0, len(v.([]interface{})))
		for _, e := range v.([]interface{}) {
			l = append(l, e.(string))
		}

		if err := d.Set(agentSelfRetryJoinWAN, l); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfRetryJoinWAN), err)
		}
	}

	if v, found := cfg["RetryMaxAttempts"]; found {
		d.Set(agentSelfRetryMaxAttempts, int(v.(float64)))
	}

	if v, found := cfg["RetryMaxAttemptsWan"]; found {
		d.Set(agentSelfRetryMaxAttemptsWAN, int(v.(float64)))
	}

	if v, found := cfg["SerfLanBindAddr"]; found {
		d.Set(agentSelfSerfLANBindAddr, v.(string))
	}

	if v, found := cfg["SerfWanBindAddr"]; found {
		d.Set(agentSelfSerfWANBindAddr, v.(string))
	}

	if v, found := cfg["Server"]; found {
		d.Set(agentSelfServerMode, v.(bool))
	}

	if v, found := cfg["ServerName"]; found {
		d.Set(agentSelfServerName, v.(string))
	}

	if v, found := cfg["SessionTTLMin"]; found {
		dur := time.Duration(int64(v.(float64)))
		d.Set(agentSelfSessionTTLMin, dur.String())
	}

	if v, found := cfg["StartJoin"]; found {
		serverList := v.([]interface{})
		l := make([]interface{}, 0, len(serverList))
		l = append(l, serverList...)
		if err := d.Set(agentSelfStartJoin, l); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfStartJoin), err)
		}
	}

	if v, found := cfg["StartJoinWan"]; found {
		serverList := v.([]interface{})
		l := make([]interface{}, 0, len(serverList))
		l = append(l, serverList...)
		if err := d.Set(agentSelfStartJoinWAN, l); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfStartJoinWAN), err)
		}
	}

	if v, found := cfg["SyslogFacility"]; found {
		d.Set(agentSelfSyslogFacility, v.(string))
	}

	if v, found := cfg["CAFile"]; found {
		d.Set(agentSelfTLSCAFile, v.(string))
	}

	if v, found := cfg["CertFile"]; found {
		d.Set(agentSelfTLSCertFile, v.(string))
	}

	if v, found := cfg["KeyFile"]; found {
		d.Set(agentSelfTLSKeyFile, v.(string))
	}

	if v, found := cfg["TLSMinVersion"]; found {
		d.Set(agentSelfTLSMinVersion, v.(string))
	}

	if v, found := cfg["VerifyIncoming"]; found {
		d.Set(agentSelfTLSVerifyIncoming, v.(bool))
	}

	if v, found := cfg["VerifyOutgoing"]; found {
		d.Set(agentSelfTLSVerifyOutgoing, v.(bool))
	}

	if v, found := cfg["VerifyServerHostname"]; found {
		d.Set(agentSelfTLSVerifyServerHostname, v.(bool))
	}

	if v, found := cfg["TaggedAddresses"]; found {
		addrs := v.(map[string]interface{})

		m := make(map[string]interface{}, len(addrs))

		// NOTE(sean@): agentSelfTaggedAddressesLAN and agentSelfTaggedAddressesWAN
		// are the only two known values that should be in this map at present, but
		// in the future this value could/will expand and the schema should be
		// releaxed to include both the known *{L,W}AN values as well as whatever
		// else the user specifies.
		for s, t := range addrs {
			m[s] = t
		}

		if err := d.Set(agentSelfTaggedAddresses, m); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfTaggedAddresses), err)
		}
	}

	if v, found := cfg["Telemetry"]; found {
		telemetryCfg := v.(map[string]interface{})

		m := make(map[string]interface{}, len(telemetryCfg))

		if v, found := telemetryCfg["CirconusAPIApp"]; found {
			m[agentSelfTelemetryCirconusAPIApp] = v.(string)
		}

		if v, found := telemetryCfg["CirconusAPIURL"]; found {
			m[agentSelfTelemetryCirconusAPIURL] = v.(string)
		}

		if v, found := telemetryCfg["CirconusBrokerID"]; found {
			m[agentSelfTelemetryCirconusBrokerID] = v.(string)
		}

		if v, found := telemetryCfg["CirconusBrokerSelectTag"]; found {
			m[agentSelfTelemetryCirconusBrokerSelectTag] = v.(string)
		}

		if v, found := telemetryCfg["CirconusCheckDisplayName"]; found {
			m[agentSelfTelemetryCirconusCheckDisplayName] = v.(string)
		}

		if v, found := telemetryCfg["CirconusCheckID"]; found {
			m[agentSelfTelemetryCirconusCheckID] = v.(string)
		}

		if v, found := telemetryCfg["CirconusCheckInstanceID"]; found {
			m[agentSelfTelemetryCirconusCheckInstanceID] = v.(string)
		}

		if v, found := telemetryCfg["CirconusCheckSearchTag"]; found {
			m[agentSelfTelemetryCirconusCheckSearchTag] = v.(string)
		}

		if v, found := telemetryCfg["CirconusCheckSubmissionURL"]; found {
			m[agentSelfTelemetryCirconusCheckSubmissionURL] = v.(string)
		}

		if v, found := telemetryCfg["CirconusCheckTags"]; found {
			m[agentSelfTelemetryCirconusCheckTags] = v.(string)
		}

		if v, found := telemetryCfg["CirconusCheckForceMetricActivation"]; found {
			m[agentSelfTelemetryCirconusCheckForceMetricActiation] = v.(string)
		}

		if v, found := telemetryCfg["CirconusSubmissionInterval"]; found {
			m[agentSelfTelemetryCirconusSubmissionInterval] = v.(string)
		}

		if v, found := telemetryCfg["DisableHostname"]; found {
			m[agentSelfTelemetryEnableHostname] = fmt.Sprintf("%t", !v.(bool))
		}

		if v, found := telemetryCfg["DogStatsdAddr"]; found {
			m[agentSelfTelemetryDogStatsdAddr] = v.(string)
		}

		if v, found := telemetryCfg["DogStatsdTags"]; found && v != nil {
			m[agentSelfTelemetryDogStatsdTags] = append([]interface{}(nil), v.([]interface{})...)
		}

		if v, found := telemetryCfg["StatsdAddr"]; found {
			m[agentSelfTelemetryStatsdAddr] = v.(string)
		}

		if v, found := telemetryCfg["StatsiteAddr"]; found {
			m[agentSelfTelemetryStatsiteAddr] = v.(string)
		}

		if v, found := telemetryCfg["StatsitePrefix"]; found {
			m[agentSelfTelemetryStatsitePrefix] = v.(string)
		}

		if err := d.Set(agentSelfTelemetry, m); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfTelemetry), err)
		}
	}

	if v, found := cfg["TranslateWanTelemetryCfg"]; found {
		d.Set(agentSelfTranslateWANAddrs, v.(bool))
	}

	if v, found := cfg["UiDir"]; found {
		d.Set(agentSelfUIDir, v.(string))
	}

	if v, found := cfg["UnixSockets"]; found {
		socketConfig := v.(map[string]interface{})

		m := make(map[string]interface{}, len(socketConfig))

		if v, found := socketConfig["Grp"]; found {
			m[agentSelfUnixSocketGroup] = v.(string)
		}

		if v, found := socketConfig["Mode"]; found {
			m[agentSelfUnixSocketMode] = v.(string)
		}

		if v, found := socketConfig["Usr"]; found {
			m[agentSelfUnixSocketUser] = v.(string)
		}

		if err := d.Set(agentSelfUnixSockets, m); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Unable to set %s: {{err}}", agentSelfUnixSockets), err)
		}
	}

	if v, found := cfg["Version"]; found {
		d.Set(agentSelfVersion, v.(string))
	}

	if v, found := cfg["VersionPrerelease"]; found {
		d.Set(agentSelfVersionPrerelease, v.(string))
	}

	if v, found := cfg["VersionPrerelease"]; found {
		d.Set(agentSelfVersionPrerelease, v.(string))
	}

	if v, found := cfg["Revision"]; found {
		d.Set(agentSelfVersionRevision, v.(string))
	}

	return nil
}
