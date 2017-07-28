---
layout: "guides"
page_title: "Managing Kubernetes with Terraform - Guides"
sidebar_current: "guides-managing-kubernetes-with-terraform"
description: |-
  This guide focuses on scheduling Kubernetes resources like Pods,
  Replication Controllers, Services etc. on top of a properly configured
  and running Kubernetes cluster.
---

# Managing Kubernetes with Terraform

## Kubernetes

[Kubernetes](https://kubernetes.io/) (K8S) is an open-source workload scheduler
with focus on containerized applications.

There are at least 2 steps involved in scheduling your first container
on a K8s cluster. You need the K8S cluster with all its components
running _somewhere_ and then schedule the K8S resources, like Pods,
Replication Controllers, Services etc.

This guide focuses mainly on the latter part and expects you to have
a properly configured & running Kubernetes cluster.

The guide also expects you to run the cluster on a cloud provider
where K8S can automatically provision a load balancer.

## Why Terraform?

While you could use `kubectl` or similar CLI-based tools mapped to API calls
to manage all K8S resources described in YAML files,
orchestration with Terraform presents a few benefits.

 - [HCL](https://www.terraform.io/docs/configuration/syntax.html) - same
    language for both lower layer with underlying infrastructure (compute)
    and scheduling layer
 - drift detection - `terraform plan` will always present you the difference
    between reality at a given time and config you intend to apply.
 - full lifecycle management - Terraform doesn't just initially create resources,
    but offers a single command for creation, update, and deletion of tracked
    resources without needing to inspect the API to identify those resources.
 - synchronous feedback - While asynchronous behaviour is often useful,
    sometimes it's counter-productive as the job of identifying operation result
    (failures or details of created resource) is left to the user. e.g. you don't
    have IP/hostname of load balancer until it has finished provisioning,
    hence you can't create any DNS record pointing to it.
 - [graph of relationships](https://www.terraform.io/docs/internals/graph.html) -
    Terraform understands relationships between resources which may help
    in scheduling - e.g. if a Persistent Volume Claim claims space from
    a particular Persistent Volume Terraform won't even attempt to create
    the PVC if creation of the PV has failed.

## Provider Setup

The easiest way to configure the provider is by creating/generating a config
in a default location (`~/.kube/config`). That allows you to leave the
provider block completely empty.

```hcl
provider "kubernetes" {}
```

If you wish to configure the provider statically you can do so

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

After specifying the provider we may now run the following command
to pull down the latest version of the Kubernetes provider.

```
$ terraform init


Initializing provider plugins...
- Downloading plugin for provider "kubernetes"...

Terraform has been successfully initialized!

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
```

## Scheduling a Simple Application

The bread and butter of any K8S app is [a Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/#what-is-a-pod).
Pod consists of one or more containers which are scheduled
on cluster nodes based on CPU or memory being available.

Here we create a pod with a single container running the nginx web server,
exposing port 80 (HTTP) which can be then exposed
through the load balancer to the real user.

Unlike in this simple example you'd commonly run more than
a single instance of your application in production to reach
high availability and adding labels will allow K8S to find all
pods (instances) for the purpose of forwarding the traffic
to the exposed port.

```hcl
resource "kubernetes_pod" "nginx" {
  metadata {
    name = "nginx-example"
    labels {
      App = "nginx"
    }
  }

  spec {
    container {
      image = "nginx:1.7.8"
      name  = "example"

      port {
        container_port = 80
      }
    }
  }
}
```

The simplest way to expose your application to users is via [Service](https://kubernetes.io/docs/concepts/services-networking/service/).
Service is capable of provisioning a load-balancer in some cloud providers
and managing the relationship between pods and that load balancer
as new pods are launched and others die for any reason.

```hcl
resource "kubernetes_service" "nginx" {
  metadata {
    name = "nginx-example"
  }
  spec {
    selector {
      App = "${kubernetes_pod.nginx.metadata.0.labels.App}"
    }
    port {
      port = 80
      target_port = 80
    }

    type = "LoadBalancer"
  }
}
```

We may also add an output which will expose the IP address to the user

```hcl
output "lb_ip" {
  value = "${kubernetes_service.nginx.load_balancer_ingress.0.ip}"
}
```

Please note that this assumes a cloud provider provisioning IP-based
load balancer (like in Google Cloud Platform). If you run on a provider
with hostname-based load balancer (like in Amazon Web Services) you
should use the following snippet instead.

```hcl
output "lb_ip" {
  value = "${kubernetes_service.nginx.load_balancer_ingress.0.hostname}"
}
```

The plan will provide you an overview of planned changes, in this case
we should see 2 resources (Pod + Service) being added.
This commands gets more useful as your infrastructure grows and
becomes more complex with more components depending on each other
and it's especially helpful during updates.

```
$ terraform plan

Refreshing Terraform state in-memory prior to plan...
The refreshed state will be used to calculate this plan, but will not be
persisted to local or remote state storage.

The Terraform execution plan has been generated and is shown below.
Resources are shown in alphabetical order for quick scanning. Green resources
will be created (or destroyed and then created if an existing resource
exists), yellow resources are being changed in-place, and red resources
will be destroyed. Cyan entries are data sources to be read.

Note: You didn't specify an "-out" parameter to save this plan, so when
"apply" is called, Terraform can't guarantee this is what will execute.

  + kubernetes_pod.nginx
      metadata.#:                                  "1"
      metadata.0.generation:                       "<computed>"
      metadata.0.labels.%:                         "1"
      metadata.0.labels.App:                       "nginx"
      metadata.0.name:                             "nginx-example"
      metadata.0.namespace:                        "default"
      metadata.0.resource_version:                 "<computed>"
      metadata.0.self_link:                        "<computed>"
      metadata.0.uid:                              "<computed>"
      spec.#:                                      "1"
      spec.0.automount_service_account_token:      "<computed>"
      spec.0.container.#:                          "1"
      spec.0.container.0.image:                    "nginx:1.7.8"
      spec.0.container.0.image_pull_policy:        "<computed>"
      spec.0.container.0.name:                     "example"
      spec.0.container.0.port.#:                   "1"
      spec.0.container.0.port.0.container_port:    "80"
      spec.0.container.0.port.0.protocol:          "TCP"
      spec.0.container.0.resources.#:              "<computed>"
      spec.0.container.0.stdin:                    "false"
      spec.0.container.0.stdin_once:               "false"
      spec.0.container.0.termination_message_path: "/dev/termination-log"
      spec.0.container.0.tty:                      "false"
      spec.0.dns_policy:                           "ClusterFirst"
      spec.0.host_ipc:                             "false"
      spec.0.host_network:                         "false"
      spec.0.host_pid:                             "false"
      spec.0.hostname:                             "<computed>"
      spec.0.image_pull_secrets.#:                 "<computed>"
      spec.0.node_name:                            "<computed>"
      spec.0.restart_policy:                       "Always"
      spec.0.service_account_name:                 "<computed>"
      spec.0.termination_grace_period_seconds:     "30"

  + kubernetes_service.nginx
      load_balancer_ingress.#:     "<computed>"
      metadata.#:                  "1"
      metadata.0.generation:       "<computed>"
      metadata.0.name:             "nginx-example"
      metadata.0.namespace:        "default"
      metadata.0.resource_version: "<computed>"
      metadata.0.self_link:        "<computed>"
      metadata.0.uid:              "<computed>"
      spec.#:                      "1"
      spec.0.cluster_ip:           "<computed>"
      spec.0.port.#:               "1"
      spec.0.port.0.node_port:     "<computed>"
      spec.0.port.0.port:          "80"
      spec.0.port.0.protocol:      "TCP"
      spec.0.port.0.target_port:   "80"
      spec.0.selector.%:           "1"
      spec.0.selector.App:         "nginx"
      spec.0.session_affinity:     "None"
      spec.0.type:                 "LoadBalancer"


Plan: 2 to add, 0 to change, 0 to destroy.
```

As we're happy with the plan output we may carry on applying
proposed changes. `terraform apply` will take of all the hard work
which includes creating resources via API in the right order,
supplying any defaults as necessary and waiting for
resources to finish provisioning to the point when it can either
present useful attributes or a failure (with reason) to the user.

```
$ terraform apply

kubernetes_pod.nginx: Creating...
  metadata.#:                                  "" => "1"
  metadata.0.generation:                       "" => "<computed>"
  metadata.0.labels.%:                         "" => "1"
  metadata.0.labels.App:                       "" => "nginx"
  metadata.0.name:                             "" => "nginx-example"
  metadata.0.namespace:                        "" => "default"
  metadata.0.resource_version:                 "" => "<computed>"
  metadata.0.self_link:                        "" => "<computed>"
  metadata.0.uid:                              "" => "<computed>"
  spec.#:                                      "" => "1"
  spec.0.automount_service_account_token:      "" => "<computed>"
  spec.0.container.#:                          "" => "1"
  spec.0.container.0.image:                    "" => "nginx:1.7.8"
  spec.0.container.0.image_pull_policy:        "" => "<computed>"
  spec.0.container.0.name:                     "" => "example"
  spec.0.container.0.port.#:                   "" => "1"
  spec.0.container.0.port.0.container_port:    "" => "80"
  spec.0.container.0.port.0.protocol:          "" => "TCP"
  spec.0.container.0.resources.#:              "" => "<computed>"
  spec.0.container.0.stdin:                    "" => "false"
  spec.0.container.0.stdin_once:               "" => "false"
  spec.0.container.0.termination_message_path: "" => "/dev/termination-log"
  spec.0.container.0.tty:                      "" => "false"
  spec.0.dns_policy:                           "" => "ClusterFirst"
  spec.0.host_ipc:                             "" => "false"
  spec.0.host_network:                         "" => "false"
  spec.0.host_pid:                             "" => "false"
  spec.0.hostname:                             "" => "<computed>"
  spec.0.image_pull_secrets.#:                 "" => "<computed>"
  spec.0.node_name:                            "" => "<computed>"
  spec.0.restart_policy:                       "" => "Always"
  spec.0.service_account_name:                 "" => "<computed>"
  spec.0.termination_grace_period_seconds:     "" => "30"
kubernetes_pod.nginx: Creation complete (ID: default/nginx-example)
kubernetes_service.nginx: Creating...
  load_balancer_ingress.#:     "" => "<computed>"
  metadata.#:                  "" => "1"
  metadata.0.generation:       "" => "<computed>"
  metadata.0.name:             "" => "nginx-example"
  metadata.0.namespace:        "" => "default"
  metadata.0.resource_version: "" => "<computed>"
  metadata.0.self_link:        "" => "<computed>"
  metadata.0.uid:              "" => "<computed>"
  spec.#:                      "" => "1"
  spec.0.cluster_ip:           "" => "<computed>"
  spec.0.port.#:               "" => "1"
  spec.0.port.0.node_port:     "" => "<computed>"
  spec.0.port.0.port:          "" => "80"
  spec.0.port.0.protocol:      "" => "TCP"
  spec.0.port.0.target_port:   "" => "80"
  spec.0.selector.%:           "" => "1"
  spec.0.selector.App:         "" => "nginx"
  spec.0.session_affinity:     "" => "None"
  spec.0.type:                 "" => "LoadBalancer"
kubernetes_service.nginx: Still creating... (10s elapsed)
kubernetes_service.nginx: Still creating... (20s elapsed)
kubernetes_service.nginx: Still creating... (30s elapsed)
kubernetes_service.nginx: Still creating... (40s elapsed)
kubernetes_service.nginx: Creation complete (ID: default/nginx-example)

Apply complete! Resources: 2 added, 0 changed, 0 destroyed.

The state of your infrastructure has been saved to the path
below. This state is required to modify and destroy your
infrastructure, so keep it safe. To inspect the complete state
use the `terraform show` command.

State path:

Outputs:

lb_ip = 35.197.9.247
```

You may now enter that IP address to your favourite browser
and you should see the nginx welcome page.

The [Kubernetes UI](https://kubernetes.io/docs/tasks/access-application-cluster/web-ui-dashboard/)
provides another way to check both the pod and the service there
once they're scheduled.

## Reaching Scalability and Availability

The Replication Controller allows you to replicate pods. This is useful
for maintaining overall availability and scalability of your application
exposed to the user.

We can just replace our Pod with RC from the previous config
and keep the Service there.

```hcl
resource "kubernetes_replication_controller" "nginx" {
  metadata {
    name = "scalable-nginx-example"
    labels {
      App = "ScalableNginxExample"
    }
  }

  spec {
    replicas = 2
    selector {
      App = "ScalableNginxExample"
    }
    template {
      container {
        image = "nginx:1.7.8"
        name  = "example"

        port {
          container_port = 80
        }

        resources {
          limits {
            cpu    = "0.5"
            memory = "512Mi"
          }
          requests {
            cpu    = "250m"
            memory = "50Mi"
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "nginx" {
  metadata {
    name = "nginx-example"
  }
  spec {
    selector {
      App = "${kubernetes_replication_controller.nginx.metadata.0.labels.App}"
    }
    port {
      port = 80
      target_port = 80
    }

    type = "LoadBalancer"
  }
}
```

You may notice we also specified how much CPU and memory do we expect
single instance of that application to consume. This is incredibly
helpful for K8S as it helps avoiding under-provisioning or over-provisioning
that would result in either unused resources (costing money) or lack
of resources (causing the app to crash or slow down).

```
$ terraform plan

...

Plan: 2 to add, 0 to change, 0 to destroy.
```

```
$ terraform apply

kubernetes_replication_controller.nginx: Creating...
...
kubernetes_replication_controller.nginx: Creation complete (ID: default/scalable-nginx-example)
kubernetes_service.nginx: Creating...
...
kubernetes_service.nginx: Still creating... (10s elapsed)
kubernetes_service.nginx: Still creating... (20s elapsed)
kubernetes_service.nginx: Still creating... (30s elapsed)
kubernetes_service.nginx: Still creating... (40s elapsed)
kubernetes_service.nginx: Still creating... (50s elapsed)
kubernetes_service.nginx: Creation complete (ID: default/nginx-example)

Apply complete! Resources: 2 added, 0 changed, 0 destroyed.

The state of your infrastructure has been saved to the path
below. This state is required to modify and destroy your
infrastructure, so keep it safe. To inspect the complete state
use the `terraform show` command.

State path:

Outputs:

lb_ip = 35.197.9.247
```

Unlike in previous example, the IP address here will direct traffic
to one of the 2 pods scheduled in the cluster.

### Updating Configuration

As our application user-base grows we might need more instances to be scheduled.
The easiest way to achieve this is to increase `replicas` field in the config
accordingly.

```hcl
resource "kubernetes_replication_controller" "example" {
...

  spec {
    replicas = 5

...

}
```

You can verify before hitting the API that you're only changing what
you intended to change and that someone else didn't modify
the resource you created earlier.

```
$ terraform plan

Refreshing Terraform state in-memory prior to plan...
The refreshed state will be used to calculate this plan, but will not be
persisted to local or remote state storage.

kubernetes_replication_controller.nginx: Refreshing state... (ID: default/scalable-nginx-example)
kubernetes_service.nginx: Refreshing state... (ID: default/nginx-example)

The Terraform execution plan has been generated and is shown below.
Resources are shown in alphabetical order for quick scanning. Green resources
will be created (or destroyed and then created if an existing resource
exists), yellow resources are being changed in-place, and red resources
will be destroyed. Cyan entries are data sources to be read.

Note: You didn't specify an "-out" parameter to save this plan, so when
"apply" is called, Terraform can't guarantee this is what will execute.

  ~ kubernetes_replication_controller.nginx
      spec.0.replicas: "2" => "5"


Plan: 0 to add, 1 to change, 0 to destroy.
```

As we're happy with the proposed plan, we can just apply that change.

```
$ terraform apply
```

and 3 more replicas will be scheduled & attached to the load balancer.

## Bonus: Managing Quotas and Limits

As an operator managing cluster you're likely also responsible for
using the cluster responsibly and fairly within teams.

Resource Quotas and Limit Ranges both offer ways to put constraints
in place around CPU, memory, disk space and other resources that
will be consumed by cluster users.

Resource Quota can constrain the whole namespace

```hcl
resource "kubernetes_resource_quota" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    hard {
      pods = 10
    }
    scopes = ["BestEffort"]
  }
}
```

whereas Limit Range can impose limits on a specific resource
type (e.g. Pod or Persistent Volume Claim).

```hcl
resource "kubernetes_limit_range" "example" {
    metadata {
        name = "terraform-example"
    }
    spec {
        limit {
            type = "Pod"
            max {
                cpu = "200m"
                memory = "1024M"
            }
        }
        limit {
            type = "PersistentVolumeClaim"
            min {
                storage = "24M"
            }
        }
        limit {
            type = "Container"
            default {
                cpu = "50m"
                memory = "24M"
            }
        }
    }
}
```

```
$ terraform plan
```

```
$ terraform apply
```

## Conclusion

Terraform offers you an effective way to manage both compute for
your Kubernetes cluster and Kubernetes resources. Check out
https://www.terraform.io/docs/providers/kubernetes/index.html
for the extensive documentation of the Kubernetes provider.
