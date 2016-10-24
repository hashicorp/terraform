---
layout: "docs"
page_title: "JSON Job Specification"
sidebar_current: "docs-jobspec-json-syntax"
description: |-
  Learn about the Job specification used to submit jobs to Nomad in JSON.
---

# Job Specification

Jobs can be specified either in [HCL](https://github.com/hashicorp/hcl) or JSON.
This guide covers the JSON syntax for submitting jobs to Nomad. A useful command
for generating valid JSON versions of HCL jobs is `nomad run -output <job.nomad>`
which will emit a JSON version of the job.

## JSON Syntax

Below is an example of a JSON object that submits a `Periodic` job to Nomad:

```
{
    "Job": {
        "Region": "global",
        "ID": "example",
        "Name": "example",
        "Type": "batch",
        "Priority": 50,
        "AllAtOnce": false,
        "Datacenters": [
            "dc1"
        ],
        "Constraints": [
            {
                "LTarget": "${attr.kernel.name}",
                "RTarget": "linux",
                "Operand": "="
            }
        ],
        "TaskGroups": [
            {
                "Name": "cache",
                "Count": 1,
                "Constraints": null,
                "Tasks": [
                    {
                        "Name": "redis",
                        "Driver": "docker",
                        "User": "foo-user",
                        "Config": {
                            "image": "redis:latest",
                            "port_map": [
                                {
                                    "db": 6379
                                }
                            ]
                        },
                        "Constraints": null,
                        "Env": {
                            "foo": "bar",
                            "baz": "pipe"
                        }
                        "Services": [
                            {
                                "Name": "cache-redis",
                                "Tags": [
                                    "global",
                                    "cache"
                                ],
                                "PortLabel": "db",
                                "Checks": [
                                    {
                                        "Id": "",
                                        "Name": "alive",
                                        "Type": "tcp",
                                        "Command": "",
                                        "Args": null,
                                        "Path": "",
                                        "Protocol": "",
                                        "Interval": 10000000000,
                                        "Timeout": 2000000000
                                    }
                                ]
                            }
                        ],
                        "Resources": {
                            "CPU": 500,
                            "MemoryMB": 256,
                            "DiskMB": 300,
                            "IOPS": 0,
                            "Networks": [
                                {
                                    "ReservedPorts": [
                                        {
                                            "Label": "rpc",
                                            "Value": 25566
                                        }
                                    ],
                                    "DynamicPorts": [
                                        {
                                            "Label": "db",
                                        }
                                    ],
                                    "MBits": 10
                                }
                            ]
                        },
                        "Meta": {
                            "foo": "bar",
                            "baz": "pipe"
                        },
                        "KillTimeout": 5000000000,
                        "LogConfig": {
                            "MaxFiles": 10,
                            "MaxFileSizeMB": 10
                        },
                        "Artifacts": [
                            {
                                "GetterSource": "http://foo.com/artifact.tar.gz",
                                "GetterOptions": {
                                    "checksum": "md5:c4aa853ad2215426eb7d70a21922e794"
                                },
                                "RelativeDest": "local/"
                            }
                        ]
                    }
                ],
                "RestartPolicy": {
                    "Interval": 300000000000,
                    "Attempts": 10,
                    "Delay": 25000000000,
                    "Mode": "delay"
                },
                "Meta": {
                    "foo": "bar",
                    "baz": "pipe"
                }
            }
        ],
        "Update": {
            "Stagger": 10000000000,
            "MaxParallel": 1
        },
        "Periodic": {
            "Enabled": true,
            "Spec": "* * * * *",
            "SpecType": "cron",
            "ProhibitOverlap": true
        },
        "Meta": {
            "foo": "bar",
            "baz": "pipe"
        }
    }
}
```

## Syntax Reference

Following is a syntax reference for the possible keys that are supported
and their default values if any for each type of object.

### Job

The `Job` object supports the following keys:

* `AllAtOnce` - Controls if the entire set of tasks in the job must
  be placed atomically or if they can be scheduled incrementally.
  This should only be used for special circumstances. Defaults to `false`.

* `Constraints` - A list to define additional constraints where a job can be
  run. See the constraint reference for more details.

* `Datacenters` - A list of datacenters in the region which are eligible
  for task placement. This must be provided, and does not have a default.

* `TaskGroups` - A list to define additional task groups. See the task group
  reference for more details.

* `Meta` - Annotates the job with opaque metadata.

* `Priority` - Specifies the job priority which is used to prioritize
  scheduling and access to resources. Must be between 1 and 100 inclusively,
  and defaults to 50.

* `Region` - The region to run the job in, defaults to "global".

* `Type` - Specifies the job type and switches which scheduler
  is used. Nomad provides the `service`, `system` and `batch` schedulers,
  and defaults to `service`. To learn more about each scheduler type visit
  [here](/docs/jobspec/schedulers.html)

*   `Update` - Specifies the task's update strategy. When omitted, rolling
    updates are disabled. The `Update` object supports the following attributes:

    * `MaxParallel` - `MaxParallel` is given as an integer value and specifies
      the number of tasks that can be updated at the same time.

    * `Stagger` - `Stagger` introduces a delay between sets of task updates and
      is given in nanoseconds.

    An example `Update` block:

    ```
    "Update": {
        "MaxParallel" : 3,
        "Stagger" : 10000000000
    }
    ```

*   `Periodic` - `Periodic` allows the job to be scheduled at fixed times, dates
    or intervals. The periodic expression is always evaluated in the UTC
    timezone to ensure consistent evaluation when Nomad Servers span multiple
    time zones. The `Periodic` object is optional and supports the following attributes:

    * `Enabled` - `Enabled` determines whether the periodic job will spawn child
    jobs.

    * `SpecType` - `SpecType` determines how Nomad is going to interpret the
      periodic expression. `cron` is the only supported `SpecType` currently.

    * `Spec` - A cron expression configuring the interval the job is launched
    at. Supports predefined expressions such as "@daily" and "@weekly" See
    [here](https://github.com/gorhill/cronexpr#implementation) for full
    documentation of supported cron specs and the predefined expressions.

    * <a id="prohibit_overlap">`ProhibitOverlap`</a> - `ProhibitOverlap` can
      be set to true to enforce that the periodic job doesn't spawn a new
      instance of the job if any of the previous jobs are still running. It is
      defaulted to false.

    An example `periodic` block:

    ```
        "Periodic": {
            "Spec": "*/15 * * * * *"
            "SpecType": "cron",
            "Enabled": true,
            "ProhibitOverlap": true
        }
    ```

### Task Group

`TaskGroups` is a list of `TaskGroup` objects, each supports the following
attributes:

* `Constraints` - This is a list of `Constraint` objects. See the constraint
  reference for more details.

* `Count` - Specifies the number of the task groups that should
  be running. Must be non-negative, defaults to one.

* `Meta` - A key/value map that annotates the task group with opaque metadata.

* `Name` - The name of the task group. Must be specified.

* `RestartPolicy` - Specifies the restart policy to be applied to tasks in this group.
  If omitted, a default policy for batch and non-batch jobs is used based on the
  job type. See the [restart policy reference](#restart_policy) for more details.

* `Tasks` - A list of `Task` object that are part of the task group.

### Task

The `Task` object supports the following keys:

* `Artifacts` - `Artifacts` is a list of `Artifact` objects which define
  artifacts to be downloaded before the task is run. See the artifacts
  reference for more details.

* `Config` - A map of key/value configuration passed into the driver
  to start the task. The details of configurations are specific to
  each driver.

* `Constraints` - This is a list of `Constraint` objects. See the constraint
  reference for more details.

* `Driver` - Specifies the task driver that should be used to run the
  task. See the [driver documentation](/docs/drivers/index.html) for what
  is available. Examples include `docker`, `qemu`, `java`, and `exec`.

*   `Env` - A map of key/value representing environment variables that
    will be passed along to the running process. Nomad variables are
    interpreted when set in the environment variable values. See the table of
    interpreted variables [here](/docs/jobspec/interpreted.html).

    For example the below environment map will be reinterpreted:

    ```
        "Env": {
            "NODE_CLASS" : "${nomad.class}"
        }
    ```

* `KillTimeout` - `KillTimeout` is a time duration in nanoseconds. It can be
  used to configure the time between signaling a task it will be killed and
  actually killing it. Drivers first sends a task the `SIGINT` signal and then
  sends `SIGTERM` if the task doesn't die after the `KillTimeout` duration has
  elapsed.

* `LogConfig` - This allows configuring log rotation for the `stdout` and `stderr`
  buffers of a Task. See the log rotation reference below for more details.

* `Meta` - Annotates the task group with opaque metadata.

* `Name` - The name of the task. This field is required.

* `Resources` - Provides the resource requirements of the task.
  See the resources reference for more details.

* `Services` - `Services` is a list of `Service` objects. Nomad integrates with
  Consul for service discovery. A `Service` object represents a routable and
  discoverable service on the network. Nomad automatically registers when a task
  is started and de-registers it when the task transitions to the dead state.
  [Click here](/docs/jobspec/servicediscovery.html) to learn more about
  services. Below is the fields in the `Service` object:

     * `Name`: Nomad automatically determines the name of a Task. By default the
       name of a service is `$(job-name)-$(task-group)-$(task-name)`. Users can
       explicitly name the service by specifying this option. If multiple
       services are defined for a Task then only one task can have the default
       name, all the services have to be explicitly named.  Users can add the
       following to the service names: `${JOB}`, `${TASKGROUP}`, `${TASK}`,
       `${BASE}`.  Nomad will replace them with the appropriate value of the
       Job, Task Group, and Task names while registering the Job. `${BASE}`
       expands to `${JOB}-${TASKGROUP}-${TASK}`.  Names must be adhere to
       [RFC-1123 §2.1](https://tools.ietf.org/html/rfc1123#section-2) and are
       limited to alphanumeric and hyphen characters (i.e. `[a-z0-9\-]`), and be
       less than 64 characters in length.

     * `Tags`: A list of string tags associated with this Service. String
       interpolation is supported in tags.

     * `PortLabel`: `PortLabel` is an optional string and is used to associate
       the port with the service.  If specified, the port label must match one
       defined in the resources block.  This could be a label to either a
       dynamic or a static port. If an incorrect port label is specified, Nomad
       doesn't register the IP:Port with Consul.

     * `Checks`: `Checks` is an array of check objects. A check object defines a
       health check associated with the service. Nomad supports the `script`,
       `http` and `tcp` Consul Checks. Script checks are not supported for the
       qemu driver since the Nomad client doesn't have access to the file system
       of a tasks using the Qemu driver.

         * `Type`:  This indicates the check types supported by Nomad. Valid
           options are currently `script`, `http` and `tcp`.

         * `Name`: The name of the health check.

         * `Interval`: This indicates the frequency of the health checks that
           Consul will perform.

         * `Timeout`: This indicates how long Consul will wait for a health
           check query to succeed.

         * `Path`:The path of the http endpoint which Consul will query to query
           the health of a service if the type of the check is `http`. Nomad
           will add the IP of the service and the port, users are only required
           to add the relative URL of the health check endpoint.

         * `Protocol`: This indicates the protocol for the http checks. Valid
           options are `http` and `https`. We default it to `http`

         * `Command`: This is the command that the Nomad client runs for doing
           script based health check.

         * `Args`: Additional arguments to the `command` for script based health
           checks.


* `User` - Set the user that will run the task. It defaults to the same user
  the Nomad client is being run as. This can only be set on Linux platforms.

### Resources

The `Resources` object supports the following keys:

* `CPU` - The CPU required in MHz.

* `DiskMB` - The disk required in MB.

* `IOPS` - The number of IOPS required given as a weight between 10-1000.

* `MemoryMB` - The memory required in MB.

* `Networks` - A list of network objects.

The Network object supports the following keys:

* `MBits` - The number of MBits in bandwidth required.

Nomad can allocate two types of ports to a task - Dynamic and Static/Reserved
ports. A network object allows the user to specify a list of `DynamicPorts` and
`ReservedPorts`. Each object supports the following attributes:

* `Value` - The port number for static ports. If the port is dynamic, then this
  attribute is ignored.
* `Label` - The label to annotate a port so that it can be referred in the
  service discovery block or environment variables.

<a id="restart_policy"></a>

### Restart Policy

The `RestartPolicy` object supports the following keys:

* `Attempts` - `Attempts` is the number of restarts allowed in an `Interval`.

* `Interval` - `Interval` is a time duration that is specified in nanoseconds.
  The `Interval` begins when the first task starts and ensures that only
  `Attempts` number of restarts happens within it. If more than `Attempts`
  number of failures happen, behavior is controlled by `Mode`.

* `Delay` - A duration to wait before restarting a task. It is specified in
  nanoseconds. A random jitter of up to 25% is added to the delay.

*   `Mode` - `Mode` is given as a string and controls the behavior when the task
    fails more than `Attempts` times in an `Interval`. Possible values are listed
    below:

    * `delay` - `delay` will delay the next restart until the next `Interval` is
      reached.

    * `fail` - `fail` will not restart the task again.

### Constraint

The `Constraint` object supports the following keys:

* `LTarget` - Specifies the attribute to examine for the
  constraint. See the table of attributes [here](/docs/jobspec/interpreted.html#interpreted_node_vars).

* `RTarget` - Specifies the value to compare the attribute against.
  This can be a literal value, another attribute or a regular expression if
  the `Operator` is in "regexp" mode.

* `Operand` - Specifies the test to be performed on the two targets. It takes on the
  following values:
  
  * `regexp` - Allows the `RTarget` to be a regular expression to be matched.

  * `distinct_host` - If set, the scheduler will not co-locate any task groups on the same
        machine. This can be specified as a job constraint which applies the
        constraint to all task groups in the job, or as a task group constraint which
        scopes the effect to just that group.

        Placing the constraint at both the job level and at the task group level is
        redundant since when placed at the job level, the constraint will be applied
        to all task groups. When specified, `LTarget` and `RTarget` should be
        omitted.

  * Comparison Operators - `=`, `==`, `is`, `!=`, `not`, `>`, `>=`, `<`, `<=`. The
    ordering is compared lexically.

### Log Rotation

The `LogConfig` object configures the log rotation policy for a task's `stdout` and
`stderr`. The `LogConfig` object supports the following attributes:

* `MaxFiles` - The maximum number of rotated files Nomad will retain for
  `stdout` and `stderr`, each tracked individually.

* `MaxFileSizeMB` - The size of each rotated file. The size is specified in
  `MB`.

If the amount of disk resource requested for the task is less than the total
amount of disk space needed to retain the rotated set of files, Nomad will return
a validation error when a job is submitted.

```
"LogConfig: {
    "MaxFiles": 3,
    "MaxFileSizeMB": 10
}
```

In the above example we have asked Nomad to retain 3 rotated files for both
`stderr` and `stdout` and size of each file is 10MB. The minimum disk space that
would be required for the task would be 60MB.

### Artifact

Nomad downloads artifacts using
[`go-getter`](https://github.com/hashicorp/go-getter). The `go-getter` library
allows downloading of artifacts from various sources using a URL as the input
source. The key/value pairs given in the `options` block map directly to
parameters appended to the supplied `source` URL. These are then used by
`go-getter` to appropriately download the artifact. `go-getter` also has a CLI
tool to validate its URL and can be used to check if the Nomad `artifact` is
valid.

Nomad allows downloading `http`, `https`, and `S3` artifacts. If these artifacts
are archives (zip, tar.gz, bz2, etc.), these will be unarchived before the task
is started.

The `Artifact` object supports the following keys:

* `GetterSource` - The path to the artifact to download.

* `RelativeDest` - An optional path to download the artifact into relative to the
  root of the task's directory. If omitted, it will default to `local/`.

* `GetterOptions` - A `map[string]string` block of options for `go-getter`.
  Full documentation of supported options are available
  [here](https://github.com/hashicorp/go-getter/tree/ef5edd3d8f6f482b775199be2f3734fd20e04d4a#protocol-specific-options-1).
  An example is given below:

```
"GetterOptions": {
    "checksum": "md5:c4aa853ad2215426eb7d70a21922e794",

    "aws_access_key_id": "<id>",
    "aws_access_key_secret": "<secret>",
    "aws_access_token": "<token>"
}
```

An example of downloading and unzipping an archive is as simple as:

```
"Artifacts": [
  {
    # The archive will be extracted before the task is run, making
    # it easy to ship configurations with your binary.
    "GetterSource": "https://example.com/my.zip",

    "GetterOptions": {
      "checksum": "md5:7f4b3e3b4dd5150d4e5aaaa5efada4c3"
    }
  }
]
```

#### S3 examples

S3 has several different types of addressing and more detail can be found
[here](http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingBucket.html#access-bucket-intro)

S3 region specific endpoints can be found
[here](http://docs.aws.amazon.com/general/latest/gr/rande.html#s3_region)

Path based style:
```
"Artifacts": [
  {
    "GetterSource": "https://s3-us-west-2.amazonaws.com/my-bucket-example/my_app.tar.gz",
  }
]
```

or to override automatic detection in the URL, use the S3-specific syntax
```
"Artifacts": [
  {
    "GetterSource": "s3::https://s3-eu-west-1.amazonaws.com/my-bucket-example/my_app.tar.gz",
  }
]
```

Virtual hosted based style
```
"Artifacts": [
  {
    "GetterSource": "my-bucket-example.s3-eu-west-1.amazonaws.com/my_app.tar.gz",
  }
]
```
