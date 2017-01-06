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
resource "circonus_check" "usage" {
  name = "JSON Application Metric Check"

  notes = <<EOF
A check to extract a few numeric metrics from a JSON response from myapp.
EOF

  type    = "json"
  target  = "host.example.org"
  collector {
    id = "/broker/1"
  }

  metric {
    name  = "some`app`counter"
    tags  = ["source:myapp", "creator:terraform"]
    type  = "numeric"
    units = "count"
  }

  metric {
    name  = "some`app`other_counter"
    tags  = ["source:myapp", "creator:terraform"]
    type  = "numeric"
    units = "count"
  }

  config {
    url = "https://myapp.example.org/stats"

    http_headers = {
      "Accept"       = "application/json"
      "X-Auth-Token" = "${var.auth_}"
    }
  }

  period       = 60
  tags = ["source:myapp", "creator:terraform"]
  timeout      = 10
}
```

## Argument Reference

* `active` - (Optional) Whether or not the check is enabled or not (default `true`).

* `collector` - (Required) A collector ID.  The collector(s) that are
  responsible for gathering the metrics. These can either be an ID for a
  Circonus collector running in the cloud or enterprise collectors running in
  your datacenter.  One collection of metrics will be automatically created for
  each collector.  Reminder: in Terraform the vernacular is "collector" where
  Circonus still refers to collectors as "brokers".

* `config` - (Optional) Configuration options for this check.  See below for a
  list of supported `config` attributes.

* `metric_limit` - (Optional) Setting a metric limit will tell the Circonus
  backend to periodically look at the check to see if there are additional
  metrics the collector has seen that we should collect. It will not reactivate
  metrics previously collected and then marked as inactive. Values are `0` to
  disable, `-1` to enable all metrics or `N+` to collect up to the value `N`
  (both `-1` and `N+` can not exceed other account restrictions).

  A number containing an integer, positive or negative

* `metric` - (Required) A list of one or more `metric` configurations.  See
  below for a list of supported `metric` attrbutes.

* `name` - (Optional) The name of the check that will be displayed in the web
  interface.

* `notes` - (Optional) Notes about this check.

* `period` - (Optional) The period between each time the check is made in
  seconds.

* `target` - (Required) A string containing the location of the thing being
  checked.  This value changes based on the check type.  For example, for an
  `http` check type this would be the URL you're checking. For a DNS check it
  would be the hostname you wanted to look up.

* `timeout` - (Optional) A floating point number representing the maximum number
  of seconds this check should wait for a result.  Defaults to `10.0`.

* `type` - (Required) The type of check that this is.  See below for a complete
  list of checks known by Terraform.

## Supported Check `config` Attributes

The following `config` attributes are only applicable for some `type`s of
checks.  In addition to some of these `config` attributes being only applicable
to specific check `type`s, some of these `config` attributes are `required`.

* `auth_method` - The HTTP Authentication method.
* `auth_password` - The HTTP Authentication user password.
* `auth_user` - The HTTP Authentication user name.
* `ca_chain` - A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for SSL checks).
* `certificate_file` - A path to a file containing the client certificate that will be presented to the remote server (for SSL checks).
* `ciphers` - A list of ciphers to be used in the SSL protocol (for SSL checks).
* `http_headers` - A map of HTTP headers.
* `http_version` - Sets the HTTP version for the check to use.
* `key_file` - A path to a file containing key to be used in conjunction with the cilent certificate (for SSL checks).
* `method` - The HTTP method to use.
* `payload` - The optional HTTP payload to send with the request.
* `port` - String representing either a port number or a standard port name
* `read_limit` - Sets an approximate limit on the data read (0 means no limit).
* `redirects`- The maximum number of Location header redirects to follow.
* `url` - The URL including schema and hostname (as you would type into a browser's location bar).

## Supported `metric` Attributes

The following attributes are available within a `metric`.

* `active` - (Optional) Whether or not the metric is active or not.  Defaults to `true`.
* `name` - (Optional) The name of the metric.  A string containing freeform text.
* `tags` - (Optional) A list of tags assigned to the metric.
* `type` - (Required) A string containing either `numeric`, `text`, `histogram`, `composite`, or `caql`.
* `units` - (Optional) The unit of measurement the metric represents (e.g., bytes, seconds, milliseconds). A string containing freeform text.

## Supported Check Types

The following is a list of supported check types in this release of Terraform:

* `caql`
* `cim`
* `circonuswindowsagent:nad`
* `circonuswindowsagent`
* `cloudwatch`
* `collectd`
* `composite`
* `dcm`
* `dhcp`
* `dns`
* `ec_console`
* `elasticsearch`
* `external`
* `ganglia`
* `googleanalytics`
* `haproxy`
* `http:apache`
* `http`
* `httptrap`
* `imap`
* `jmx`
* `json:couchdb`
* `json:mongodb`
* `json:nad`
* `json:riak`
* `json`
* `keynote_pulse`
* `keynote`
* `ldap`
* `memcached`
* `mongodb`
* `munin`
* `mysql`
* `newrelic_rpm`
* `nginx`
* `nrpe`
* `ntp`
* `oracle`
* `ping_icmp`
* `pop3`
* `postgres`
* `redis`
* `resmon`
* `smtp`
* `snmp:momentum`
* `snmp`
* `sqlserver`
* `ssh2`
* `statsd`
* `tcp`
* `varnish`

## Import Example

`circonus_check` supports importing resources.  Supposing the following
Terraform:

```
provider "circonus" {
  alias = "b8fec159-f9e5-4fe6-ad2c-dc1ec6751586"
}

resource "circonus_check" "usage" {
  type    = "json"
  target  = "api.circonus.com"
  collector {
    id = "/broker/1"
  }

  metric {
    name  = "_usage`0`_used"
    tags  = ["source:circonus", "creator:terraform"]
    type  = "numeric"
  }

  metric {
    name  = "_usage`0`_limit"
    tags  = ["source:circonus", "creator:terraform"]
    type  = "numeric"
  }

  config {
    url = "https://api.circonus.com/account/current"

    http_headers = {
      "Accept"                = "application/json"
      "X-Circonus-App-Name"   = "TerraformCheck"
      "X-Circonus-Auth-Token" = "${var.circonus_api_token}"
    }
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
