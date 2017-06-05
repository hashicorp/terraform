---
layout: "kubernetes"
page_title: "Provider: Kubernetes"
sidebar_current: "docs-kubernetes-index"
description: |-
  The Kubernetes (K8s) provider is used to interact with the resources supported by Kubernetes. The provider needs to be configured with the proper credentials before it can be used.
---

# Kubernetes Provider

The Kubernetes (K8S) provider is used to interact with the resources supported by Kubernetes. The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

-> **Note:** The Kubernetes provider is new as of Terraform 0.9. It is ready to be used but many features are still being added. If there is a Kubernetes feature missing, please report it in the GitHub repo.

## Example Usage

```hcl
provider "kubernetes" {
  config_context_auth_info = "ops"
  config_context_cluster   = "mycluster"
}

resource "kubernetes_namespace" "example" {
  metadata {
    name = "my-first-namespace"
  }
}
```

## Kubernetes versions

Both backward and forward compatibility with Kubernetes API is mostly defined
by the [official K8S Go library](https://github.com/kubernetes/kubernetes) which we ship with Terraform.
Below are versions of the library bundled with given versions of Terraform.

* Terraform `<= 0.9.6` - Kubernetes `1.5.4`
* Terraform `0.9.7+` - Kubernetes `1.6.1`

## Authentication

There are generally two ways to configure the Kubernetes provider.

### File config

The provider always first tries to load **a config file** from a given
(or default) location. Depending on whether you have current context set
this _may_ require `config_context_auth_info` and/or `config_context_cluster`
and/or `config_context`.

#### Setting default config context

Here's an example for how to set default context and avoid all provider configuration:

```
kubectl config set-context default-system \
  --cluster=chosen-cluster \
  --user=chosen-user

kubectl config use-context default-system
```

Read [more about `kubectl` in the official docs](https://kubernetes.io/docs/user-guide/kubectl-overview/).

### Statically defined credentials

The other way is **statically** define all the credentials:

```hcl
provider "kubernetes" {
  host     = "https://104.196.242.174"
  username = "ClusterMaster"
  password = "MindTheGap"

  client_certificate     = "${file("~/.kube/client-cert.pem")}"
  client_key             = "${file("~/.kube/client-key.pem")}"
  cluster_ca_certificate = "${file("~/.kube/cluster-ca-cert.pem")}"
}
```

If you have **both** valid configuration in a config file and static configuration, the static one is used as override.
i.e. any static field will override its counterpart loaded from the config.

## Argument Reference

The following arguments are supported:

* `host` - (Optional) The hostname (in form of URI) of Kubernetes master. Can be sourced from `KUBE_HOST`. Defaults to `https://localhost`.
* `username` - (Optional) The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_USER`.
* `password` - (Optional) The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_PASSWORD`.
* `insecure`- (Optional) Whether server should be accessed without verifying the TLS certificate. Can be sourced from `KUBE_INSECURE`. Defaults to `false`.
* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication. Can be sourced from `KUBE_CLIENT_CERT_DATA`.
* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication. Can be sourced from `KUBE_CLIENT_KEY_DATA`.
* `cluster_ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication. Can be sourced from `KUBE_CLUSTER_CA_CERT_DATA`.
* `config_path` - (Optional) Path to the kube config file. Can be sourced from `KUBE_CONFIG` or `KUBECONFIG`. Defaults to `~/.kube/config`.
* `config_context` - (Optional) Context to choose from the config file. Can be sourced from `KUBE_CTX`.
* `config_context_auth_info` - (Optional) Authentication info context of the kube config (name of the kubeconfig user, `--user` flag in `kubectl`). Can be sourced from `KUBE_CTX_AUTH_INFO`.
* `config_context_cluster` - (Optional) Cluster context of the kube config (name of the kubeconfig cluster, `--cluster` flag in `kubectl`). Can be sourced from `KUBE_CTX_CLUSTER`.
