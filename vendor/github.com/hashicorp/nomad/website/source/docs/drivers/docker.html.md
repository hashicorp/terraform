---
layout: "docs"
page_title: "Drivers: Docker"
sidebar_current: "docs-drivers-docker"
description: |-
  The Docker task driver is used to run Docker based tasks.
---

# Docker Driver

Name: `docker`

The `docker` driver provides a first-class Docker workflow on Nomad. The Docker
driver handles downloading containers, mapping ports, and starting, watching,
and cleaning up after containers.

## Task Configuration

The `docker` driver is configured via a `config` block:

```
task "webservice" {
    driver = "docker"
    config = {
        image = "redis"
        labels = {
            group = "webservice-cache"
        }
    }
}
```

The following options are available for use in the job specification.

* `image` - The Docker image to run. The image may include a tag or custom URL and should include `https://` if required.
  By default it will be fetched from Docker Hub.

* `load` - (Optional) A list of paths to image archive files. If
  this key is not specified, Nomad assumes the `image` is hosted on a repository
  and attempts to pull the image. The `artifact` blocks can be specified to
  download each of the archive files. The equivalent of `docker load -i path`
  would be run on each of the archive files.

* `command` - (Optional) The command to run when starting the container.

* `args` - (Optional) A list of arguments to the optional `command`. If no
  `command` is present, `args` are ignored. References to environment variables
  or any [interpretable Nomad variables](/docs/jobspec/interpreted.html) will be
  interpreted before launching the task. For example:

  ```
  args = ["${nomad.datacenter}", "${MY_ENV}", "${meta.foo}"]
  ```

* `labels` - (Optional) A key/value map of labels to set to the containers on
  start.

* `privileged` - (Optional) `true` or `false` (default). Privileged mode gives
  the container access to devices on the host. Note that this also requires the
  nomad agent and docker daemon to be configured to allow privileged
  containers.

* `ipc_mode` - (Optional) The IPC mode to be used for the container. The default
  is `none` for a private IPC namespace. Other values are `host` for sharing
  the host IPC namespace or the name or id of an existing container. Note that
  it is not possible to refer to Nomad started Docker containers since their
  names are not known in advance. Note that setting this option also requires the
  Nomad agent to be configured to allow privileged containers.

* `pid_mode` - (Optional) `host` or not set (default). Set to `host` to share
  the PID namespace with the host. Note that this also requires the Nomad agent
  to be configured to allow privileged containers.

* `uts_mode` - (Optional) `host` or not set (default). Set to `host` to share
  the UTS namespace with the host. Note that this also requires the Nomad agent
  to be configured to allow privileged containers.

* `network_mode` - (Optional) The network mode to be used for the container. In
  order to support userspace networking plugins in Docker 1.9 this accepts any
  value. The default is `bridge` for all operating systems but Windows, which
  defaults to `nat`. Other networking modes may not work without additional
  configuration on the host (which is outside the scope of Nomad).  Valid values
  pre-docker 1.9 are `default`, `bridge`, `host`, `none`, or `container:name`.
  See below for more details.

* `hostname` - (Optional) The hostname to assign to the container. When
  launching more than one of a task (using `count`) with this option set, every
  container the task starts will have the same hostname.

* `dns_servers` - (Optional) A list of DNS servers for the container to use
  (e.g. ["8.8.8.8", "8.8.4.4"]). *Docker API v1.10 and above only*

* `dns_search_domains` - (Optional) A list of DNS search domains for the container
  to use.

* `SSL` - (Optional) If this is set to true, Nomad uses SSL to talk to the
  repository. The default value is `true`.

* `port_map` - (Optional) A key/value map of port labels (see below).

* `auth` - (Optional) Provide authentication for a private registry (see below).

* `tty` - (Optional) `true` or `false` (default). Allocate a pseudo-TTY for the
  container.

* `interactive` - (Optional) `true` or `false` (default). Keep STDIN open on
  the container.
  
* `shm_size` - (Optional) The size (bytes) of /dev/shm for the container.

* `work_dir` - (Optional) The working directory inside the container.

### Container Name

Nomad creates a container after pulling an image. Containers are named
`{taskName}-{allocId}`. This is necessary in order to place more than one
container from the same task on a host (e.g. with count > 1). This also means
that each container's name is unique across the cluster.

This is not configurable.

### Authentication

If you want to pull from a private repo (for example on dockerhub or quay.io),
you will need to specify credentials in your job via the `auth` option.

The `auth` object supports the following keys:

* `username` - (Optional) The account username.

* `password` - (Optional) The account password.

* `email` - (Optional) The account email.

* `server_address` - (Optional) The server domain/IP without the protocol.
  Docker Hub is used by default.

Example:

```
task "secretservice" {
    driver = "docker"

    config {
        image = "secret/service"

        auth {
            username = "dockerhub_user"
            password = "dockerhub_password"
        }
    }
}
```

**Please note that these credentials are stored in Nomad in plain text.**
Secrets management will be added in a later release.

## Networking

Docker supports a variety of networking configurations, including using host
interfaces, SDNs, etc. Nomad uses `bridged` networking by default, like Docker.

You can specify other networking options, including custom networking plugins
in Docker 1.9. **You may need to perform additional configuration on the host
in order to make these work.** This additional configuration is outside the
scope of Nomad.

### Allocating Ports

You can allocate ports to your task using the port syntax described on the
[networking page](/docs/jobspec/networking.html). Here is a recap:

```
task "webservice" {
    driver = "docker"

    resources {
        network {
            port "http" {}
            port "https" {}
        }
    }
}
```

### Forwarding and Exposing Ports

A Docker container typically specifies which port a service will listen on by
specifying the `EXPOSE` directive in the `Dockerfile`.

Because dynamic ports will not match the ports exposed in your Dockerfile,
Nomad will automatically expose all of the ports it allocates to your
container.

These ports will be identified via environment variables. For example:

```
port "http" {}
```

If Nomad allocates port `23332` to your task for `http`, `23332` will be
automatically exposed and forwarded to your container, and the driver will set
an environment variable `NOMAD_PORT_http` with the value `23332` that you can
read inside your container.

This provides an easy way to use the `host` networking option for better
performance.

### Using the Port Map

If you prefer to use the traditional port-mapping method, you can specify the
`port_map` option in your job specification. It looks like this:

```
task "redis" {
    driver = "docker"

    resources {
        network {
            mbits = 20
            port "redis" {}
        }
    }

    config {
      image = "redis"

      port_map {
        redis = 6379
      }
    }
}
```

If Nomad allocates port `23332` to your task, the Docker driver will
automatically setup the port mapping from `23332` on the host to `6379` in your
container, so it will just work!

Note that by default this only works with `bridged` networking mode. It may
also work with custom networking plugins which implement the same API for
expose and port forwarding.

### Networking Protocols

The Docker driver configures ports on both the `tcp` and `udp` protocols.

This is not configurable.

### Other Networking Modes

Some networking modes like `container` or `none` will require coordination
outside of Nomad. First-class support for these options may be improved later
through Nomad plugins or dynamic job configuration.

## Host Requirements

Nomad requires Docker to be installed and running on the host alongside the
Nomad agent. Nomad was developed against Docker `1.8.2` and `1.9`.

By default Nomad communicates with the Docker daemon using the daemon's unix
socket. Nomad will need to be able to read/write to this socket. If you do not
run Nomad as root, make sure you add the Nomad user to the Docker group so
Nomad can communicate with the Docker daemon.

For example, on Ubuntu you can use the `usermod` command to add the `vagrant`
user to the `docker` group so you can run Nomad without root:

    sudo usermod -G docker -a vagrant

For the best performance and security features you should use recent versions
of the Linux Kernel and Docker daemon.

## Agent Configuration

The `docker` driver has the following [client configuration
options](/docs/agent/config.html#options):

* `docker.endpoint` - Defaults to `unix:///var/run/docker.sock`. You will need
  to customize this if you use a non-standard socket (http or another
  location).

* `docker.auth.config` - Allows an operator to specify a json file which is in
  the dockercfg format containing authentication information for private registry.

* `docker.tls.cert` - Path to the server's certificate file (`.pem`). Specify
  this along with `docker.tls.key` and `docker.tls.ca` to use a TLS client to
  connect to the docker daemon. `docker.endpoint` must also be specified or
  this setting will be ignored.

* `docker.tls.key` - Path to the client's private key (`.pem`). Specify this
  along with `docker.tls.cert` and `docker.tls.ca` to use a TLS client to
  connect to the docker daemon. `docker.endpoint` must also be specified or
  this setting will be ignored.

* `docker.tls.ca` - Path to the server's CA file (`.pem`). Specify this along
  with `docker.tls.cert` and `docker.tls.key` to use a TLS client to connect to
  the docker daemon. `docker.endpoint` must also be specified or this setting
  will be ignored.

* `docker.cleanup.image` Defaults to `true`. Changing this to `false` will
  prevent Nomad from removing images from stopped tasks.

* `docker.volumes.selinuxlabel`: Allows the operator to set a SELinux
  label to the allocation and task local bind-mounts to containers.

* `docker.privileged.enabled` Defaults to `false`. Changing this to `true` will
  allow containers to use `privileged` mode, which gives the containers full
  access to the host's devices. Note that you must set a similar setting on the
  Docker daemon for this to work.

Note: When testing or using the `-dev` flag you can use `DOCKER_HOST`,
`DOCKER_TLS_VERIFY`, and `DOCKER_CERT_PATH` to customize Nomad's behavior. If
`docker.endpoint` is set Nomad will **only** read client configuration from the
config file.

An example is given below: 

```
    client {
        options = {
            "docker.cleanup.image" = "false"
        }
    }
```

## Agent Attributes

The `docker` driver will set the following client attributes:

* `driver.docker` - This will be set to "1", indicating the driver is
  available.
* `driver.docker.version` - This will be set to version of the docker server

## Resource Isolation

### CPU

Nomad limits containers' CPU based on CPU shares. CPU shares allow containers
to burst past their CPU limits. CPU limits will only be imposed when there is
contention for resources. When the host is under load your process may be
throttled to stabilize QOS depending on how many shares it has. You can see how
many CPU shares are available to your process by reading `NOMAD_CPU_LIMIT`.
1000 shares are approximately equal to 1Ghz.

Please keep the implications of CPU shares in mind when you load test workloads
on Nomad.

### Memory

Nomad limits containers' memory usage based on total virtual memory. This means
that containers scheduled by Nomad cannot use swap. This is to ensure that a
swappy process does not degrade performance for other workloads on the same
host.

Since memory is not an elastic resource, you will need to make sure your
container does not exceed the amount of memory allocated to it, or it will be
terminated or crash when it tries to malloc. A process can inspect its memory
limit by reading `NOMAD_MEMORY_LIMIT`, but will need to track its own memory
usage. Memory limit is expressed in megabytes so 1024 = 1Gb.

### IO

Nomad's Docker integration does not currently provide QOS around network or
filesystem IO. These will be added in a later release.

### Security

Docker provides resource isolation by way of
[cgroups and namespaces](https://docs.docker.com/introduction/understanding-docker/#the-underlying-technology).
Containers essentially have a virtual file system all to themselves. If you
need a higher degree of isolation between processes for security or other
reasons, it is recommended to use full virtualization like
[QEMU](/docs/drivers/qemu.html).
