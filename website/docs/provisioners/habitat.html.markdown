---
layout: "docs"
page_title: "Provisioner: habitat"
sidebar_current: "docs-provisioners-habitat"
description: |-
  The `habitat` provisioner installs the Habitat supervisor, and loads configured services.
---

# Habitat Provisioner

The `habitat` provisioner installs the [Habitat](https://habitat.sh) supervisor and loads configured services. This provisioner only supports Linux targets using the `ssh` connection type at this time.
## Requirements

The `habitat` provisioner has some prerequisites for specific connection types:

- For `ssh` type connections, we assume a few tools to be available on the remote host:
  * `curl`
  * `tee`
  * `setsid` - Only if using the `unmanaged` service type.

Without these prerequisites, your provisioning execution will fail.

## Example usage

```hcl
resource "aws_instance" "redis" {
  count = 3

  provisioner "habitat" {
    peers = [aws_instance.redis[0].private_ip]
    use_sudo = true
    service_type = "systemd"

    service {
      name = "core/redis"
      topology = "leader"
      user_toml = file("conf/redis.toml")
    }
  }
}

```

## Argument Reference

There are 2 configuration levels, `supervisor` and `service`.  Configuration placed directly within the `provisioner` block are supervisor configurations, and a provisioner can define zero or more services to run, and each service will have a `service` block within the `provisioner`.  A `service` block can also contain zero or more `bind` blocks to create service group bindings.

### Supervisor Arguments
* `version (string)` - (Optional) The Habitat version to install on the remote machine.  If not specified, the latest available version is used.
* `license (string)` - (Optional) Sets the acceptance of Chef licensing (one of `accept-no-persist` or `accept`): https://www.chef.io/end-user-license-agreement/
* `auto_update (bool)` - (Optional) If set to `true`, the supervisor will auto-update itself as soon as new releases are available on the specified `channel`.
* `http_disable (bool)` - (Optional) If set to `true`, disables the supervisor HTTP listener entirely.
* `peer (string)` - (Optional, deprecated) IP addresses or FQDN's for other Habitat supervisors to peer with, like: `--peer 1.2.3.4 --peer 5.6.7.8`. (Defaults to none)
* `peers (array)` - (Optional) A list of IP or FQDN's of other supervisor instance(s) to peer with. (Defaults to none)
* `service_type (string)` - (Optional) Method used to run the Habitat supervisor.  Valid options are `unmanaged` and `systemd`.  (Defaults to `systemd`)
* `service_name (string)` - (Optional) The name of the Habitat supervisor service, if using an init system such as `systemd`. (Defaults to `hab-supervisor`)
* `use_sudo (bool)` - (Optional) Use `sudo` when executing remote commands.  Required when the user specified in the `connection` block is not `root`.  (Defaults to `true`)
* `permanent_peer (bool)` - (Optional) Marks this supervisor as a permanent peer.  (Defaults to false)
* `listen_ctl (string)` - (Optional) The listen address for the countrol gateway system (Defaults to 127.0.0.1:9632)
* `listen_gossip (string)` - (Optional) The listen address for the gossip system (Defaults to 0.0.0.0:9638)
* `listen_http (string)` - (Optional) The listen address for the HTTP gateway (Defaults to 0.0.0.0:9631)
* `ring_key (string)` - (Optional) The name of the ring key for encrypting gossip ring communication (Defaults to no encryption)
* `ring_key_content (string)` - (Optional) The key content.  Only needed if using ring encryption and want the provisioner to take care of uploading and importing it.  Easiest to source from a file (eg `ring_key_content = "${file("conf/foo-123456789.sym.key")}"`) (Defaults to none)
* `ctl_secret (string)` - (Optional) Specify a secret to use (from `hab sup secret generate`) for control gateway communication between hab client(s) and the supervisor.  (Defaults to none)
* `url (string)` - (Optional) The URL of a Builder service to download packages and receive updates from.  (Defaults to https://bldr.habitat.sh)
* `channel (string)` - (Optional) The release channel in the Builder service to use. (Defaults to `stable`)
* `events (string)` - (Optional) Name of the service group running a Habitat EventSrv to forward Supervisor and service event data to. (Defaults to none)
* `organization (string)` - (Optional) The organization that the Supervisor and it's subsequent services are part of. (Defaults to `default`)
* `gateway_auth_token (string)` - (Optional) The http gateway authorization token (Defaults to none)
* `builder_auth_token (string)` - (Optional) The builder authorization token when using a private origin. (Defaults to none)

### Service Arguments
* `name (string)` - (Required) The Habitat package identifier of the service to run. (ie `core/haproxy` or `core/redis/3.2.4/20171002182640`)
* `binds (array)` - (Optional) An array of bind specifications. (ie `binds = ["backend:nginx.default"]`)
* `bind` - (Optional) An alternative way of declaring binds.  This method can be easier to deal with when populating values from other values or variable inputs without having to do string interpolation. The following example is equivalent to `binds = ["backend:nginx.default"]`:

```hcl
bind {
  alias = "backend"
  service = "nginx"
  group = "default"
}
```
* `topology (string)` - (Optional) Topology to start service in. Possible values `standalone` or `leader`.  (Defaults to `standalone`)
* `strategy (string)` - (Optional) Update strategy to use. Possible values `at-once`, `rolling` or `none`.  (Defaults to `none`)
* `user_toml (string)` - (Optional) TOML formatted user configuration for the service. Easiest to source from a file (eg `user_toml = "${file("conf/redis.toml")}")`.  (Defaults to none)
* `channel (string)` - (Optional) The release channel in the Builder service to use. (Defaults to `stable`)
* `group (string)` - (Optional) The service group to join.  (Defaults to `default`)
* `url (string)` - (Optional) The URL of a Builder service to download packages and receive updates from.  (Defaults to https://bldr.habitat.sh)
* `application (string)` - (Optional) The application name.  (Defaults to none)
* `environment (string)` - (Optional) The environment name.  (Defaults to none)
* `service_key (string)` - (Optional) The key content of a service private key, if using service group encryption.  Easiest to source from a file (eg `service_key = "${file("conf/redis.default@org-123456789.box.key")}"`) (Defaults to none)
