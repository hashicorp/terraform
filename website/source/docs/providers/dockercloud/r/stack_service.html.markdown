---
layout: "dockercloud"
page_title: "Docker Cloud: dockercloud_stack_service"
sidebar_current: "docs-dockercloud-resource-stack-service"
description: |-
  Provides a Docker Cloud stack service resource.
---

# dockercloud\_stack\_service

Provides a Docker Cloud stack service resource.

## Example Usage

```
# Create a new node cluster
resource "dockercloud_node_cluster" "default" {
    name = "dev"
    node_provider = "aws"
    region = "us-east-1"
    size = "t2.micro"
}

# Create a stack
resource "dockercloud_stack" "web" {
    name = "web"
    depends_on = ["dockercloud_node_cluster.default"]
}

# Create a sample web service
resource "dockercloud_stack_service" "web" {
    stack_uri = "${dockercloud_stack.web.uri}"
    name = "web_server"
    image = "python:3.2"
    entrypoint = "python -m http.server"

    # Explicitly set dependency on the node cluster
    depends_on = ["dockercloud_node_cluster.default"]
}
```

## Argument Reference

The following arguments are supported:

* `stack_uri` - (Required) The URI of the stack to create the service in
* `name` - (Required) The name of the service (cannot contain underscores).
* `image` - (Required) The image to start the container with.
* `entrypoint` - (Optional) The entrypoint used at container startup.
* `command` - (Optional) The command used at container startup.
* `workdir` - (Optional) The directory to start from at container startup.
* `pid` - (Optional) The process ID to run the command as (either `none` or `host`).
* `net` - (Optional) The networking type of the container (either `bridge` or `host`).
* `deployment_strategy` - (Optional) Container distribution among nodes. Must be one of `EMPTIEST_NODE`, `HIGH_AVAILABILITY` or `EVERY_NODE`.
* `autorestart` - (Optional) Whether the containers for this service should be restarted if they stop. Must be one of `OFF`, `ON_FAILURE` or `ALWAYS`.
* `autodestroy` - (Optional) Whether the containers should be terminated if they stop. Must be one of `OFF`, `ON_SUCCESS` or `ALWAYS`.
* `autoredeploy` - (Optional, bool) Whether to redeploy the containers of the service when its image is updated in Docker Cloud registry.
* `privileged` - (Optional, bool) Run container in privileged mode.
* `sequential_deployment` - (Optional, bool) Whether the containers should be launched and scaled in sequence.
* `container_count` - (Optional) Number of containers to start. When `deployment_strategy` is set to anything other than `EMPTIEST_NODE`, `container_count` cannot be set.
* `roles` - (Optional) List of Docker Cloud API roles to grant the service. Currently, only `global` is supported.
* `tags` - (Optional) List of tags to be used to deploy the service.
* `bindings` - (Optional, block) Bindings this service has to mount. See [Bindings](#bindgins) below for details.
* `env` - (Optional, block) Environment variables to be added in the containers on launch. See [Env](#env) below for details.
* `links` - (Optional, block) Other services to link this service to. See [Links](#links) below for details.
* `ports` - (Optional, block) Port information to be published in the containers for this service. See [Ports](#ports) below for details.
* `redeploy_on_change` - (Optional, bool) When a non-destructive config update is applied to the service, redeploy the running containers with the new configuration (default: `false`).
* `reuse_existing_volumes` - (Optional, bool) Reuse container volumes when redeploying.

<a id="bindings"></a>
### Bindings

`bindings` is a block within the configuration that can be repeated to specify mount points in the container. Each `bindings` block supports the following:

* `container_path` - (Optional, string) The container path where the volume is mounted.
* `host_path` - (Optional, string) The host path of the volume.
* `rewritable` - (Optional, bool) Grant the volume writable permissions.
* `volumes_from` - (Optional, string) The resource URI of the service to mount volumes from.

<a id="env"></a>
### Env

`env` is a block within the configuration that can be repeated to specify environment variables to be injected into each service container. Each `env` block supports the following:

* `key` - (Required, string) Name of the environment variable.
* `value` - (Required, string) Value of the environment variable.

<a id="links"></a>
### Links

`links` is a block within the configuration that can be repeated to specify other containers to link this service to. Each `links` block supports the following:

* `to` - (Required, string) Name of the service to link to.
* `name` - (Optional, string) Override the default name with this value.

<a id="ports"></a>
### Ports

`ports` is a block within the configuration that can be repeated to specify the port mappings of the container. Each `ports` block supports the following:

* `internal` - (Required, int) Port within the container.
* `external` - (Required, int) Port exposed out of the container.
* `protocol` - (Optional, string) Protocol that can be used over this port, defaults to TCP.

## Attributes Reference

The following attributes are exported:

* `id` - The uuid of the service
* `uri` - The DockerCloud URI of the service
* `public_dns` - The assigned public DNS name of the service
