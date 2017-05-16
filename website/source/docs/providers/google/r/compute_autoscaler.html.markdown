---
layout: "google"
page_title: "Google: google_compute_autoscaler"
sidebar_current: "docs-google-compute-autoscaler"
description: |-
  Manages an Autoscaler within GCE.
---

# google\_compute\_autoscaler

A Compute Engine Autoscaler automatically adds or removes virtual machines from
a managed instance group based on increases or decreases in load. This allows
your applications to gracefully handle increases in traffic and reduces cost
when the need for resources is lower. You just define the autoscaling policy and
the autoscaler performs automatic scaling based on the measured load. For more
information, see [the official
documentation](https://cloud.google.com/compute/docs/autoscaler/) and
[API](https://cloud.google.com/compute/docs/autoscaler/v1beta2/autoscalers)


## Example Usage

```hcl
resource "google_compute_instance_template" "foobar" {
  name           = "foobar"
  machine_type   = "n1-standard-1"
  can_ip_forward = false

  tags = ["foo", "bar"]

  disk {
    source_image = "debian-cloud/debian-8"
  }

  network_interface {
    network = "default"
  }

  metadata {
    foo = "bar"
  }

  service_account {
    scopes = ["userinfo-email", "compute-ro", "storage-ro"]
  }
}

resource "google_compute_target_pool" "foobar" {
  name = "foobar"
}

resource "google_compute_instance_group_manager" "foobar" {
  name = "foobar"
  zone = "us-central1-f"

  instance_template  = "${google_compute_instance_template.foobar.self_link}"
  target_pools       = ["${google_compute_target_pool.foobar.self_link}"]
  base_instance_name = "foobar"
}

resource "google_compute_autoscaler" "foobar" {
  name   = "foobar"
  zone   = "us-central1-f"
  target = "${google_compute_instance_group_manager.foobar.self_link}"

  autoscaling_policy = {
    max_replicas    = 5
    min_replicas    = 1
    cooldown_period = 60

    cpu_utilization {
      target = 0.5
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the autoscaler.

* `target` - (Required) The full URL to the instance group manager whose size we
  control.

* `zone` - (Required) The zone of the target.

* `autoscaling_policy.` - (Required) The parameters of the autoscaling
  algorithm. Structure is documented below.

- - -

* `description` - (Optional) An optional textual description of the instance
    group manager.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

The `autoscaling_policy` block contains:

* `max_replicas` - (Required) The group will never be larger than this.

* `min_replicas` - (Required) The group will never be smaller than this.

* `cooldown_period` - (Optional) Period to wait between changes. This should be
  at least double the time your instances take to start up.

* `cpu_utilization` - (Optional) A policy that scales when the cluster's average
  CPU is above or below a given threshold. Structure is documented below.

* `metric` - (Optional) A policy that scales according to Google Cloud
  Monitoring metrics  Structure is documented below.

* `load_balancing_utilization` - (Optional) A policy that scales when the load
  reaches a proportion of a limit defined in the HTTP load balancer. Structure
is documented below.

The `cpu_utilization` block contains:

* `target` - The floating point threshold where CPU utilization should be. E.g.
  for 50% one would specify 0.5.

The `metric` block contains (more documentation
[here](https://cloud.google.com/monitoring/api/metrics)):

* `name` - The name of the Google Cloud Monitoring metric to follow, e.g.
  `compute.googleapis.com/instance/network/received_bytes_count`

* `type` - Either "cumulative", "delta", or "gauge".

* `target` - The desired metric value per instance. Must be a positive value.

The `load_balancing_utilization` block contains:

* `target` - The floating point threshold where load balancing utilization
  should be. E.g. if the load balancer's `maxRatePerInstance` is 10 requests
  per second (RPS) then setting this to 0.5 would cause the group to be scaled
  such that each instance receives 5 RPS.


## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URL of the created resource.
