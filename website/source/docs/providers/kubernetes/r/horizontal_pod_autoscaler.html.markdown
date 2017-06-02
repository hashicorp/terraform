---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_horizontal_pod_autoscaler"
sidebar_current: "docs-kubernetes-resource-horizontal-pod-autoscaler"
description: |-
  Horizontal Pod Autoscaler automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization.
---

# kubernetes_horizontal_pod_autoscaler

Horizontal Pod Autoscaler automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization.


## Example Usage

```hcl
resource "kubernetes_horizontal_pod_autoscaler" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    max_replicas = 10
    min_replicas = 8
    scale_target_ref {
      kind = "ReplicationController"
      name = "MyApp"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard horizontal pod autoscaler's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
* `spec` - (Required) Behaviour of the autoscaler. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the horizontal pod autoscaler that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the horizontal pod autoscaler. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels
* `name` - (Optional) Name of the horizontal pod autoscaler, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names
* `namespace` - (Optional) Namespace defines the space within which name of the horizontal pod autoscaler must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this horizontal pod autoscaler that can be used by clients to determine when horizontal pod autoscaler has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
* `self_link` - A URL representing this horizontal pod autoscaler.
* `uid` - The unique in time and space value for this horizontal pod autoscaler. More info: http://kubernetes.io/docs/user-guide/identifiers#uids

### `spec`

#### Arguments

* `max_replicas` - (Required) Upper limit for the number of pods that can be set by the autoscaler.
* `min_replicas` - (Optional) Lower limit for the number of pods that can be set by the autoscaler, defaults to `1`.
* `scale_target_ref` - (Required) Reference to scaled resource. e.g. Replication Controller
* `target_cpu_utilization_percentage` - (Optional) Target average CPU utilization (represented as a percentage of requested CPU) over all the pods. If not specified the default autoscaling policy will be used.

### `scale_target_ref`

#### Arguments

* `api_version` - (Optional) API version of the referent
* `kind` - (Required) Kind of the referent. e.g. `ReplicationController`. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds
* `name` - (Required) Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names

## Import

Horizontal Pod Autoscaler can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_horizontal_pod_autoscaler.example default/terraform-example
```
