---
layout: "circonus"
page_title: "Circonus: circonus_trigger"
sidebar_current: "docs-circonus-resource-circonus_trigger"
description: |-
  Manages a Circonus trigger.
---

# circonus\_trigger

The ``circonus_trigger`` resource creates and manages a
[Circonus Trigger](https://login.circonus.com/resources/api/calls/rule_set).

~> **NOTE regarding `cirocnus_trigger` vs a Circonus Ruleset:** The
`circonus_trigger` resource is implemented in terms of a [Circonus
Ruleset](https://login.circonus.com/resources/api/calls/rule_set).  The
`circonus_trigger` creates a higher-level abstraction over the implementation of a
Ruleset.  As such, the naming and structure does not map 1:1 with the
underlying Circonus API.

## Usage

```
variable "myapp-tags" {
  type = "list"
  default = [ "app:myapp", "owner:myteam" ]
}

resource "circonus_trigger" "myapp-cert-ttl-alarm" {
  check = "${circonus_check.myapp-https.checks[0]}"
  stream_name = "cert_end_in"
  link = "https://wiki.example.org/playbook/how-to-renew-cert"

  if {
    value {
      less = "${2 * 24 * 3600}"
    }

    then {
      notify = [ "${circonus_contact_group.myapp-owners.id}" ]
      severity = 1
    }
  }

  if {
    value {
      less = "${7 * 24 * 3600}"
    }

    then {
      notify = [ "${circonus_contact_group.myapp-owners.id}" ]
      severity = 2
    }
  }

  if {
    value {
      less = "${21 * 24 * 3600}"
    }

    then {
      notify = [ "${circonus_contact_group.myapp-owners.id}" ]
      severity = 3
    }
  }

  if {
    value {
      absent = "24h"
    }

    then {
      notify = [ "${circonus_contact_group.myapp-owners.id}" ]
      severity = 1
    }
  }

  tags = [ "${var.myapp-tags}" ]
}

resource "circonus_trigger" "myapp-healthy-alarm" {
  check = "${circonus_check.myapp-https.checks[0]}"
  stream_name = "duration"
  link = "https://wiki.example.org/playbook/debug-down-app"

  if {
    value {
      # SEV1 if it takes more than 9.5s for us to complete an HTTP request
      more = "${9.5 * 1000}"
    }

    then {
      notify = [ "${circonus_contact_group.myapp-owners.id}" ]
      severity = 1
    }
  }

  if {
    value {
      # SEV2 if it takes more than 5s for us to complete an HTTP request
      more = "${5 * 1000}"
    }

    then {
      notify = [ "${circonus_contact_group.myapp-owners.id}" ]
      severity = 2
    }
  }

  if {
    value {
      # SEV3 if the average response time is more than 500ms using a moving
      # average over the last 10min.  Any transient problems should have
      # resolved themselves by now.  Something's wrong, need to page someone.
      over {
        last = "10m"
        using = "average"
      }
      more = "500"
    }

    then {
      notify = [ "${circonus_contact_group.myapp-owners.id}" ]
      severity = 3
    }
  }

  if {
    value {
      # SEV4 if it takes more than 500ms for us to complete an HTTP request.  We
      # want to record that things were slow, but not wake anyone up if it
      # momentarily pops above 500ms.
      less = "500"
    }

    then {
      notify = [ "${circonus_contact_group.myapp-owners.id}" ]
      severity = 3
    }
  }

  if {
    value {
      # If for whatever reason we're not recording any values for the last
      # 24hrs, fire off a SEV1.
      absent = "24h"
    }

    then {
      notify = [ "${circonus_contact_group.myapp-owners.id}" ]
      severity = 1
    }
  }

  tags = [ "${var.myapp-tags}" ]
}

resource "circonus_contact_group" "myapp-owners" {
  name = "My App Owners"
  tags = [ "${var.myapp-tags}" ]
}

resource "circonus_check" "myapp-https" {
  name = "My App's HTTPS Check"

  notes = <<-EOF
A check to create metric streams for Time to First Byte, HTTP transaction
duration, and the TTL of a TLS cert.
EOF

  collector {
    id = "/broker/1"
  }

  http {
    code = "^200$"
    headers = {
      X-Request-Type = "health-check",
    }
    url = "https://www.example.com/myapp/healthz"
  }

  stream {
    name = "${circonus_metric.myapp-cert-ttl.name}"
    tags = "${circonus_metric.myapp-cert-ttl.tags}"
    type = "${circonus_metric.myapp-cert-ttl.type}"
    unit = "${circonus_metric.myapp-cert-ttl.unit}"
  }

  stream {
    name = "${circonus_metric.myapp-duration.name}"
    tags = "${circonus_metric.myapp-duration.tags}"
    type = "${circonus_metric.myapp-duration.type}"
    unit = "${circonus_metric.myapp-duration.unit}"
  }

  period       = 60
  tags         = ["source:circonus", "author:terraform"]
  timeout      = 10
}

resource "circonus_metric" "myapp-cert-ttl" {
  name = "cert_end_in"
  type = "numeric"
  unit = "seconds"
  tags = [ "${var.myapp-tags}", "resource:tls" ]
}

resource "circonus_metric" "myapp-duration" {
  name = "duration"
  type = "numeric"
  unit = "miliseconds"
  tags = [ "${var.myapp-tags}" ]
}
```

## Argument Reference

* `check` - (Required) The Circonus ID that this Trigger will use to search for
  a metric stream to alert on.

* `if` - (Required) One or more ordered predicate clauses that describe when
  Circonus should generate a notification.  See below for details on the
  structure of an `if` configuration clause.

* `link` - (Optional) A link to external documentation (or anything else you
  feel is important) when a notification is sent.  This value will show up in
  email alerts and the Circonus UI.

* `metric_type` - (Optional) The type of stream this trigger will operate on.
  Valid values are `numeric` (the default) and `text`.

* `notes` - (Optional) Notes about this trigger.

* `parent` - (Optional) A Circonus Metric ID that, if specified and active with
  a severity 1 alert, will silence this trigger until all of the severity 1
  alarms on the parent clear.  This value must match the format
  `${check_id}_${metric_name}`.

* `stream_name` - (Required) The name of the metric stream within a given check
  that this trigger is active on.

* `tags` - (Optional) A list of tags assigned to this trigger.

## `if` Configuration

The `if` configuration block is an
[ordered list of rules](https://login.circonus.com/user/docs/Alerting/Rules/Configure) that
are evaluated in order, first to last.  The first `if` condition to evaluate
true shortcircuits all other `if` blocks in this trigger.  An `if` block is also
referred to as a "rule."  It is advised that all high-severity rules are ordered
before low-severity rules otherwise low-severity rules will mask notifications
that should be delivered with a high-severity.

`if` blocks are made up of two configuration blocks: `value` and `then`.  The
`value` configuration block specifies the criteria underwhich the metric streams
are evaluated.  The `then` configuration block, optional, specifies what action
to take.

### `value` Configuration

A `value` block can have only one of several "predicate" attributes specified
because they conflict with each other.  The list of mutually exclusive
predicates is dependent on the `metric_type`.  To evaluate multiple predicates,
create multiple `if` configuration blocks in the proper order.

#### `numeric` Predicates

Metric types of type `numeric` support the following predicates.  Only one of
the following predicates may be specified at a time.

* `absent` - (Optional) If a metric has not been observed in this duration the
  rule will fire.  When present, this duration is evaluated in terms of seconds.

* `changed` - (Optional) A boolean indicating this rule should fire when the
  value changes (e.g. `n != n<sub>1</sub>`).

* `less` - (Optional) When the value is less than this value, this rule will
  fire (e.g. `n < ${less}`).

* `more` - (Optional) When the value is greater than this value, this rule
  will fire (e.g. `n > ${more}`).

Additionally, a `numeric` check can also evaluate data based on a windowing
function versus the last measured value in the metric stream.  In order to have
a rule evaluate on derived value from a window, include a nested `over`
attribute inside of the `value` configuration block.  An `over` attribute needs
two attributes:

* `last` - (Optional) A duration for the sliding window.  Default `300s`.

* `using` - (Optional) The window function to use over the `last` interval.
  Valid window functions include: `average` (the default), `stddev`, `derive`,
  `derive_stddev`, `counter`, `counter_stddev`, `derive_2`, `derive_2_stddev`,
  `counter_2`, and `counter_2_stddev`.

#### `text` Predicates

Metric types of type `text` support the following predicates:

* `absent` - (Optional) If a metric has not been observed in this duration the
  rule will fire.  When present, this duration is evaluated in terms of seconds.

* `changed` - (Optional) A boolean indicating this rule should fire when the
  last value in the metric stream changed from it's previous value (e.g. `n !=
  n-1`).

* `contains` - (Optional) When the last value in the metric stream the value is
  less than this value, this rule will fire (e.g. `strstr(n, ${contains}) !=
  NULL`).

* `equals` - (Optional) When the last value in the metric stream value exactly
  matches this configured value, this rule will fire (e.g. `strcmp(n, ${equals})
  == 0`).

* `excludes` - (Optional) When the last value in the metric stream does not
  match this configured value, this rule will fire (e.g. `strstr(n, ${contains})
  == NULL`).

* `missing` - (Optional) When the last value in the metric stream does not match
  this configured value, this rule will fire (e.g. `strstr(n, ${missing}) ==
  NULL`).

### `then` Configuration

A `then` block can have the following attributes:

* `after` - (Optional) Only execute this notification after waiting for this
  number of minutes.  Defaults to immediately, or `0m`.
* `notify` - (Optional) A list of contact group IDs to notify when this rule is
  sends off a notification.
* `severity` - (Optional) The severity level of the notification.  This can be
  set to any value between `1` and `5`.  Defaults to `1`.

## Import Example

`circonus_trigger` supports importing resources.  Supposing the following
Terraform (and that the referenced [`circonus_metric`](metric.html)
and [`circonus_check`](check.html) have already been imported):

```
resource "circonus_trigger" "icmp-latency-alarm" {
  check = "${circonus_check.api_latency.checks[0]}"
  stream_name = "maximum"

  if {
    value {
      absent = "600s"
    }

    then {
      notify = [ "${circonus_contact_group.test-trigger.id}" ]
      severity = 1
    }
  }

  if {
    value {
      over {
        last = "120s"
        using = "average"
      }

      more = 0.5 # units are in miliseconds
    }

    then {
      notify = [ "${circonus_contact_group.test-trigger.id}" ]
      severity = 2
    }
  }
}
```

It is possible to import a `circonus_trigger` resource with the following command:

```
$ terraform import circonus_trigger.usage ID
```

Where `ID` is the `_cid` or Circonus ID of the Ruleset
(e.g. `/rule_set/201285_maximum`) and `circonus_trigger.icmp-latency-alarm` is
the name of the resource whose state will be populated as a result of the
command.
