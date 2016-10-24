---
layout: "docs"
page_title: "Drivers: Raw Exec"
sidebar_current: "docs-drivers-raw-exec"
description: |-
  The Raw Exec task driver simply fork/execs and provides no isolation.
---

# Raw Fork/Exec Driver

Name: `raw_exec`

The `raw_exec` driver is used to execute a command for a task without any
isolation. Further, the task is started as the same user as the Nomad process.
As such, it should be used with extreme care and is disabled by default.

## Task Configuration

The `raw_exec` driver supports the following configuration in the job spec:

* `command` - The command to execute. Must be provided. If executing a binary
  that exists on the host, the path must be absolute. If executing a binary that
  is download from an [`artifact`](/docs/jobspec/index.html#artifact_doc), the
  path can be relative from the allocations's root directory.

*   `args` - (Optional) A list of arguments to the optional `command`.
    References to environment variables or any [interpretable Nomad
    variables](/docs/jobspec/interpreted.html) will be interpreted
    before launching the task. For example:

    ```
        args = ["${nomad.datacenter}", "${MY_ENV}", "${meta.foo}"]
    ```

## Examples

To run a binary present on the Node:

```
  task "example" {
    driver = "raw_exec"

    config {
      # When running a binary that exists on the host, the path must be absolute
      command = "/bin/sleep"
      args = ["1"]
    }
  }
```

To execute a binary downloaded from an [`artifact`](/docs/jobspec/index.html#artifact_doc):

```
  task "example" {
    driver = "raw_exec"

    config {
      command = "binary.bin"
    }

    artifact {
      source = "https://dl.dropboxusercontent.com/u/1234/binary.bin"
      options {
        checksum = "sha256:abd123445ds4555555555"
      }
    }
  }
```

## Client Requirements

The `raw_exec` driver can run on all supported operating systems. It is however
disabled by default. In order to be enabled, the Nomad client configuration must
explicitly enable the `raw_exec` driver in the client's
[options](/docs/agent/config.html#options) field:

```
    client {
        options = {
            "driver.raw_exec.enable" = "1"
        }
    }
```

## Client Attributes

The `raw_exec` driver will set the following client attributes:

* `driver.raw_exec` - This will be set to "1", indicating the
  driver is available.

## Resource Isolation

The `raw_exec` driver provides no isolation.
