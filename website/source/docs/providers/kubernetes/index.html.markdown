---
layout: "kubernetes"
page_title: "Provider: Kubernetes"
sidebar_current: "docs-kubernetes-index"
description: |-
  The Kubernetes Cluster Provider
---

# Kubernetes Provider

The Kubernetes provider is used to interact with [Kubernetes](http://kubernetes.io/) cluster via given master endpoint.
The provider needs to be configured before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Kubernetes provider
provider "kubernetes" {
    endpoint = "https://192.168.1.1"
    username = "mr.yoda"
    password = "adoy.rm"
}

# Create a new pod
resource "kubernetes_pod" "default" {
    ...
}
```

## Configuration Reference

The following keys can be used to configure the provider.

* `endpoint` - (Required) The IP address or hostname of Kubernetes master

* `username` - (Required) The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint

* `password` - (Required) The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint

* `insecure` - (Optional) Whether the Kubernetes master should be accessed without verifying the TLS certificate

* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication

* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication

* `cluster_ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication
