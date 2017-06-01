---
layout: "circonus"
page_title: "Circonus: circonus_check"
sidebar_current: "docs-circonus-resource-circonus_check"
description: |-
  Manages a Circonus check.
---

# circonus\_check

The ``circonus_check`` resource creates and manages a
[Circonus Check](https://login.circonus.com/resources/api/calls/check_bundle).

~> **NOTE regarding `cirocnus_check` vs a Circonus Check Bundle:** The
`circonus_check` resource is implemented in terms of a
[Circonus Check Bundle](https://login.circonus.com/resources/api/calls/check_bundle).
The `circonus_check` creates a higher-level abstraction over the implementation
of a Check Bundle.  As such, the naming and structure does not map 1:1 with the
underlying Circonus API.

## Usage

```hcl
variable api_token {
  default = "my-token"
}

resource "circonus_check" "usage" {
  name = "Circonus Usage Check"

  notes = <<-EOF
A check to extract a usage metric.
EOF

  collector {
    id = "/broker/1"
  }

  metric {
    name = "${circonus_metric.used.name}"
    tags = "${circonus_metric.used.tags}"
    type = "${circonus_metric.used.type}"
    unit = "${circonus_metric.used.unit}"
  }

  json {
    url = "https://api.circonus.com/v2"

    http_headers = {
      Accept                = "application/json"
      X-Circonus-App-Name   = "TerraformCheck"
      X-Circonus-Auth-Token = "${var.api_token}"
    }
  }

  period       = 60
  tags         = ["source:circonus", "author:terraform"]
  timeout      = 10
}

resource "circonus_metric" "used" {
  name = "_usage`0`_used"
  type = "numeric"
  unit = "qty"

  tags = {
    source = "circonus"
  }
}
```

## Argument Reference

* `active` - (Optional) Whether or not the check is enabled or not (default
  `true`).

* `caql` - (Optional) A [Circonus Analytics Query Language
  (CAQL)](https://login.circonus.com/user/docs/CAQL) check.  See below for
  details on how to configure a `caql` check.

* `cloudwatch` - (Optional) A [CloudWatch
  check](https://login.circonus.com/user/docs/Data/CheckTypes/CloudWatch) check.
  See below for details on how to configure a `cloudwatch` check.

* `collector` - (Required) A collector ID.  The collector(s) that are
  responsible for running a `circonus_check`. The `id` can be the Circonus ID
  for a Circonus collector (a.k.a. "broker") running in the cloud or an
  enterprise collector running in your datacenter.  One collection of metrics
  will be automatically created for each `collector` specified.

* `consul` - (Optional) A native Consul check.  See below for details on how to
  configure a `consul` check.

* `http` - (Optional) A poll-based HTTP check.  See below for details on how to configure
  the `http` check.

* `httptrap` - (Optional) An push-based HTTP check.  This check method expects
  clients to send a specially crafted HTTP JSON payload.  See below for details
  on how to configure the `httptrap` check.

* `icmp_ping` - (Optional) An ICMP ping check.  See below for details on how to
  configure the `icmp_ping` check.

* `json` - (Optional) A JSON check.  See below for details on how to configure
  the `json` check.

* `metric` - (Required) A list of one or more `metric` configurations.  All
  metrics obtained from this check instance will be available as individual
  metric streams.  See below for a list of supported `metric` attrbutes.

* `metric_limit` - (Optional) Setting a metric limit will tell the Circonus
  backend to periodically look at the check to see if there are additional
  metrics the collector has seen that we should collect. It will not reactivate
  metrics previously collected and then marked as inactive. Values are `0` to
  disable, `-1` to enable all metrics or `N+` to collect up to the value `N`
  (both `-1` and `N+` can not exceed other account restrictions).

* `mysql` - (Optional) A MySQL check.  See below for details on how to configure
  the `mysql` check.

* `name` - (Optional) The name of the check that will be displayed in the web
  interface.

* `notes` - (Optional) Notes about this check.

* `period` - (Optional) The period between each time the check is made in
  seconds.

* `postgresql` - (Optional) A PostgreSQL check.  See below for details on how to
  configure the `postgresql` check.

* `statsd` - (Optional) A statsd check.  See below for details on how to
  configure the `statsd` check.

* `tags` - (Optional) A list of tags assigned to this check.

* `target` - (Required) A string containing the location of the thing being
  checked.  This value changes based on the check type.  For example, for an
  `http` check type this would be the URL you're checking. For a DNS check it
  would be the hostname you wanted to look up.

* `tcp` - (Optional) A TCP check.  See below for details on how to configure the
  `tcp` check (includes TLS support).

* `timeout` - (Optional) A floating point number representing the maximum number
  of seconds this check should wait for a result.  Defaults to `10.0`.

## Supported `metric` Attributes

The following attributes are available within a `metric`.

* `active` - (Optional) Whether or not the metric is active or not.  Defaults to `true`.
* `name` - (Optional) The name of the metric.  A string containing freeform text.
* `tags` - (Optional) A list of tags assigned to the metric.
* `type` - (Required) A string containing either `numeric`, `text`, `histogram`, `composite`, or `caql`.
* `units` - (Optional) The unit of measurement the metric represents (e.g., bytes, seconds, milliseconds). A string containing freeform text.

## Supported Check Types

Circonus supports a variety of different checks.  Each check type has its own
set of options that must be configured.  Each check type conflicts with every
other check type (i.e. a `circonus_check` configured for a `json` check will
conflict with all other check types, therefore a `postgresql` check must be a
different `circonus_check` resource).

### `caql` Check Type Attributes

* `query` - (Required) The [CAQL
  Query](https://login.circonus.com/user/docs/caql_reference) to run.

Available metrics depend on the payload returned in the `caql` check.  See the
[`caql` check type](https://login.circonus.com/resources/api/calls/check_bundle) for
additional details.

### `cloudwatch` Check Type Attributes

* `api_key` - (Required) The AWS access key.  If this value is not explicitly
  set, this value is populated by the environment variable `AWS_ACCESS_KEY_ID`.

* `api_secret` - (Required) The AWS secret key.  If this value is not explicitly
  set, this value is populated by the environment variable `AWS_SECRET_ACCESS_KEY`.

* `dimmensions` - (Required) A map of the CloudWatch dimmensions to include in
  the check.

* `metric` - (Required) A list of metric names to collect in this check.

* `namespace` - (Required) The namespace to pull parameters from.

* `url` - (Required) The AWS URL to pull from.  This should be set to the
  region-specific endpoint (e.g. prefer
  `https://monitoring.us-east-1.amazonaws.com` over
  `https://monitoring.amazonaws.com`).

* `version` - (Optional) The version of the Cloudwatch API to use.  Defaults to
  `2010-08-01`.

Available metrics depend on the payload returned in the `cloudwatch` check.  See the
[`cloudwatch` check type](https://login.circonus.com/resources/api/calls/check_bundle) for
additional details.  The `circonus_check` `period` attribute must be set to
either `60s` or `300s` for CloudWatch metrics.

Example CloudWatch check (partial metrics collection):

```hcl
variable "cloudwatch_rds_tags" {
  type = "list"
  default = [
    "app:postgresql",
    "app:rds",
    "source:cloudwatch",
  ]
}

resource "circonus_check" "rds_metrics" {
  active = true
  name = "Terraform test: RDS Metrics via CloudWatch"
  notes = "Collect RDS metrics"
  period = "60s"

  collector {
    id = "/broker/1"
  }

  cloudwatch {
    dimmensions = {
      DBInstanceIdentifier = "my-db-name",
    }

    metric = [
      "CPUUtilization",
      "DatabaseConnections",
    ]

    namespace = "AWS/RDS"
    url = "https://monitoring.us-east-1.amazonaws.com"
  }

  metric {
    name = "CPUUtilization"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "%"
  }

  metric {
    name = "DatabaseConnections"
    tags = [ "${var.cloudwatch_rds_tags}" ]
    type = "numeric"
    unit = "connections"
  }
}
```

### `consul` Check Type Attributes

* `acl_token` - (Optional) An ACL Token authenticate the API request.  When an
  ACL Token is set, this value is transmitted as an HTTP Header in order to not
  show up in any logs.  The default value is an empty string.

* `allow_stale` - (Optional) A boolean value that indicates whether or not this
  check should require the health information come from the Consul leader node.
  For scalability reasons, this value defaults to `false`.  See below for
  details on detecting the staleness of health information.

* `ca_chain` - (Optional) A path to a file containing all the certificate
  authorities that should be loaded to validate the remote certificate (required
  when `http_addr` is a TLS-enabled endpoint).

* `certificate_file` - (Optional) A path to a file containing the client
  certificate that will be presented to the remote server (required when
  `http_addr` is a TLS-enabled endpoint).

* `check_blacklist` - (Optional) A list of check names to exclude from the
  result of checks (i.e. no metrics will be generated by whose check name is in
  the `check_blacklist`).  This blacklist is applied to the `node`,
  `service`, and `state` check modes.

* `ciphers` - (Optional) A list of ciphers to be used in the TLS protocol
  (only used when `http_addr` is a TLS-enabled endpoint).

* `dc` - (Optional) Explicitly name the Consul datacenter to use.  The default
  value is an empty string.  When an empty value is specified, the Consul
  datacenter of the agent at the `http_addr` is implicitly used.

* `headers` - (Optional) A map of the HTTP headers to be sent when executing the
  check.  NOTE: the `headers` attribute is processed last and will takes
  precidence over any other derived value that is transmitted as an HTTP header
  to Consul (i.e. it is possible to override the `acl_token` by setting a
  headers value).

* `http_addr` - (Optional) The Consul HTTP endpoint to to query for health
  information.  The default value is `http://consul.service.consul:8500`.  The
  scheme must change from `http` to `https` when the endpoint has been
  TLS-enabled.

* `key_file` - (Optional) A path to a file containing key to be used in
  conjunction with the cilent certificate (required when `http_addr` is a
  TLS-enabled endpoint).

* `node` - (Optional) Check the health of this node.  The value can be either a
  Consul Node ID (Consul Version >= 0.7.4) or Node Name.  See also the
  `service_blacklist`, `node_blacklist`, and `check_blacklist` attributes.  This
  attribute conflicts with the `service` and `state` attributes.

* `node_blacklist` - (Optional) A list of node IDs or node names to exclude from
  the results of checks (i.e. no metrics will be generated from nodes in the
  `node_blacklist`).  This blacklist is applied to the `node`, `service`, and
  `state` check modes.

* `service` - (Optional) Check the cluster-wide health of this named service.
  See also the `service_blacklist`, `node_blacklist`, and `check_blacklist`
  attributes.  This attribute conflicts with the `node` and `state` attributes.

* `service_blacklist` - (Optional) A list of service names to exclude from the
  result of checks (i.e. no metrics will be generated by services whose service
  name is in the `service_blacklist`).  This blacklist is applied to the `node`,
  `service`, and `state` check modes.

* `state` - (Optional) A Circonus check to monitor Consul checks across the
  entire Consul cluster.  This value may be either `passing`, `warning`, or
  `critical`.  This `consul` check mode is intended to act as the cluster check
  of last resort.  This check type is useful when first starting and is intended
  to act as a check of last resort before transitioning to explicitly defined
  checks for individual services or nodes.  The metrics returned from check will
  be sorted based on the `CreateIndex` of the entry in order to have a stable
  set of metrics in the array of returned values.  See also the
  `service_blacklist`, `node_blacklist`, and `check_blacklist` attributes.  This
  attribute conflicts with the `node` and `state` attributes.

Available metrics depend on the consul check being performed (`node`, `service`,
or `state`).  In addition to the data avilable from the endpoints, the `consul`
check also returns a set of metrics that are a variant of:
`{Num,Pct}{,Passing,Warning,Critical}{Checks,Nodes,Services}` (see the
`GLOB_BRACE` section of your local `glob(3)` documentation).

Example Consul check (partial metrics collection):

```hcl
resource "circonus_check" "consul_server" {
  active = true
  name = "%s"
  period = "60s"

  collector {
    # Collector ID must be an Enterprise broker able to reach the Consul agent
    # listed in `http_addr`.
    id = "/broker/2110"
  }

  consul {
    service = "consul"

    # Other consul check modes:
    # node = "consul1"
    # state = "critical"
  }

  metric {
    name = "NumNodes"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "numeric"
  }

  metric {
    name = "LastContact"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "Index"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "numeric"
    unit = "transactions"
  }

  metric {
    name = "KnownLeader"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "text"
  }

  tags = [ "source:consul", "lifecycle:unittest" ]
}
```

### `http` Check Type Attributes

* `auth_method` - (Optional) HTTP Authentication method to use.  When set must
  be one of the values `Basic`, `Digest`, or `Auto`.

* `auth_password` - (Optional) The password to use during authentication.

* `auth_user` - (Optional) The user to authenticate as.

* `body_regexp` - (Optional) This regular expression is matched against the body
  of the response. If a match is not found, the check will be marked as "bad."

* `ca_chain` - (Optional) A path to a file containing all the certificate
  authorities that should be loaded to validate the remote certificate (for TLS
  checks).

* `certificate_file` - (Optional) A path to a file containing the client
  certificate that will be presented to the remote server (for TLS checks).

* `ciphers` - (Optional) A list of ciphers to be used in the TLS protocol (for
  HTTPS checks).

* `code` - (Optional) The HTTP code that is expected. If the code received does
  not match this regular expression, the check is marked as "bad."

* `extract` - (Optional) This regular expression is matched against the body of
  the response globally. The first capturing match is the key and the second
  capturing match is the value. Each key/value extracted is registered as a
  metric for the check.

* `headers` - (Optional) A map of the HTTP headers to be sent when executing the
  check.

* `key_file` - (Optional) A path to a file containing key to be used in
  conjunction with the cilent certificate (for TLS checks).

* `method` - (Optional) The HTTP Method to use.  Defaults to `GET`.

* `payload` - (Optional) The information transferred as the payload of an HTTP
  request.

* `read_limit` - (Optional) Sets an approximate limit on the data read (`0`
  means no limit). Default `0`.

* `redirects` - (Optional) The maximum number of HTTP `Location` header
  redirects to follow. Default `0`.

* `url` - (Required) The target for this `json` check.  The `url` must include
  the scheme, host, port (optional), and path to use
  (e.g. `https://app1.example.org/healthz`)

* `version` - (Optional) The HTTP version to use.  Defaults to `1.1`.

Available metrics include: `body_match`, `bytes`, `cert_end`, `cert_end_in`,
`cert_error`, `cert_issuer`, `cert_start`, `cert_subject`, `code`, `duration`,
`truncated`, `tt_connect`, and `tt_firstbyte`.  See the
[`http` check type](https://login.circonus.com/resources/api/calls/check_bundle) for
additional details.

### `httptrap` Check Type Attributes

* `async_metrics` - (Optional) Boolean value specifies whether or not httptrap
  metrics are logged immediately or held until the status message is to be
  emitted.  Default `false`.

* `secret` - (Optional) Specify the secret with which metrics may be
  submitted.

Available metrics depend on the payload returned in the `httptrap` doc.  See
the [`httptrap` check type](https://login.circonus.com/resources/api/calls/check_bundle)
for additional details.

### `json` Check Type Attributes

* `auth_method` - (Optional) HTTP Authentication method to use.  When set must
  be one of the values `Basic`, `Digest`, or `Auto`.

* `auth_password` - (Optional) The password to use during authentication.

* `auth_user` - (Optional) The user to authenticate as.

* `ca_chain` - (Optional) A path to a file containing all the certificate
  authorities that should be loaded to validate the remote certificate (for TLS
  checks).

* `certificate_file` - (Optional) A path to a file containing the client
  certificate that will be presented to the remote server (for TLS checks).

* `ciphers` - (Optional) A list of ciphers to be used in the TLS protocol (for
  HTTPS checks).

* `headers` - (Optional) A map of the HTTP headers to be sent when executing the
  check.

* `key_file` - (Optional) A path to a file containing key to be used in
  conjunction with the cilent certificate (for TLS checks).

* `method` - (Optional) The HTTP Method to use.  Defaults to `GET`.

* `port` - (Optional) The TCP Port number to use.  Defaults to `81`.

* `read_limit` - (Optional) Sets an approximate limit on the data read (`0`
  means no limit). Default `0`.

* `redirects` - (Optional) The maximum number of HTTP `Location` header
  redirects to follow. Default `0`.

* `url` - (Required) The target for this `json` check.  The `url` must include
  the scheme, host, port (optional), and path to use
  (e.g. `https://app1.example.org/healthz`)

* `version` - (Optional) The HTTP version to use.  Defaults to `1.1`.

Available metrics depend on the payload returned in the `json` doc.  See the
[`json` check type](https://login.circonus.com/resources/api/calls/check_bundle) for
additional details.

### `icmp_ping` Check Type Attributes

The `icmp_ping` check requires the `target` top-level attribute to be set.

* `availability` - (Optional) The percentage of ping packets that must be
  returned for this measurement to be considered successful.  Defaults to
  `100.0`.
* `count` - (Optional) The number of ICMP ping packets to send.  Defaults to
  `5`.
* `interval` - (Optional) Interval between packets.  Defaults to `2s`.

Available metrics include: `available`, `average`, `count`, `maximum`, and
`minimum`.  See the
[`ping_icmp` check type](https://login.circonus.com/resources/api/calls/check_bundle)
for additional details.

### `mysql` Check Type Attributes

The `mysql` check requires the `target` top-level attribute to be set.

* `dsn` - (Required) The [MySQL DSN/connect
  string](https://github.com/go-sql-driver/mysql/blob/master/README.md) to
  use to talk to MySQL.
* `query` - (Required) The SQL query to execute.

### `postgresql` Check Type Attributes

The `postgresql` check requires the `target` top-level attribute to be set.

* `dsn` - (Required) The [PostgreSQL DSN/connect
  string](https://www.postgresql.org/docs/current/static/libpq-connect.html) to
  use to talk to PostgreSQL.
* `query` - (Required) The SQL query to execute.

Available metric names are dependent on the output of the `query` being run.

### `statsd` Check Type Attributes

* `source_ip` - (Required) Any statsd messages from this IP address (IPv4 or
  IPv6) will be associated with this check.

Available metrics depend on the metrics sent to the `statsd` check.

### `tcp` Check Type Attributes

* `banner_regexp` - (Optional) This regular expression is matched against the
  response banner. If a match is not found, the check will be marked as bad.

* `ca_chain` - (Optional) A path to a file containing all the certificate
  authorities that should be loaded to validate the remote certificate (for TLS
  checks).

* `certificate_file` - (Optional) A path to a file containing the client
  certificate that will be presented to the remote server (for TLS checks).

* `ciphers` - (Optional) A list of ciphers to be used in the TLS protocol (for
  HTTPS checks).

* `host` - (Required) Hostname or IP address of the host to connect to.

* `key_file` - (Optional) A path to a file containing key to be used in
  conjunction with the cilent certificate (for TLS checks).

* `port` - (Required) Integer specifying the port on which the management
  interface can be reached.

* `tls` - (Optional) When enabled establish a TLS connection.

Available metrics include: `banner`, `banner_match`, `cert_end`, `cert_end_in`,
`cert_error`, `cert_issuer`, `cert_start`, `cert_subject`, `duration`,
`tt_connect`, `tt_firstbyte`.  See the
[`tcp` check type](https://login.circonus.com/resources/api/calls/check_bundle)
for additional details.

Sample `tcp` check:

```hcl
resource "circonus_check" "tcp_check" {
  name = "TCP and TLS check"
  notes = "Obtains the connect time and TTL for the TLS cert"
  period = "60s"

  collector {
    id = "/broker/1"
  }

  tcp {
    host = "127.0.0.1"
    port = 443
    tls = true
  }

  metric {
    name = "cert_end_in"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "tt_connect"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "miliseconds"
  }

  tags = [ "${var.tcp_check_tags}" ]
}
```

## Out Parameters

* `check_by_collector` - Maps the ID of the collector (`collector_id`, the map
  key) to the `check_id` (value) that is registered to a collector.

* `check_id` - If there is only one `collector` specified for the check, this
  value will be populated with the `check_id`.  If more than one `collector` is
  specified in the check, then this value will be an empty string.
  `check_by_collector` will always be populated.

* `checks` - List of `check_id`s created by this `circonus_check`.  There is one
  element in this list per collector specified in the check.

* `created` - UNIX time at which this check was created.

* `last_modified` - UNIX time at which this check was last modified.

* `last_modified_by` - User ID in Circonus who modified this check last.

* `reverse_connect_urls` - Only relevant to Circonus support.

* `uuids` - List of Check `uuid`s created by this `circonus_check`.  There is
  one element in this list per collector specified in the check.

## Import Example

`circonus_check` supports importing resources.  Supposing the following
Terraform (and that the referenced [`circonus_metric`](metric.html) has already
been imported):

```hcl
provider "circonus" {
  alias = "b8fec159-f9e5-4fe6-ad2c-dc1ec6751586"
}

resource "circonus_metric" "used" {
  name = "_usage`0`_used"
  type = "numeric"
}

resource "circonus_check" "usage" {
  collector {
    id = "/broker/1"
  }

  json {
    url = "https://api.circonus.com/account/current"

    http_headers = {
      "Accept"                = "application/json"
      "X-Circonus-App-Name"   = "TerraformCheck"
      "X-Circonus-Auth-Token" = "${var.api_token}"
    }
  }

  metric {
    name = "${circonus_metric.used.name}"
    type = "${circonus_metric.used.type}"
  }
}
```

It is possible to import a `circonus_check` resource with the following command:

```
$ terraform import circonus_check.usage ID
```

Where `ID` is the `_cid` or Circonus ID of the Check Bundle
(e.g. `/check_bundle/12345`) and `circonus_check.usage` is the name of the
resource whose state will be populated as a result of the command.
