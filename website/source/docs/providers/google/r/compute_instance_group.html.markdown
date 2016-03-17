---
layout: "google"
page_title: "Google: google_compute_instance_group"
sidebar_current: "docs-google-compute-instance-group"
description: |-
  Manages an Instance Group within GCE.
---

# google\_compute\_instance\_group

The Google Compute Engine Instance Group API creates and manages pools
of homogeneous Compute Engine virtual machine instances from a common instance
template.  For more information, see [the official documentation](https://cloud.google.com/compute/docs/instance-groups/unmanaged-groups)
and [API](https://cloud.google.com/compute/docs/reference/latest/instanceGroups)

## Example Usage

Empty instance group
```
resource "google_compute_instance_group" "foobar" {
	name = "terraform-test"
	description = "Terraform test instance group"
	zone = "us-central1-a"
}
```

With instances and named ports
```
resource "google_compute_instance_group" "foobar" {
	name = "terraform-test"
	description = "Terraform test instance group"
	instances = [ 
	              "${google_compute_instance.test.self_link}", 
	              "${google_compute_instance.test2.self_link}"
	            ]
	named_port {
		name = "http"
		port = "8080"
	}
	named_port {
		name = "https"
		port = "8443"
	}
	zone = "us-central1-a"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the instance group. Must be 1-63
characters long and comply with [RFC1035](https://www.ietf.org/rfc/rfc1035.txt).
Supported characters include lowercase letters, numbers, and hyphens.

* `description` - (Optional) An optional textual description of the instance
  group.

* `instances` - (Optional) List of instances in the group. They should be given as
  self_link URLs. When adding instances they must all be in the same network and 
  zone as the instance group.

* `named_port` - (Optional) Named ports are key:value pairs that represent a 
  service name and the port number that the service runs on. The key:value pairs 
  are simple metadata that the Load Balancing service can use. This can specified 
  multiple times

* `zone` - (Required) The zone that this instance group should be created in.

The `named_port` block supports:

* `name` - The name which the port will be mapped to.

* `port` - The port number to map the name to.

## Attributes Reference

The following attributes are exported:

* `network` - The network the instance group is in.

* `size` - The number of instances in the group.

* `self_link` - The URL of the created resource.
