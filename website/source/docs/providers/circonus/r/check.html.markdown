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

```
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

  stream {
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

* `collector` - (Required) A collector ID.  The collector(s) that are
  responsible for running a `circonus_check`. The `id` can be the Circonus ID
  for a Circonus collector (a.k.a. "broker") running in the cloud or an
  enterprise collector running in your datacenter.  One collection of metrics
  will be automatically created for each `collector` specified.

* `json` - (Optional) A JSON check.  See below for details on how to configure
  the `json` check.

* `icmp_ping` - (Optional) An ICMP ping check.  See below for details on how to
  configure the `icmp_ping` check.

* `metric_limit` - (Optional) Setting a metric limit will tell the Circonus
  backend to periodically look at the check to see if there are additional
  metrics the collector has seen that we should collect. It will not reactivate
  metrics previously collected and then marked as inactive. Values are `0` to
  disable, `-1` to enable all metrics or `N+` to collect up to the value `N`
  (both `-1` and `N+` can not exceed other account restrictions).

* `name` - (Optional) The name of the check that will be displayed in the web
  interface.

* `notes` - (Optional) Notes about this check.

* `period` - (Optional) The period between each time the check is made in
  seconds.

* `postgresql` - (Optional) A PostgreSQL check.  See below for details on how to
  configure the `postgresql` check.

* `stream` - (Required) A list of one or more `stream` configurations.  See
  below for a list of supported `stream` attrbutes.  A collection of known
  metrics are aggregated into a metric stream.

* `tags` - (Optional) A list of tags assigned to this check.

* `target` - (Required) A string containing the location of the thing being
  checked.  This value changes based on the check type.  For example, for an
  `http` check type this would be the URL you're checking. For a DNS check it
  would be the hostname you wanted to look up.

* `timeout` - (Optional) A floating point number representing the maximum number
  of seconds this check should wait for a result.  Defaults to `10.0`.

## Supported `stream` Attributes

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

### `icmp_ping` Check Type Attributes

The `icmp_ping` check requires the `target` top-level attribute to be set.

* `availability` - (Optional) The percentage of ping packets that must be
  returned for this measurement to be considered successful.  Defaults to
  `100.0`.
* `count` - (Optional) The number of ICMP ping packets to send.  Defaults to
  `5`.
* `interval` - (Optional) Interval between packets.  Defaults to `2s`.

### `json` Check Type Attributes

* `headers` - (Optional) A map of the HTTP headers to be sent when executing the
  check.

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

### `postgresql` Check Type Attributes

The `postgresql` check requires the `target` top-level attribute to be set.

* `dsn` - (Required) The [PostgreSQL DSN/connect
  string](https://www.postgresql.org/docs/current/static/libpq-connect.html) to
  use to talk to PostgreSQL.
* `query` - (Required) The SQL query to execute.

## Import Example

`circonus_check` supports importing resources.  Supposing the following
Terraform (and that the referenced [`circonus_metric`](metric.html) has already
been imported):

```
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

  stream {
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
