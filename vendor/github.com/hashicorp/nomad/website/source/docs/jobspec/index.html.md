---
layout: "docs"
page_title: "Job Specification"
sidebar_current: "docs-jobspec-syntax"
description: |-
  Learn about the Job specification used to submit jobs to Nomad.
---

# Job Specification

Jobs can be specified either in [HCL](https://github.com/hashicorp/hcl) or JSON.
HCL is meant to strike a balance between human readable and editable, and machine-friendly.

For machine-friendliness, Nomad can also read JSON configurations. In general, we recommend
using the HCL syntax.

## HCL Syntax

For a detailed description of HCL general syntax, [see this guide](https://github.com/hashicorp/hcl#syntax).
Here we cover the details of the Job specification for Nomad:

```
# Define a job called my-service
job "my-service" {
    # Job should run in the US region
    region = "us"

    # Spread tasks between us-west-1 and us-east-1
    datacenters = ["us-west-1", "us-east-1"]

    # run this with service scheduler
    type = "service"

    # Rolling updates should be sequential
    update {
        stagger = "30s"
        max_parallel = 1
    }

    group "webs" {
        # We want 5 web servers
        count = 5

        # Create a web front end using a docker image
        task "frontend" {
            driver = "docker"
            config {
                image = "hashicorp/web-frontend"
            }
            service {
                port = "http"
                check {
                    type = "http"
                    path = "/health"
                    interval = "10s"
                    timeout = "2s"
                }
            }
            env {
                DB_HOST = "db01.example.com"
                DB_USER = "web"
                DB_PASSWORD = "loremipsum"
            }
            resources {
                cpu = 500
                memory = 128
                network {
                    mbits = 100
                    # Request for a dynamic port
                    port "http" {
                    }
                    # Request for a static port
                    port "https" {
                        static = 443
                    }
                }
            }
        }
    }
}
```

This is a fairly simple example job, but demonstrates many of the features and syntax
of the job specification. The primary "objects" are the job, task group, and task.
Each job file has only a single job, however a job may have multiple task groups,
and each task group may have multiple tasks. Task groups are a set of tasks that
must be co-located on a machine. Groups with a single task and count of one
can be declared outside of a group which is created implicitly.

Constraints can be specified at the job, task group, or task level to restrict
where a task is eligible for running. An example constraint looks like:

```
# Restrict to only nodes running linux
constraint {
    attribute = "${attr.kernel.name}"
    value = "linux"
}
```

Jobs can also specify additional metadata at the job, task group, or task level.
This metadata is opaque to Nomad and can be used for any purpose, including
defining constraints on the metadata. Metadata can be specified by:

```
# Setup ELB via metadata and setup foo
meta {
    foo = "bar"
    elb_mode = "tcp"
    elb_check_interval = "10s"
}
```

## Syntax Reference

Following is a syntax reference for the possible keys that are supported
and their default values if any for each type of object.

### Job

The `job` object supports the following keys:

* `all_at_once` - Controls if the entire set of tasks in the job must
  be placed atomically or if they can be scheduled incrementally.
  This should only be used for special circumstances. Defaults to `false`.

* `constraint` - This can be provided multiple times to define additional
  constraints. See the constraint reference for more details.

* `datacenters` - A list of datacenters in the region which are eligible
  for task placement. This must be provided, and does not have a default.

* `group` - This can be provided multiple times to define additional
  task groups. See the task group reference for more details.

* `meta` - Annotates the job with opaque metadata.

* `priority` - Specifies the job priority which is used to prioritize
  scheduling and access to resources. Must be between 1 and 100 inclusively,
  with a larger value corresponding to a higher priority. Defaults to 50.

* `region` - The region to run the job in, defaults to "global".

* `task` - This can be specified multiple times to add a task as
  part of the job. Tasks defined directly in a job are wrapped in
  a task group of the same name.

* `type` - Specifies the job type and switches which scheduler
  is used. Nomad provides the `service`, `system` and `batch` schedulers,
  and defaults to `service`. To learn more about each scheduler type visit
  [here](/docs/jobspec/schedulers.html)

<a id="update"></a>

*   `update` - Specifies the task's update strategy. When omitted, rolling
    updates are disabled. The `update` block supports the following keys:

    * `max_parallel` - `max_parallel` is given as an integer value and specifies
      the number of tasks that can be updated at the same time.

    * `stagger` - `stagger` introduces a delay between sets of task updates and
      is given as an as a time duration. If stagger is provided as an integer,
      seconds are assumed. Otherwise the "s", "m", and "h" suffix can be used,
      such as "30s".

    An example `update` block:

    ```
    update {
        // Update 3 tasks at a time.
        max_parallel = 3

        // Wait 30 seconds between updates.
        stagger = "30s"
    }
    ```

*   `periodic` - `periodic` allows the job to be scheduled at fixed times, dates
    or intervals. The periodic expression is always evaluated in the UTC
    timezone to ensure consistent evaluation when Nomad Servers span multiple
    time zones. The `periodic` block is optional and supports the following keys:

    * `enabled` - `enabled` determines whether the periodic job will spawn child
    jobs. `enabled` is defaulted to true if the block is included.

    * `cron` - A cron expression configuring the interval the job is launched
    at. Supports predefined expressions such as "@daily" and "@weekly" See
    [here](https://github.com/gorhill/cronexpr#implementation) for full
    documentation of supported cron specs and the predefined expressions.

    * <a id="prohibit_overlap">`prohibit_overlap`</a> - `prohibit_overlap` can
      be set to true to enforce that the periodic job doesn't spawn a new
      instance of the job if any of the previous jobs are still running. It is
      defaulted to false.

    An example `periodic` block:

    ```
        periodic {
            // Launch every 15 minutes
            cron = "*/15 * * * * *"

            // Do not allow overlapping runs.
            prohibit_overlap = true
        }
    ```

### Task Group

The `group` object supports the following keys:

* `count` - Specifies the number of the task groups that should
  be running. Must be non-negative, defaults to one.

* `constraint` - This can be provided multiple times to define additional
  constraints. See the constraint reference for more details.

* `restart` - Specifies the restart policy to be applied to tasks in this group.
  If omitted, a default policy for batch and non-batch jobs is used based on the
  job type. See the [restart policy reference](#restart_policy) for more details.

* `task` - This can be specified multiple times, to add a task as
  part of the group.

* `meta` - A key/value map that annotates the task group with opaque metadata.

### Task

The `task` object supports the following keys:

* `driver` - Specifies the task driver that should be used to run the
  task. See the [driver documentation](/docs/drivers/index.html) for what
  is available. Examples include `docker`, `qemu`, `java`, and `exec`.

* `user` - Set the user that will run the task. It defaults to the same user
  the Nomad client is being run as. This can only be set on Linux platforms.

* `constraint` - This can be provided multiple times to define additional
  constraints. See the constraint reference for more details.

* `config` - A map of key/value configuration passed into the driver
  to start the task. The details of configurations are specific to
  each driver.

* `service` - Nomad integrates with Consul for service discovery. A service
  block represents a routable and discoverable service on the network. Nomad
  automatically registers when a task is started and de-registers it when the
  task transitions to the dead state. [Click
  here](/docs/jobspec/servicediscovery.html) to learn more about services.

*   `env` - A map of key/value representing environment variables that
    will be passed along to the running process. Nomad variables are
    interpreted when set in the environment variable values. See the table of
    interpreted variables [here](/docs/jobspec/interpreted.html).

    For example the below environment map will be reinterpreted:

    ```
        env {
            // The value will be interpreted by the client and set to the
            // correct value.
            NODE_CLASS = "${nomad.class}"
        }
    ```

* `resources` - Provides the resource requirements of the task.
  See the [resources reference](#resources) for more details.

* `meta` - Annotates the task group with opaque metadata.

<a id="kill_timeout"></a>

* `kill_timeout` - `kill_timeout` is a time duration that can be specified using
  the `s`, `m`, and `h` suffixes, such as `30s`. It can be used to configure the
  time between signaling a task it will be killed and actually killing it. Nomad
  sends an `os.Interrupt` which on Unix systems is defined as `SIGINT`. After
  the timeout a kill signal is sent (on Unix `SIGKILL`).

* `logs` - Logs allows configuring log rotation for the `stdout` and `stderr`
  buffers of a Task. See the [log rotation section](#log_rotation) for more details.

* `artifact` - Defines an artifact to be downloaded before the task is run. This
  can be provided multiple times to define additional artifacts to download. See
  the artifacts reference for more details.

### Resources

The `resources` object supports the following keys:

* `cpu` - The CPU required in MHz. Defaults to `100`.

* `disk` - The disk required in MB. Defaults to `200`.

* `iops` - The number of IOPS required given as a weight between 10-1000. Defaults to `0`.

* `memory` - The memory required in MB. Defaults to `300`.

* `network` - The network required. Details below.

The `network` object supports the following keys:

* `mbits` (required) - The number of MBits in bandwidth required.

*   `port` - `port` is a repeatable object that can be used to specify both
    dynamic ports and reserved ports. It has the following format:

    ```
    port "label" {
        // If the `static` field is omitted, a dynamic port will be assigned.
        static = 6539
    }
    ```

<a id="restart_policy"></a>

### Restart Policy

The `restart` object supports the following keys:

* `attempts` - `attempts` is the number of restarts allowed in an `interval`.

* `interval` - `interval` is a time duration that can be specified using the
  `s`, `m`, and `h` suffixes, such as `30s`.  The `interval` begins when the
  first task starts and ensures that only `attempts` number of restarts happens
  within it. If more than `attempts` number of failures happen, behavior is
  controlled by `mode`.

* `delay` - A duration to wait before restarting a task. It is specified as a
  time duration using the `s`, `m`, and `h` suffixes, such as `30s`. A random
  jitter of up to 25% is added to the delay.

*   `mode` - Controls the behavior when the task fails more than `attempts`
    times in an interval. Possible values are listed below:

    * `delay` - `delay` will delay the next restart until the next `interval` is
      reached.

    * `fail` - `fail` will not restart the task again.

The default `batch` restart policy is:

```
restart {
    attempts = 15
    delay = "15s"
    interval = "168h" # 7 days
    mode = "delay"
}
```

The default non-batch restart policy is:

```
restart {
    interval = "1m"
    attempts = 2
    delay = "15s"
    mode = "delay"
}
```

### Constraint

The `constraint` object supports the following keys:

* `attribute` - Specifies the attribute to examine for the
  constraint. See the table of attributes [here](/docs/jobspec/interpreted.html#interpreted_node_vars).

*   `operator` - Specifies the comparison operator. Defaults to equality,
    and can be `=`, `==`, `is`, `!=`, `not`, `>`, `>=`, `<`, `<=`. The
    ordering is compared lexically. The following are equivalent:

      * `=`, `==` and `is`
      * `!=` and `not`

* `value` - Specifies the value to compare the attribute against.
  This can be a literal value or another attribute.

* `version` - Specifies a version constraint against the attribute.
  This sets the operator to `version` and the `value` to what is
  specified. This supports a comma separated list of constraints,
  including the pessimistic operator. See the
  [go-version](https://github.com/hashicorp/go-version) repository
  for examples.

* `regexp` - Specifies a regular expression constraint against
  the attribute. This sets the operator to "regexp" and the `value`
  to the regular expression.

*   `distinct_hosts` - `distinct_hosts` accepts a boolean value and defaults to
    `false`. If set, the scheduler will not co-locate any task groups on the same
    machine. This can be specified as a job constraint which applies the
    constraint to all task groups in the job, or as a task group constraint which
    scopes the effect to just that group.

    Placing the constraint at both the job level and at the task group level is
    redundant since when placed at the job level, the constraint will be applied
    to all task groups.

<a id="log_rotation"></a>

### Log Rotation

The `logs` object configures the log rotation policy for a task's `stdout` and
`stderr`. The `logs` object supports the following keys:

* `max_files` - The maximum number of rotated files Nomad will retain for
  `stdout` and `stderr`, each tracked individually.

* `max_file_size` - The size of each rotated file. The size is specified in
  `MB`.

If the amount of disk resource requested for the task is less than the total
amount of disk space needed to retain the rotated set of files, Nomad will return
a validation error when a job is submitted.

```
logs {
    max_files = 3
    max_file_size = 10
}
```

In the above example we have asked Nomad to retain 3 rotated files for both
`stderr` and `stdout` and size of each file is 10MB. The minimum disk space that
would be required for the task would be 60MB.

<a id="artifact_doc"></a>

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

The `artifact` object supports the following keys:

* `source` - The path to the artifact to download.

* `destination` - An optional path to download the artifact into relative to the
  root of the task's directory. If the `destination` key is omitted, it will
  default to `local/`.

* `options` - The `options` block allows setting parameters for `go-getter`.
  Full documentation of supported options are available
  [here](https://github.com/hashicorp/go-getter/tree/ef5edd3d8f6f482b775199be2f3734fd20e04d4a#protocol-specific-options-1).
  An example is given below:

```
options {
    # Validate the downloaded artifact
    checksum = "md5:c4aa853ad2215426eb7d70a21922e794"

    # S3 options for downloading artifacts from S3
    aws_access_key_id     = "<id>"
    aws_access_key_secret = "<secret>"
    aws_access_token      = "<token>"
}
```

An example of downloading and unzipping an archive is as simple as:

```
artifact {
  # The archive will be extracted before the task is run, making
  # it easy to ship configurations with your binary.
  source = "https://example.com/my.zip"

  options {
    checksum = "md5:7f4b3e3b4dd5150d4e5aaaa5efada4c3"
  }
}
```

#### S3 examples

S3 has several different types of addressing and more detail can be found
[here](http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingBucket.html#access-bucket-intro)

S3 region specific endpoints can be found
[here](http://docs.aws.amazon.com/general/latest/gr/rande.html#s3_region)

Path based style:
```
artifact {
  source = "https://s3-us-west-2.amazonaws.com/my-bucket-example/my_app.tar.gz"
}
```

or to override automatic detection in the URL, use the S3-specific syntax
```
artifact {
  source = "s3::https://s3-eu-west-1.amazonaws.com/my-bucket-example/my_app.tar.gz"
}
```

Virtual hosted based style
```
artifact {
  source = "my-bucket-example.s3-eu-west-1.amazonaws.com/my_app.tar.gz"
}
```

## JSON Syntax

Job files can also be specified in JSON. The conversion is straightforward
and self-documented. The downsides of JSON are less human readability and
the lack of comments. Otherwise, the two are completely interoperable.

See the [JSON API documentation](/docs/jobspec/json.html) for more details on
the JSON structure.
