---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_namespace"
sidebar_current: "docs-kubernetes-resource-namespace"
description: |-
  Creates a Kubernetes namespace.
---

# kubernetes\_namespace

Namespace provides a scope for names. Names of resources need to be unique within a namespace, but not across namespaces.

Namespace is a way to divide cluster resources between multiple uses (via [resource quota](/docs/providers/kubernetes/r/resource_quota.html)).

## Example Usage

```
resource "kubernetes_namespace" "default" {
    name = "myns"
    labels {
        name = "development"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the namespace.
* `labels` - (Optional) A list of labels attached to the namespace.

## Attributes Reference

The following attributes are exported:

* `id` - Unique ID of the namespace.
