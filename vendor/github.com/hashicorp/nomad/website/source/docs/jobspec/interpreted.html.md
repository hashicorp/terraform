---
layout: "docs"
page_title: "Interpreted Variables"
sidebar_current: "docs-jobspec-interpreted"
description: |-
  Learn about the Nomad's interpreted variables.
---
# Interpreted Variables

Nomad support interpreting two classes of variables, node attributes and runtime
environment variables. Node attributes are interpretable in constraints, task
environment variables and certain driver fields. Runtime environment variables
are not interpretable in constraints because they are only defined once the
scheduler has placed them on a particular node.

The syntax for interpreting variables is `${variable}`. An example and a
comprehensive list of interpretable fields can be seen below:

```
task "demo" {
    driver = "docker"

    # Drivers support interpreting node attributes and runtime environment
    # variables
    config {
        image = "my-app"

        # Interpret runtime variables to inject the address to bind to and the
        # location to write logs to.
        args = ["--bind=${NOMAD_ADDR_RPC}", "--logs=${NOMAD_ALLOC_DIR}/logs"]

        port_map {
            RPC = 6379
        }
    }

    # Constraints only support node attributes as runtime environment variables
    # are only defined after the task is placed on a node.
    constraint {
        attribute = "${attr.kernel.name}"
        value = "linux"
    }

    # Environment variables are interpreted and can contain both runtime and
    # node attributes.
    env {
        "DC" = "Running on datacenter ${node.datacenter}"
        "VERSION" = "Version ${NOMAD_META_VERSION}"
    }

    # Meta keys are also interpretable.
    meta {
        VERSION = "v0.3"
    }
}
```

## Node Variables <a id="interpreted_node_vars"></a>

Below is a full listing of node attributes that are interpretable. These
attributes are Interpreted by __both__ constraints and within the task and
driver.

<table class="table table-bordered table-striped">
  <tr>
    <th>Variable</th>
    <th>Description</th>
    <th>Example</th>
  </tr>
  <tr>
    <td>${node.unique.id}</td>
    <td>The 36 character unique client node identifier</td>
    <td>9afa5da1-8f39-25a2-48dc-ba31fd7c0023</td>
  </tr>
  <tr>
    <td>${node.datacenter}</td>
    <td>The client node's datacenter</td>
    <td>dc1</td>
  </tr>
  <tr>
    <td>${node.unique.name}</td>
    <td>The client node's name</td>
    <td>nomad-client-10-1-2-4</td>
  </tr>
  <tr>
    <td>${node.class}</td>
    <td>The client node's class</td>
    <td>linux-64bit</td>
  </tr>
  <tr>
    <td>${attr."key"}</td>
    <td>The attribute given by `key` on the client node.</td>
    <td>platform.aws.instance-type:r3.large</td>
  </tr>
  <tr>
    <td>${meta."key"}</td>
    <td>The metadata value given by `key` on the client node.</td>
    <td></td>
  </tr>
</table>

Below is a table documenting common node attributes:

<table class="table table-bordered table-striped">
  <tr>
    <th>Attribute</th>
    <th>Description</th>
  </tr>
  <tr>
    <td>arch</td>
    <td>CPU architecture of the client. Examples: `amd64`, `386`</td>
  </tr>
  <tr>
    <td>consul.datacenter</td>
    <td>The Consul datacenter of the client node if Consul found</td>
  </tr>
  <tr>
    <td>cpu.numcores</td>
    <td>Number of CPU cores on the client</td>
  </tr>
  <tr>
    <td>driver."key"</td>
    <td>See the [task drivers](/docs/drivers/index.html) for attribute documentation</td>
  </tr>
  <tr>
    <td>unique.hostname</td>
    <td>Hostname of the client</td>
  </tr>
  <tr>
    <td>kernel.name</td>
    <td>Kernel of the client. Examples: `linux`, `darwin`</td>
  </tr>
  <tr>
    <td>kernel.version</td>
    <td>Version of the client kernel. Examples: `3.19.0-25-generic`, `15.0.0`</td>
  </tr>
  <tr>
    <td>platform.aws.ami-id</td>
    <td>On EC2, the AMI ID of the client node</td>
  </tr>
  <tr>
    <td>platform.aws.instance-type</td>
    <td>On EC2, the instance type of the client node</td>
  </tr>
  <tr>
    <td>os.name</td>
    <td>Operating system of the client. Examples: `ubuntu`, `windows`, `darwin`</td>
  </tr>
  <tr>
    <td>os.version</td>
    <td>Version of the client OS</td>
  </tr>
</table>

## Environment Variables <a id="interpreted_env_vars"></a>

The following are runtime environment variables that describe the environment
the task is running in. These are only defined once the task has been placed on
a particular node and as such can not be used in constraints.

<table class="table table-bordered table-striped">
  <tr>
    <th>Variable</th>
    <th>Description</th>
  </tr>
  <tr>
    <td>${NOMAD_ALLOC_DIR}</td>
    <td>The path to the shared `alloc/` directory. See
    [here](/docs/jobspec/environment.html#task_dir) for more
    information.</td>
  </tr>
  <tr>
    <td>${NOMAD_TASK_DIR}</td>
    <td>The path to the task `local/` directory. See
    [here](/docs/jobspec/environment.html#task_dir) for more
    information.</td>
  </tr>
  <tr>
    <td>${NOMAD_MEMORY_LIMIT}</td>
    <td>The memory limit in MBytes for the task</td>
  </tr>
  <tr>
    <td>${NOMAD_CPU_LIMIT}</td>
    <td>The CPU limit in MHz for the task</td>
  </tr>
  <tr>
    <td>${NOMAD_ALLOC_ID}</td>
    <td>The allocation ID of the task</td>
  </tr>
  <tr>
    <td>${NOMAD_ALLOC_NAME}</td>
    <td>The allocation name of the task</td>
  </tr>
  <tr>
    <td>${NOMAD_ALLOC_INDEX}</td>
    <td>The allocation index; useful to distinguish instances of task groups</td>
  </tr>
  <tr>
    <td>${NOMAD_TASK_NAME}</td>
    <td>The task's name</td>
  </tr>
  <tr>
    <td>${NOMAD_IP_"label"}</td>
    <td>The IP for the given port `label`. See
    [here](/docs/jobspec/networking.html) for more information.</td>
  </tr>
  <tr>
    <td>${NOMAD_PORT_"label"}</td>
    <td>The port for the port `label`. See [here](/docs/jobspec/networking.html)
    for more information.</td>
  </tr>
  <tr>
    <td>${NOMAD_ADDR_"label"}</td>
    <td>The `ip:port` pair for the given port `label`. See
    [here](/docs/jobspec/networking.html) for more information.</td>
  </tr>
  <tr>
    <td>${NOMAD_HOST_PORT_"label"}</td>
    <td>The port on the host if port forwarding is being used for the port
    `label`. See [here](/docs/jobspec/networking.html#mapped_ports) for more
    information.</td>
  </tr>
  <tr>
    <td>${NOMAD_META_"key"}</td>
    <td>The metadata value given by `key` on the task's metadata</td>
  </tr>
  <tr>
    <td>${"env_key"}</td>
    <td>Interpret an environment variable with key `env_key` set on the task.</td>
  </tr>
</table>

