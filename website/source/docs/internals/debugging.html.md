---
layout: "docs"
page_title: "Debugging"
sidebar_current: "docs-internals-debug"
description: |-
  Terraform has detailed logs which can be enabled by setting the TF_LOG environmental variable to any value. This will cause detailed logs to appear on stderr
---

# Debugging Terraform

Terraform has detailed logs which can be enabled by setting the `TF_LOG` environmental variable to any value. This will cause detailed logs to appear on stderr.

You can set `TF_LOG` to one of the log levels `TRACE`, `DEBUG`, `INFO`, `WARN` or `ERROR` to change the verbosity of the logs. `TRACE` is the most verbose and it is the default if `TF_LOG` is set to something other than a log level name.

To persist logged output you can set `TF_LOG_PATH` in order to force the log to always go to a specific file when logging is enabled. Note that even when `TF_LOG_PATH` is set, `TF_LOG` must be set in order for any logging to be enabled.

If you find a bug with Terraform, please include the detailed log by using a service such as gist.
