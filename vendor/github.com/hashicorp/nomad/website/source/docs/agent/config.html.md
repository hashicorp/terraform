---
layout: "docs"
page_title: "Configuration"
sidebar_current: "docs-agent-config"
description: |-
  Learn about the configuration options available for the Nomad agent.
---

# Configuration

Nomad agents have a variety of options that can be specified via configuration
files or command-line flags. Config files must be written in
[HCL](https://github.com/hashicorp/hcl) or JSON syntax. Nomad can read and
combine options from multiple configuration files or directories to configure
the Nomad agent.

## Loading Configuration Files

When specifying multiple config file options on the command-line, the files are
loaded in the order they are specified. For example:

    nomad agent -config server.conf -config /etc/nomad -config extra.json

Will load configuration from `server.conf`, from `.hcl` and `.json` files under
`/etc/nomad`, and finally from `extra.json`.

Configuration files in directories are loaded alphabetically. With the
directory option, only files ending with the `.hcl` or `.json` extensions are
used. Directories are not loaded recursively.

As each file is processed, its contents are merged into the existing
configuration. When merging, any non-empty values from the latest config file
will append or replace options in the current configuration. An empty value
means `""` for strings, `0` for integer or float values, and `false` for
booleans. Since empty values are ignored you cannot disable an option like
server mode once you've enabled it.

Complex data types like arrays or maps are usually merged. [Some configuration
options](#cli) can also be specified using the command-line interface. Please
refer to the sections below for the details of each option.

## Configuration Syntax

The preferred configuration syntax is HCL, which supports comments, but you can
also use JSON. Below is an example configuration file in HCL syntax.

```
bind_addr = "0.0.0.0"
data_dir = "/var/lib/nomad"

advertise {
  # We need to specify our host's IP because we can't
  # advertise 0.0.0.0 to other nodes in our cluster.
  rpc = "1.2.3.4:4647"
}

server {
  enabled = true
  bootstrap_expect = 3
}

client {
  enabled = true
  network_speed = 10
  options {
    "driver.raw_exec.enable" = "1"
  }
}

consul {
    # Consul's HTTP Address
    address = "1.2.3.4:8500"
}

atlas {
  infrastructure = "hashicorp/mars"
  token = "atlas.v1.AFE84330943"
}
```

Note that it is strongly recommended _not_ to operate a node as both `client`
and `server`, although this is supported to simplify development and testing.

## General Options

The following configuration options are available to both client and server
nodes, unless otherwise specified:

* <a id="region">`region`</a>: Specifies the region the Nomad agent is a
  member of. A region typically maps to a geographic region, for example `us`,
  with potentially multiple zones, which map to [datacenters](#datacenter) such
  as `us-west` and `us-east`. Defaults to `global`.

* `datacenter`: Datacenter of the local agent. All members of a datacenter
  should share a local LAN connection. Defaults to `dc1`.

* <a id="name">`name`</a>: The name of the local node. This value is used to
  identify individual nodes in a given datacenter and must be unique
  per-datacenter. By default this is set to the local host's name.

* `data_dir`: A local directory used to store agent state. Client nodes use this
  directory by default to store temporary allocation data as well as cluster
  information. Server nodes use this directory to store cluster state, including
  the replicated log and snapshot data. This option is required to start the
  Nomad agent and must be specified as an absolute path.

* `log_level`: Controls the verbosity of logs the Nomad agent will output. Valid
  log levels include `WARN`, `INFO`, or `DEBUG` in increasing order of
  verbosity. Defaults to `INFO`.

* <a id="bind_addr">`bind_addr`</a>: Used to indicate which address the Nomad
  agent should bind to for network services, including the HTTP interface as
  well as the internal gossip protocol and RPC mechanism. This should be
  specified in IP format, and can be used to easily bind all network services to
  the same address. It is also possible to bind the individual services to
  different addresses using the [addresses](#addresses) configuration option.
  Defaults to the local loopback address `127.0.0.1`.

* `enable_debug`: Enables the debugging HTTP endpoints. These endpoints can be
  used with profiling tools to dump diagnostic information about Nomad's
  internals. It is not recommended to leave this enabled in production
  environments. Defaults to `false`.

* `ports`: Controls the network ports used for different services required by
  the Nomad agent. The value is a key/value mapping of port numbers, and accepts
  the following keys:
  <br>
  * `http`: The port used to run the HTTP server. Applies to both client and
    server nodes. Defaults to `4646`.
  * `rpc`: The port used for internal RPC communication between agents and
    servers, and for inter-server traffic for the consensus algorithm (raft).
    Defaults to `4647`. Only used on server nodes.
  * `serf`: The port used for the gossip protocol for cluster membership. Both
    TCP and UDP should be routable between the server nodes on this port.
    Defaults to `4648`. Only used on server nodes.

* <a id="addresses">`addresses`</a>: Controls the bind address for individual
  network services. Any values configured in this block take precedence over the
  default [bind_addr](#bind_addr). The value is a map of IP addresses and
  supports the following keys:
  <br>
  * `http`: The address the HTTP server is bound to. This is the most common
    bind address to change. Applies to both clients and servers.
  * `rpc`: The address to bind the internal RPC interfaces to. Should be exposed
    only to other cluster members if possible. Used only on server nodes, but
    must be accessible from all agents.
  * `serf`: The address used to bind the gossip layer to. Both a TCP and UDP
    listener will be exposed on this address. Should be restricted to only
    server nodes from the same datacenter if possible. Used only on server
    nodes.

* `advertise`: Controls the advertise address for individual network services.
  This can be used to advertise a different address to the peers of a server or
  a client node to support more complex network configurations such as NAT. This
  configuration is optional, and defaults to the bind address of the specific
  network service if it is not provided. The value is a map of IP addresses and
  ports and supports the following keys:
  <br>
  * `http`: The address to advertise for the HTTP interface. This should be
    reachable by all the nodes from which end users are going to use the Nomad
    CLI tools.
    ```
    advertise {
       http = "1.2.3.4:4646"
    }
    ```
  * `rpc`: The address to advertise for the RPC interface. This address should
    be reachable by all of the agents in the cluster. For example:
    ```
    advertise {
      rpc = "1.2.3.4:4647"
    }
    ```
  * `serf`: The address advertised for the gossip layer. This address must be
    reachable from all server nodes. It is not required that clients can reach
    this address.

* `consul`: The `consul` configuration block changes how Nomad interacts with
  Consul. Nomad can automatically advertise Nomad services via Consul, and can
  automatically bootstrap itself using Consul. For more details see the [`consul`
  section](#consul_options).

<a id="telemetry_config"></a>

* `telemetry`: Used to control how the Nomad agent exposes telemetry data to
  external metrics collection servers. This is a key/value mapping and supports
  the following keys:
  <br>
  * `statsite_address`: Address of a
    [statsite](https://github.com/armon/statsite) server to forward metrics data
    to.
  * `statsd_address`: Address of a [statsd](https://github.com/etsy/statsd)
    server to forward metrics to.
  * `disable_hostname`: A boolean indicating if gauge values should not be
    prefixed with the local hostname.
  * `circonus_api_token`
    A valid [Circonus](http://circonus.com/) API Token used to create/manage check. If provided, metric management is enabled.
  * `circonus_api_app`
    A valid app name associated with the API token. By default, this is set to "consul".
  * `circonus_api_url`
    The base URL to use for contacting the Circonus API. By default, this is set to "https://api.circonus.com/v2".
  * `circonus_submission_interval`
    The interval at which metrics are submitted to Circonus. By default, this is set to "10s" (ten seconds).
  * `circonus_submission_url`
    The `check.config.submission_url` field, of a Check API object, from a previously created HTTPTRAP check.
  * `circonus_check_id`
    The Check ID (not **check bundle**) from a previously created HTTPTRAP check. The numeric portion of the `check._cid` field in the Check API object.
  * `circonus_check_force_metric_activation`
    Force activation of metrics which already exist and are not currently active. If check management is enabled, the default behavior is to add new metrics as they are encountered. If the metric already exists in the check, it will **not** be activated. This setting overrides that behavior. By default, this is set to "false".
  * `circonus_check_instance_id`
    Serves to uniquely identify the metrics coming from this *instance*.  It can be used to maintain metric continuity with transient or ephemeral instances as they move around within an infrastructure. By default, this is set to hostname:application name (e.g. "host123:consul").
  * `circonus_check_search_tag`
    A special tag which, when coupled with the instance id, helps to narrow down the search results when neither a Submission URL or Check ID is provided. By default, this is set to service:app (e.g. "service:consul").
  * `circonus_broker_id`
    The ID of a specific Circonus Broker to use when creating a new check. The numeric portion of `broker._cid` field in a Broker API object. If metric management is enabled and neither a Submission URL nor Check ID is provided, an attempt will be made to search for an existing check using Instance ID and Search Tag. If one is not found, a new HTTPTRAP check will be created. By default, this is not used and a random Enterprise Broker is selected, or, the default Circonus Public Broker.
  * `circonus_broker_select_tag`
    A special tag which will be used to select a Circonus Broker when a Broker ID is not provided. The best use of this is to as a hint for which broker should be used based on *where* this particular instance is running (e.g. a specific geo location or datacenter, dc:sfo). By default, this is not used.

* `leave_on_interrupt`: Enables gracefully leaving when receiving the
  interrupt signal. By default, the agent will exit forcefully on any signal.

* `leave_on_terminate`: Enables gracefully leaving when receiving the
  terminate signal. By default, the agent will exit forcefully on any signal.

* `enable_syslog`: Enables logging to syslog. This option only works on
  Unix based systems.

* `syslog_facility`: Controls the syslog facility that is used. By default,
  `LOCAL0` will be used. This should be used with `enable_syslog`.

* `disable_update_check`: Disables automatic checking for security bulletins
  and new version releases.

* `disable_anonymous_signature`: Disables providing an anonymous signature
  for de-duplication with the update check. See `disable_update_check`.

* `http_api_response_headers`: This object allows adding headers to the
  HTTP API responses. For example, the following config can be used to enable
  CORS on the HTTP API endpoints:
  ```
  http_api_response_headers {
      Access-Control-Allow-Origin = "*"
  }
  ```

* `atlas`: See the [`atlas` options](#atlas_options) for more details.

## <a id="consul_options"></a>Consul Options

The following options are used to configure [Consul](https://www.consul.io)
integration and are entirely optional.

* `consul`: The top-level config key used to contain all Consul-related
  configuration options. The value is a key/value map which supports the
  following keys:
  <br>
  * `address`: The address to the local Consul agent given in the format of
    `host:port`. Defaults to `127.0.0.1:8500`, which is the same as the Consul
    default HTTP address.

  * `token`: Token is used to provide a per-request ACL token. This options
    overrides the Consul Agent's default token.

  * `auth`: The auth information to use for http access to the Consul Agent
    given as `username:password`.

  * `ssl`: This boolean option sets the transport scheme to talk to the Consul
    Agent as `https`. Defaults to `false`.

  * `verify_ssl`: This option enables SSL verification when the transport
    scheme for the Consul API client is `https`. Defaults to `true`.

  * `ca_file`: Optional path to the CA certificate used for Consul
    communication, defaults to the system bundle if not specified.

  * `cert_file`: The path to the certificate used for Consul communication. If
    this is set then you need to also set `key_file`.

  * `key_file`: The path to the private key used for Consul communication. If
    this is set then you need to also set `cert_file`.

  * `server_service_name`: The name of the service that Nomad registers servers
    with. Defaults to `nomad`.

  * `client_service_name`: The name of the service that Nomad registers clients
    with. Defaults to `nomad-client`.

  * `auto_advertise`: When enabled Nomad advertises its services to Consul. The
    services are named according to `server_service_name` and
    `client_service_name`. Nomad Servers and Clients advertise their respective
    services, each tagged appropriately with either `http` or `rpc` tag. Nomad
    Servers also advertise a `serf` tagged service.  Defaults to `true`.  

  * `server_auto_join`: Servers will automatically discover and join other
    Nomad Servers by searching for the Consul service name defined in the
    `server_service_name` option. This search only happens if the Server does
    not have a leader. Defaults to `true`.

  * `client_auto_join`:  Client will automatically discover Servers in the
    Client's region by searching for the Consul service name defined in the
    `server_service_name` option. The search occurs if the Client is not
    registered with any Servers or it is unable to heartbeat to the leader of
    the region, in which case it may be partitioned and searches for other
    Servers. Defaults to `true`

When `server_auto_join`, `client_auto_join` and `auto_advertise` are all
enabled, which is by default, and Consul is available, the Nomad cluster will
self-bootstrap.

## <a id="atlas_options"></a>Atlas Options

**NOTE**: Nomad integration with Atlas is awaiting release of Atlas features
for Nomad support.  Nomad currently only validates configuration options for
Atlas but does not use them.
See [#183](https://github.com/hashicorp/nomad/issues/183) for more details.

The following options are used to configure [Atlas](https://atlas.hashicorp.com)
integration and are entirely optional.

* `atlas`: The top-level config key used to contain all Atlas-related
  configuration options. The value is a key/value map which supports the
  following keys:
  <br>
  * <a id="infrastructure">`infrastructure`</a>: The Atlas infrastructure name to
    connect this agent to. This value should be of the form
    `<org>/<infrastructure>`, and requires a valid [token](#token) authorized on
    the infrastructure.
  * <a id="token">`token`</a>: The Atlas token to use for authentication. This
    token should have access to the provided [infrastructure](#infrastructure).
  * <a id="join">`join`</a>: A boolean indicating if the auto-join feature of
    Atlas should be enabled. Defaults to `false`.
  * `endpoint`: The address of the Atlas instance to connect to. Defaults to the
    public Atlas endpoint and is only used if both
    [infrastructure](#infrastructure) and [token](#token) are provided.


## Server-specific Options

The following options are applicable to server agents only and need not be
configured on client nodes.

* `server`: This is the top-level key used to define the Nomad server
  configuration. It is a key/value mapping which supports the following keys:
  <br>
  * `enabled`: A boolean indicating if server mode should be enabled for the
    local agent. All other server options depend on this value being set.
    Defaults to `false`.
  * <a id="bootstrap_expect">`bootstrap_expect`</a>: This is an integer
    representing the number of server nodes to wait for before bootstrapping. It
    is most common to use the odd-numbered integers `3` or `5` for this value,
    depending on the cluster size. A value of `1` does not provide any fault
    tolerance and is not recommended for production use cases.
  * `data_dir`: This is the data directory used for server-specific data,
    including the replicated log. By default, this directory lives inside of the
    [data_dir](#data_dir) in the "server" sub-path.
  * `protocol_version`: The Nomad protocol version spoken when communicating
    with other Nomad servers. This value is typically not required as the agent
    internally knows the latest version, but may be useful in some upgrade
    scenarios.
  * `num_schedulers`: The number of parallel scheduler threads to run. This
    can be as many as one per core, or `0` to disallow this server from making
    any scheduling decisions. This defaults to the number of CPU cores.
  * `enabled_schedulers`: This is an array of strings indicating which
    sub-schedulers this server will handle. This can be used to restrict the
    evaluations that worker threads will dequeue for processing. This
    defaults to all available schedulers.
  * `node_gc_threshold` This is a string with a unit suffix, such as "300ms",
    "1.5h" or "25m". Valid time units are "ns", "us" (or "µs"), "ms", "s",
    "m", "h". Controls how long a node must be in a terminal state before it is
    garbage collected and purged from the system.
  * <a id="rejoin_after_leave">`rejoin_after_leave`</a> When provided, Nomad will ignore a previous leave and
    attempt to rejoin the cluster when starting. By default, Nomad treats leave
    as a permanent intent and does not attempt to join the cluster again when
    starting. This flag allows the previous state to be used to rejoin the
    cluster.
  * <a id="retry_join">`retry_join`</a> Similar to [`start_join`](#start_join) but allows retrying a join
    if the first attempt fails. This is useful for cases where we know the
    address will become available eventually. Use `retry_join` with an array as a replacement for
    `start_join`, do not use both options.
  * <a id="retry_interval">`retry_interval`</a> The time to wait between join attempts. Defaults to 30s.
  * <a id="retry_max">`retry_max`</a> The maximum number of join attempts to be made before exiting
    with a return code of 1. By default, this is set to 0 which is interpreted
    as infinite retries.
  * <a id="start_join">`start_join`</a> An array of strings specifying addresses
    of nodes to join upon startup. If Nomad is unable to join with any of the
    specified addresses, agent startup will fail. By default, the agent won't
    join any nodes when it starts up. Addresses can be given as an IP, a domain
    name, or an IP:Port pair. If the port isn't specified the default Serf port,
    4648, is used.  DNS names may also be used.

## Client-specific Options

The following options are applicable to client agents only and need not be
configured on server nodes.

* `client`: This is the top-level key used to define the Nomad client
  configuration. Like the server configuration, it is a key/value mapping which
  supports the following keys:
  <br>
  * `enabled`: A boolean indicating if client mode is enabled. All other client
    configuration options depend on this value. Defaults to `false`.
  * <a id="state_dir">`state_dir`</a>: This is the state dir used to store
    client state. By default, it lives inside of the [data_dir](#data_dir), in
    the "client" sub-path. It must be specified as an absolute path.
  * <a id="alloc_dir">`alloc_dir`</a>: A directory used to store allocation data.
    Depending on the workload, the size of this directory can grow arbitrarily
    large as it is used to store downloaded artifacts for drivers (QEMU images,
    JAR files, etc.). It is therefore important to ensure this directory is
    placed some place on the filesystem with adequate storage capacity. By
    default, this directory lives under the [data_dir](#data_dir) at the
    "alloc" sub-path. It must be specified as an absolute path.
  * <a id="servers">`servers`</a>: An array of server addresses. This list is
    used to register the client with the server nodes and advertise the
    available resources so that the agent can receive work. If a port is not specified
    in the array of server addresses, the default port `4647` will be used.
  * <a id="node_class">`node_class`</a>: A string used to logically group client
    nodes by class. This can be used during job placement as a filter. This
    option is not required and has no default.
  * <a id="meta">`meta`</a>: This is a key/value mapping of metadata pairs. This
    is a free-form map and can contain any string values.
  * <a id="options">`options`</a>: This is a key/value mapping of internal
    configuration for clients, such as for driver configuration. Please see
    [here](#options_map) for a description of available options.
  * <a id="chroot_env">`chroot_env`</a>: This is a key/value mapping that
    defines the chroot environment for jobs using the Exec and Java drivers.
    Please see [here](#chroot_env_map) for an example and further information.
  * <a id="network_interface">`network_interface`</a>: This is a string to force
    network fingerprinting to use a specific network interface
  * <a id="network_speed">`network_speed`</a>: This is an int that sets the
    default link speed of network interfaces, in megabits, if their speed can
    not be determined dynamically.
  * `max_kill_timeout`: `max_kill_timeout` is a time duration that can be
    specified using the `s`, `m`, and `h` suffixes, such as `30s`. If a job's
    task specifies a `kill_timeout` greater than `max_kill_timeout`,
    `max_kill_timeout` is used. This is to prevent a user being able to set an
    unreasonable timeout. If unset, a default is used.
<a id="reserved"></a>
  * `reserved`: `reserved` is used to reserve a portion of the nodes resources
    from being used by Nomad when placing tasks.  It can be used to target
    a certain capacity usage for the node. For example, 20% of the nodes CPU
    could be reserved to target a CPU utilization of 80%. The block has the
    following format:

    ```
    reserved {
        cpu = 500
        memory = 512
        disk = 1024
        reserved_ports = "22,80,8500-8600"
    }
    ```

    * `cpu`: `cpu` is given as MHz to reserve.
    * `memory`: `memory` is given as MB to reserve.
    * `disk`: `disk` is given as MB to reserve.
    * `reserved_ports`: `reserved_ports` is a comma separated list of ports
      to reserve on all fingerprinted network devices. Ranges can be
      specified by using a hyphen separated the two inclusive ends.

### <a id="options_map"></a>Client Options Map

The following is not an exhaustive list of options that can be passed to the
Client, but rather the set of options that configure the Client and not the
drivers. To find the options supported by an individual driver, see the drivers
documentation [here](/docs/drivers/index.html)

* `driver.whitelist`: A comma separated list of whitelisted drivers (e.g.
  "docker,qemu"). If specified, drivers not in the whitelist will be disabled.
  If the whitelist is empty, all drivers are fingerprinted and enabled where
  applicable.

*   `env.blacklist`: Nomad passes the host environment variables to `exec`,
    `raw_exec` and `java` tasks. `env.blacklist` is a comma separated list of
    environment variable keys not to pass to these tasks. If specified, the
    defaults are overridden. The following are the default:

    * `CONSUL_TOKEN`
    * `VAULT_TOKEN`
    * `ATLAS_TOKEN`
    * `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
    * `GOOGLE_APPLICATION_CREDENTIALS`

*   `user.blacklist`: An operator specifiable blacklist of users which a task is
    not allowed to run as when using a driver in `user.checked_drivers`.
    Defaults to:

    * `root`
    * `Administrator`

*   `user.checked_drivers`: An operator specifiable list of drivers to enforce
    the the `user.blacklist`. For drivers using containers, this enforcement often
    doesn't make sense and as such the default is set to:

    * `exec`
    * `qemu`
    * `java`

* `fingerprint.whitelist`: A comma separated list of whitelisted fingerprinters.
  If specified, fingerprinters not in the whitelist will be disabled. If the
  whitelist is empty, all fingerprinters are used.

### <a id="chroot_env_map"></a>Client ChrootEnv Map

Drivers based on [Isolated Fork/Exec](/docs/drivers/exec.html) implement file
system isolation using chroot on Linux.  The `chroot_env` map allows the chroot
environment to be configured using source paths on the host operating system.
The mapping format is: `source_path -> dest_path`.

The following example specifies a chroot which contains just enough to run the
`ls` utility, and not much else:

```
chroot_env {
    "/bin/ls" = "/bin/ls"
    "/etc/ld.so.cache" = "/etc/ld.so.cache"
    "/etc/ld.so.conf" = "/etc/ld.so.conf"
    "/etc/ld.so.conf.d" = "/etc/ld.so.conf.d"
    "/lib" = "/lib"
    "/lib64" = "/lib64"
}
```

When `chroot_env` is unspecified, the `exec` driver will use a default chroot
environment with the most commonly used parts of the operating system. See
`exec` documentation for the full list [here](/docs/drivers/exec.html#chroot).

## <a id="cli"></a>Command-line Options

A subset of the available Nomad agent configuration can optionally be passed in
via CLI arguments. The `agent` command accepts the following arguments:

* `alloc-dir=<path>`: Equivalent to the Client [alloc_dir](#alloc_dir) config
   option.
* `-atlas=<infrastructure>`: Equivalent to the Atlas
  [infrastructure](#infrastructure) config option.
* `-atlas-join`: Equivalent to the Atlas [join](#join) config option.
* `-atlas-token=<token>`: Equivalent to the Atlas [token](#token) config option.
* `-bind=<address>`: Equivalent to the [bind_addr](#bind_addr) config option.
* `-bootstrap-expect=<num>`: Equivalent to the
  [bootstrap_expect](#bootstrap_expect) config option.
* `-client`: Enable client mode on the local agent.
* `-config=<path>`: Specifies the path to a configuration file or a directory of
  configuration files to load. Can be specified multiple times.
* `-data-dir=<path>`: Equivalent to the [data_dir](#data_dir) config option.
* `-dc=<datacenter>`: Equivalent to the [datacenter](#datacenter) config option.
* `-dev`: Start the agent in development mode. This enables a pre-configured
  dual-role agent (client + server) which is useful for developing or testing
  Nomad. No other configuration is required to start the agent in this mode.
* `-join=<address>`: Address of another agent to join upon starting up. This can
  be specified multiple times to specify multiple agents to join.
* `-log-level=<level>`: Equivalent to the [log_level](#log_level) config option.
* `-meta=<key=value>`: Equivalent to the Client [meta](#meta) config option.
* `-network-interface<interface>`: Equivalent to the Client
   [network_interface](#network_interface) config option.
* `-network-speed<MBits>`: Equivalent to the Client
  [network_speed](#network_speed) config option.
* `-node=<name>`: Equivalent to the [name](#name) config option.
* `-node-class=<class>`: Equivalent to the Client [node_class](#node_class)
  config option.
* `-region=<region>`: Equivalent to the [region](#region) config option.
* `-rejoin`: Equivalent to the [rejoin_after_leave](#rejoin_after_leave) config option.
* `-retry-interval`: Equivalent to the [retry_interval](#retry_interval) config option.
* `-retry-join`: Similar to `-join` but allows retrying a join if the first attempt fails.
* `-retry-max`: Similar to the [retry_max](#retry_max) config option.
* `-server`: Enable server mode on the local agent.
* `-servers=<host:port>`: Equivalent to the Client [servers](#servers) config
  option.
* `-state-dir=<path>`: Equivalent to the Client [state_dir](#state_dir) config
  option.
