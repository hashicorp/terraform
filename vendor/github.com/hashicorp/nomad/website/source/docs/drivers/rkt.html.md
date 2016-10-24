---
layout: "docs"
page_title: "Drivers: Rkt"
sidebar_current: "docs-drivers-rkt"
description: |-
  The rkt task driver is used to run application containers using rkt.
---

# Rkt Driver - Experimental

Name: `rkt`

The `rkt` driver provides an interface for using CoreOS rkt for running
application containers. Currently, the driver supports launching containers but
does not support dynamic ports. This can lead to port conflicts and as such,
this driver is being marked as experimental and should be used with care.

## Task Configuration

The `rkt` driver supports the following configuration in the job spec:

* `image` - The image to run which may be specified by name, hash, ACI address
  or docker registry.

* `command` - (Optional) A command to execute on the ACI.

*   `args` - (Optional) A list of arguments to the optional `command`.
    References to environment variables or any [interpretable Nomad
    variables](/docs/jobspec/interpreted.html) will be interpreted
    before launching the task. For example:

    ```
        args = ["${nomad.datacenter}", "${MY_ENV}", ${meta.foo}"]
    ```

* `trust_prefix` - (Optional) The trust prefix to be passed to rkt. Must be
  reachable from the box running the nomad agent. If not specified, the image is
  run without verifying the image signature.

* `dns_servers` - (Optional) A list of DNS servers to be used in the containers

* `dns_search_domains` - (Optional) A list of DNS search domains to be used in
   the containers

* `debug` - (Optional) Enable rkt command debug option

## Task Directories

The `rkt` driver currently does not support mounting of the `alloc/` and `local/` directory. 
Once support is added, version `v0.10.0` or above of `rkt` will be required. 

## Client Requirements

The `rkt` driver requires rkt to be installed and in your systems `$PATH`.
The `trust_prefix` must be accessible by the node running Nomad. This can be an
internal source, private to your cluster, but it must be reachable by the client
over HTTP.

## Client Attributes

The `rkt` driver will set the following client attributes:

* `driver.rkt` - Set to `1` if rkt is found on the host node. Nomad determines
this by executing `rkt version` on the host and parsing the output
* `driver.rkt.version` - Version of `rkt` eg: `0.8.1`. Note that the minimum required
version is `0.14.0`
* `driver.rkt.appc.version` - Version of `appc` that `rkt` is using eg: `0.8.1`

## Resource Isolation

This driver supports CPU and memory isolation by delegating to `rkt`. Network isolation
is not supported as of now.
