---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_resource_quota"
sidebar_current: "docs-kubernetes-resource-resource-quota"
description: |-
  Creates a Kubernetes resource quota.
---

# kubernetes\_resource\_quota

When several users or teams share a cluster with a fixed number of nodes, there is a concern that one team could use more than its fair share of resources.

Resource quotas are a tool for administrators to address this concern.

Read more in the [documentation](https://github.com/kubernetes/kubernetes/blob/v1.0.3/docs/admin/resource-quota.md).

## Example Usage

```
resource "kubernetes_resource_quota" "default" {
    name = "myns"
    labels {
        name = "development"
    }
    spec = <<SPEC
{
  "hard": {
    "memory": "1Gi",
    "cpu": "20",
    "pods": "10",
    "services": "5",
    "replicationcontrollers":"20",
    "resourcequotas":"1"
  }
}
SPEC
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource_quota.
* `labels` - (Optional) A list of labels attached to the resource_quota.
* `spec` - (Required) The specification of the volume. Only `spec` section of
    the YAML or JSON structure is required (i.e. `hard` will be on the root level).
