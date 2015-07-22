---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service"
sidebar_current: "docs-kubernetes-resource-service"
description: |-
  Creates a Kubernetes Service.
---

# kubernetes\_service

A Kubernetes Service is an abstraction which defines a logical set of Pods
and a policy by which to access them - sometimes called a micro-service.

See [documentation](http://kubernetes.io/v1.0/docs/user-guide/services.html#overview) for details.


## Example Usage

```
resource "kubernetes_service" "wp" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
ports:
- port: 8000
  targetPort: 80
  protocol: TCP
selector:
  app: nginx
SPEC
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the service.
* `spec` - (Required) The specification of the service. Only `spec` section of
    the YAML or JSON structure is required (i.e. `ports` & `selector` will be on the root level).
    See [documentation](http://kubernetes.io/v1.0/docs/user-guide/walkthrough/k8s201.html#services) for details.
* `namespace` - (Optional) Namespace defines the space within which `name` must be unique.
    See [documentation](https://github.com/GoogleCloudPlatform/kubernetes/blob/v1/docs/design/namespaces.md)
    for details about other effects & features of namespaces.
* `labels` - (Optional) A list of labels attached to the service.

## Attributes Reference

The following attributes are exported:

* `id` - Unique ID of the service
