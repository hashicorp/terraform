---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_config_map"
sidebar_current: "docs-kubernetes-resource-config-map"
description: |-
  The resource provides mechanisms to inject containers with configuration data while keeping containers agnostic of Kubernetes.
---

# kubernetes_config_map

The resource provides mechanisms to inject containers with configuration data while keeping containers agnostic of Kubernetes.
Config Map can be used to store fine-grained information like individual properties or coarse-grained information like entire config files or JSON blobs.

## Example Usage

```hcl
resource "kubernetes_config_map" "example" {
  metadata {
    name = "my-config"
  }

  data {
    api_host = "myhost:443"
    db_host  = "dbhost:5432"
  }
}
```

## Argument Reference

The following arguments are supported:

* `data` - (Optional) A map of the configuration data.
* `metadata` - (Required) Standard config map's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the config map that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the config map. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels
* `name` - (Optional) Name of the config map, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names
* `namespace` - (Optional) Namespace defines the space within which name of the config map must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this config map that can be used by clients to determine when config map has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
* `self_link` - A URL representing this config map.
* `uid` - The unique in time and space value for this config map. More info: http://kubernetes.io/docs/user-guide/identifiers#uids

## Import

Config Map can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_config_map.example default/my-config
```
