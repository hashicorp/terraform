---
layout: "logentries"
page_title: "Logentries: logentries_log"
sidebar_current: "docs-logentries-log"
description: |-
  Creates a Logentries log.
---

# logentries\_log

Provides a Logentries log resource.

## Example Usage

```hcl
# Create a log and add it to the log set
resource "logentries_log" "app_log" {
  logset_id = "${logentries_logset.host_logs.id}"
  name      = "myapp-log"
  source    = "token"
}
```

## Argument Reference

The following arguments are supported:

* `logset_id` - (Required) The id of the `logentries_logset` resource.
* `name` - (Required) The name of the log. The name should be short and descriptive. For example, Apache Access, Hadoop Namenode.
* `retention_period` - (Optional, default `ACCOUNT_DEFAULT`) The retention period (`1W`, `2W`, `1M`, `2M`, `6M`, `1Y`, `2Y`, `UNLIMITED`, `ACCOUNT_DEFAULT`)
* `source` - (Optional, default `token`) The log source (`token`, `syslog`, `agent`, `api`). Review the Logentries [log inputs documentation](https://docs.logentries.com/docs/) for more information.
* `type` - (Optional) The log type. See the Logentries [log type documentation](https://logentries.com/doc/log-types/) for more information.

## Attributes Reference

The following attributes are exported:

* `token` - If the log `source` is `token`, this value holds the generated log token that is used by logging clients. See the Logentries [token-based input documentation](https://logentries.com/doc/input-token/) for more information.
