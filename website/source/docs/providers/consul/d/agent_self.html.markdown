---
layout: "consul"
page_title: "Consul: consul_agent_self"
sidebar_current: "docs-consul-data-source-agent-self"
description: |-
  Provides the configuration information of the local Consul agent.
---

# consul_agent__self

The `consul_agent_self` data source returns
[configuration and status data](https://www.consul.io/docs/agent/http/agent.html#agent_self)
from the agent specified in the `provider`.

## Example Usage

```hcl
data "consul_agent_self" "read-dc1-agent" {
  query_options {
    # Optional parameter: implicitly uses the current datacenter of the agent
    datacenter = "dc1"
  }
}

# Set the description to a whitespace delimited list of the services
resource "example_resource" "app" {
  description = "Consul datacenter ${data.consul_agent_self.read-dc1-agent.datacenter}"

  # ...
}
```

## Attributes Reference

The following attributes are exported:

* [`acl_datacenter`](https://www.consul.io/docs/agent/options.html#acl_datacenter)
* [`acl_default_policy`](https://www.consul.io/docs/agent/options.html#acl_default_policy)
* `acl_disabled_ttl`
* [`acl_down_policy`](https://www.consul.io/docs/agent/options.html#acl_down_policy)
* [`acl_enforce_0_8_semantics`](https://www.consul.io/docs/agent/options.html#acl_enforce_version_8)
* [`acl_ttl`](https://www.consul.io/docs/agent/options.html#acl_ttl)
* [`addresses`](https://www.consul.io/docs/agent/options.html#addresses)
* [`advertise_addr`](https://www.consul.io/docs/agent/options.html#_advertise)
* [`advertise_addr_wan`](https://www.consul.io/docs/agent/options.html#_advertise-wan)
* [`advertise_addrs`](https://www.consul.io/docs/agent/options.html#advertise_addrs)
* [`atlas_join`](https://www.consul.io/docs/agent/options.html#_atlas_join)
* [`bind_addr`](https://www.consul.io/docs/agent/options.html#_bind)
* [`bootstrap_expect`](https://www.consul.io/docs/agent/options.html#_bootstrap_expect)
* [`bootstrap_mode`](https://www.consul.io/docs/agent/options.html#_bootstrap)
* `check_deregister_interval_min`
* `check_reap_interval`
* [`check_update_interval`](https://www.consul.io/docs/agent/options.html#check_update_interval)
* [`client_addr`](https://www.consul.io/docs/agent/options.html#_client)
* `dns` - A map of DNS configuration attributes.  See below for details on the
  contents of the `dns` attribute.
* [`dns_recursors`](https://www.consul.io/docs/agent/options.html#recursors) - A
  list of all DNS recursors.
* [`data_dir`](https://www.consul.io/docs/agent/options.html#_data_dir)
* [`datacenter`](https://www.consul.io/docs/agent/options.html#_datacenter)
* [`dev_mode`](https://www.consul.io/docs/agent/options.html#_dev)
* [`domain`](https://www.consul.io/docs/agent/options.html#_domain)
* [`enable_anonymous_signature`](https://www.consul.io/docs/agent/options.html#disable_anonymous_signature)
* `enable_coordinates`
* [`enable_debug`](https://www.consul.io/docs/agent/options.html#enable_debug)
* [`enable_remote_exec`](https://www.consul.io/docs/agent/options.html#disable_remote_exec)
* [`enable_syslog`](https://www.consul.io/docs/agent/options.html#_syslog)
* [`enable_ui`](https://www.consul.io/docs/agent/options.html#_ui)
* [`enable_update_check`](https://www.consul.io/docs/agent/options.html#disable_update_check)
* [`id`](https://www.consul.io/docs/agent/options.html#_node_id)
* [`leave_on_int`](https://www.consul.io/docs/agent/options.html#skip_leave_on_interrupt)
* [`leave_on_term`](https://www.consul.io/docs/agent/options.html#leave_on_terminate)
* [`log_level`](https://www.consul.io/docs/agent/options.html#_log_level)
* [`name`](https://www.consul.io/docs/agent/options.html#_node)
* [`performance`](https://www.consul.io/docs/agent/options.html#performance)
* [`pid_file`](https://www.consul.io/docs/agent/options.html#_pid_file)
* [`ports`](https://www.consul.io/docs/agent/options.html#ports)
* [`protocol_version`](https://www.consul.io/docs/agent/options.html#_protocol)
* [`reconnect_timeout_lan`](https://www.consul.io/docs/agent/options.html#reconnect_timeout)
* [`reconnect_timeout_wan`](https://www.consul.io/docs/agent/options.html#reconnect_timeout_wan)
* [`rejoin_after_leave`](https://www.consul.io/docs/agent/options.html#_rejoin)
* [`retry_join`](https://www.consul.io/docs/agent/options.html#retry_join)
* [`retry_join_ec2`](https://www.consul.io/docs/agent/options.html#retry_join_ec2) -
  A map of EC2 retry attributes.  See below for details on the available
  information.
* [`retry_join_gce`](https://www.consul.io/docs/agent/options.html#retry_join_gce) -
  A map of GCE retry attributes.  See below for details on the available
  information.
* [`retry_join_wan`](https://www.consul.io/docs/agent/options.html#_retry_join_wan)
* [`retry_max_attempts`](https://www.consul.io/docs/agent/options.html#_retry_max)
* [`retry_max_attempts_wan`](https://www.consul.io/docs/agent/options.html#_retry_max_wan)
* [`serf_lan_bind_addr`](https://www.consul.io/docs/agent/options.html#_serf_lan_bind)
* [`serf_wan_bind_addr`](https://www.consul.io/docs/agent/options.html#_serf_wan_bind)
* [`server_mode`](https://www.consul.io/docs/agent/options.html#_server)
* [`server_name`](https://www.consul.io/docs/agent/options.html#server_name)
* [`session_ttl_min`](https://www.consul.io/docs/agent/options.html#session_ttl_min)
* [`start_join`](https://www.consul.io/docs/agent/options.html#start_join)
* [`start_join_wan`](https://www.consul.io/docs/agent/options.html#start_join_wan)
* [`syslog_facility`](https://www.consul.io/docs/agent/options.html#syslog_facility)
* [`tls_ca_file`](https://www.consul.io/docs/agent/options.html#ca_file)
* [`tls_cert_file`](https://www.consul.io/docs/agent/options.html#cert_file)
* [`tls_key_file`](https://www.consul.io/docs/agent/options.html#key_file)
* [`tls_min_version`](https://www.consul.io/docs/agent/options.html#tls_min_version)
* [`tls_verify_incoming`](https://www.consul.io/docs/agent/options.html#verify_incoming)
* [`tls_verify_outgoing`](https://www.consul.io/docs/agent/options.html#verify_outgoing)
* [`tls_verify_server_hostname`](https://www.consul.io/docs/agent/options.html#verify_server_hostname)
* [`tagged_addresses`](https://www.consul.io/docs/agent/options.html#translate_wan_addrs)
* [`telemetry`](https://www.consul.io/docs/agent/options.html#telemetry) - A map
  of telemetry configuration.
* [`translate_wan_addrs`](https://www.consul.io/docs/agent/options.html#translate_wan_addrs)
* [`ui_dir`](https://www.consul.io/docs/agent/options.html#ui_dir)
* [`unix_sockets`](https://www.consul.io/docs/agent/options.html#unix_sockets)
* `version` - The version of the Consul agent.
* `version_prerelease`
* `version_revision`

### DNS Attributes

* [`allow_stale`](https://www.consul.io/docs/agent/options.html#allow_stale)
* [`enable_compression`](https://www.consul.io/docs/agent/options.html#disable_compression)
* [`enable_truncate`](https://www.consul.io/docs/agent/options.html#enable_truncate)
* [`max_stale`](https://www.consul.io/docs/agent/options.html#max_stale)
* [`node_ttl`](https://www.consul.io/docs/agent/options.html#node_ttl)
* [`only_passing`](https://www.consul.io/docs/agent/options.html#only_passing)
* [`recursor_timeout`](https://www.consul.io/docs/agent/options.html#recursor_timeout)
* [`service_ttl`](https://www.consul.io/docs/agent/options.html#service_ttl)
* [`udp_answer_limit`](https://www.consul.io/docs/agent/options.html#udp_answer_limit)

### Retry Join EC2 Attributes

* [`region`](https://www.consul.io/docs/agent/options.html#region)
* [`tag_key`](https://www.consul.io/docs/agent/options.html#tag_key)
* [`tag_value`](https://www.consul.io/docs/agent/options.html#tag_value)

### Retry Join GCE Attributes

* [`credentials_file`](https://www.consul.io/docs/agent/options.html#credentials_file)
* [`project_name`](https://www.consul.io/docs/agent/options.html#project_name)
* [`tag_value`](https://www.consul.io/docs/agent/options.html#tag_value)
* [`zone_pattern`](https://www.consul.io/docs/agent/options.html#zone_pattern)

### Telemetry Attributes

* [`circonus_api_app`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_api_app)
* [`circonus_api_token`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_api_token)
* [`circonus_api_url`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_api_url)
* [`circonus_broker_id`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_broker_id)
* [`circonus_check_id`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_check_id)
* [`circonus_check_tags`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_check_tags)
* [`circonus_display_name`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_check_display_name)
* [`circonus_force_metric_activation`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_check_force_metric_activation)
* [`circonus_instance_id`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_check_instance_id)
* [`circonus_search_tag`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_check_search_tag)
* [`circonus_select_tag`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_broker_select_tag)
* [`circonus_submission_interval`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_submission_interval)
* [`circonus_submission_url`](https://www.consul.io/docs/agent/options.html#telemetry-circonus_submission_url)
* [`dogstatsd_addr`](https://www.consul.io/docs/agent/options.html#telemetry-dogstatsd_addr)
* [`dogstatsd_tags`](https://www.consul.io/docs/agent/options.html#telemetry-dogstatsd_tags)
* [`enable_hostname`](https://www.consul.io/docs/agent/options.html#telemetry-disable_hostname)
* [`statsd_addr`](https://www.consul.io/docs/agent/options.html#telemetry-statsd_address)
* [`statsite_addr`](https://www.consul.io/docs/agent/options.html#telemetry-statsite_address)
* [`statsite_prefix`](https://www.consul.io/docs/agent/options.html#telemetry-statsite_prefix)
