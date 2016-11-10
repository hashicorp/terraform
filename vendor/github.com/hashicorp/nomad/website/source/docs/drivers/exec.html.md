---
layout: "docs"
page_title: "Drivers: Exec"
sidebar_current: "docs-drivers-exec"
description: |-
  The Exec task driver is used to run binaries using OS isolation primitives.
---

# Isolated Fork/Exec Driver

Name: `exec`

The `exec` driver is used to simply execute a particular command for a task.
However, unlike [`raw_exec`](raw_exec.html) it uses the underlying isolation
primitives of the operating system to limit the tasks access to resources. While
simple, since the `exec` driver  can invoke any command, it can be used to call
scripts or other wrappers which provide higher level features.

## Task Configuration

The `exec` driver supports the following configuration in the job spec:

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
    driver = "exec"

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
    driver = "exec"

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

The `exec` driver can only be run when on Linux and running Nomad as root.
`exec` is limited to this configuration because currently isolation of resources
is only guaranteed on Linux. Further the host must have cgroups mounted properly
in order for the driver to work.

If you are receiving the error `* Constraint "missing drivers" filtered <> nodes`
and using the exec driver, check to ensure that you are running Nomad as root. This
also applies for running Nomad in -dev mode.


## Client Attributes

The `exec` driver will set the following client attributes:

* `driver.exec` - This will be set to "1", indicating the
  driver is available.

## Resource Isolation

The resource isolation provided varies by the operating system of
the client and the configuration.

On Linux, Nomad will use cgroups, and a chroot to isolate the
resources of a process and as such the Nomad agent must be run as root.

### <a id="chroot"></a>Chroot
The chroot is populated with data in the following folders from the host
machine:

`["/bin", "/etc", "/lib", "/lib32", "/lib64", "/run/resolvconf", "/sbin",
"/usr"]`

This list is configurable through the agent client
[configuration file](/docs/agent/config.html#chroot_env).
