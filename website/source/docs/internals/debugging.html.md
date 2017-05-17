---
layout: "docs"
page_title: "Debugging"
sidebar_current: "docs-internals-debug"
description: |-
  Terraform has detailed logs which can be enabled by setting the TF_LOG environment variable to any value. This will cause detailed logs to appear on stderr
---

# Debugging Terraform

Terraform has detailed logs which can be enabled by setting the `TF_LOG` environment variable to any value. This will cause detailed logs to appear on stderr.

You can set `TF_LOG` to one of the log levels `TRACE`, `DEBUG`, `INFO`, `WARN` or `ERROR` to change the verbosity of the logs. `TRACE` is the most verbose and it is the default if `TF_LOG` is set to something other than a log level name.

To persist logged output you can set `TF_LOG_PATH` in order to force the log to always be appended to a specific file when logging is enabled. Note that even when `TF_LOG_PATH` is set, `TF_LOG` must be set in order for any logging to be enabled.

If you find a bug with Terraform, please include the detailed log by using a service such as gist.

## Interpreting a Crash Log

If Terraform ever crashes (a "panic" in the Go runtime), it saves a log file
with the debug logs from the session as well as the panic message and backtrace
to `crash.log`. Generally speaking, this log file is meant to be passed along
to the developers via a GitHub Issue. As a user, you're not required to dig
into this file.

However, if you are interested in figuring out what might have gone wrong
before filing an issue, here are the basic details of how to read a crash
log.

The most interesting part of a crash log is the panic message itself and the
backtrace immediately following. So the first thing to do is to search the file
for `panic: `, which should jump you right to this message. It will look
something like this:

```text
panic: runtime error: invalid memory address or nil pointer dereference

goroutine 123 [running]:
panic(0xabc100, 0xd93000a0a0)
	/opt/go/src/runtime/panic.go:464 +0x3e6
github.com/hashicorp/terraform/builtin/providers/aws.resourceAwsSomeResourceCreate(...)
	/opt/gopath/src/github.com/hashicorp/terraform/builtin/providers/aws/resource_aws_some_resource.go:123 +0x123
github.com/hashicorp/terraform/helper/schema.(*Resource).Refresh(...)
	/opt/gopath/src/github.com/hashicorp/terraform/helper/schema/resource.go:209 +0x123
github.com/hashicorp/terraform/helper/schema.(*Provider).Refresh(...)
	/opt/gopath/src/github.com/hashicorp/terraform/helper/schema/provider.go:187 +0x123
github.com/hashicorp/terraform/rpc.(*ResourceProviderServer).Refresh(...)
	/opt/gopath/src/github.com/hashicorp/terraform/rpc/resource_provider.go:345 +0x6a
reflect.Value.call(...)
	/opt/go/src/reflect/value.go:435 +0x120d
reflect.Value.Call(...)
	/opt/go/src/reflect/value.go:303 +0xb1
net/rpc.(*service).call(...)
	/opt/go/src/net/rpc/server.go:383 +0x1c2
created by net/rpc.(*Server).ServeCodec
	/opt/go/src/net/rpc/server.go:477 +0x49d
```

The key part of this message is the first two lines that involve `hashicorp/terraform`. In this example:

```text
github.com/hashicorp/terraform/builtin/providers/aws.resourceAwsSomeResourceCreate(...)
	/opt/gopath/src/github.com/hashicorp/terraform/builtin/providers/aws/resource_aws_some_resource.go:123 +0x123
```

The first line tells us that the method that failed is
`resourceAwsSomeResourceCreate`, which we can deduce that involves the creation
of a (fictional) `aws_some_resource`.

The second line points to the exact line of code that caused the panic,
which--combined with the panic message itself--is normally enough for a
developer to quickly figure out the cause of the issue.

As a user, this information can help work around the problem in a pinch, since
it should hopefully point to the area of the code base in which the crash is
happening.
