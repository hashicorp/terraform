---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_limit_range"
sidebar_current: "docs-kubernetes-resource-limit-range"
description: |-
  Creates a Kubernetes Limit Range.
---

# kubernetes\_limit\_range

By default, [pods](/docs/providers/kubernetes/r/pod.html) run with unbounded CPU and memory limits. This means that any pod in the system will be able to consume as much CPU and memory on the node that executes the pod. Limit Range allows you to define limits.

Read more about Limit Ranges [in docs](https://github.com/kubernetes/kubernetes/tree/master/docs/admin/limitrange#limit-range).

## Example Usage

```
resource "kubernetes_limit_range" "wp" {
    name = "mylimits"

    spec = <<SPEC
limits:
  - max:
      cpu: "2"
      memory: 1Gi
    min:
      cpu: 250m
      memory: 6Mi
    type: Pod
  - default:
      cpu: 250m
      memory: 100Mi
    max:
      cpu: "2"
      memory: 1Gi
    min:
      cpu: 250m
      memory: 6Mi
    type: Container
SPEC
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the limit range.
* `spec` - (Required) The specification of the limit range. Only `spec` section of
    the YAML or JSON structure is required (i.e. `limits` will be on the root level).
    See [documentation](http://kubernetes.io/v1.0/docs/user-guide/persistent-volumes/README.html) for details.
* `namespace` - (Optional) Namespace defines the space within which `name` must be unique.
    See [documentation](https://github.com/GoogleCloudPlatform/kubernetes/blob/v1/docs/design/namespaces.md)
    for details about other effects & features of namespaces.
* `labels` - (Optional) A list of labels attached to the limit range.

## Attributes Reference

The following attributes are exported:

* `id` - Unique ID of the limit range.

