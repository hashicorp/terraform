---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_namespace"
sidebar_current: "docs-kubernetes-resource-namespace"
description: |-
  Kubernetes supports multiple virtual clusters backed by the same physical cluster. These virtual clusters are called namespaces.
---

# kubernetes_namespace

Kubernetes supports multiple virtual clusters backed by the same physical cluster. These virtual clusters are called namespaces.
Read more about namespaces at https://kubernetes.io/docs/user-guide/namespaces/

## Example Usage

```hcl
resource "kubernetes_namespace" "example" {
  metadata {
    annotations {
      name = "example-annotation"
    }

    labels {
      mylabel = "label-value"
    }

    name = "terraform-example-namespace"
  }
}

```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard namespace's [metadata](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata).

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the namespace that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more about [name idempotency](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency).
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) namespaces. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels
* `name` - (Optional) Name of the namespace, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this namespace that can be used by clients to determine when namespaces have changed. Read more about [concurrency control and consistency](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency).
* `self_link` - A URL representing this namespace.
* `uid` - The unique in time and space value for this namespace. More info: http://kubernetes.io/docs/user-guide/identifiers#uids

## Import

Namespaces can be imported using their name, e.g.

```
$ terraform import kubernetes_namespace.n terraform-example-namespace
```
