---
layout: "google"
page_title: "Google: google_compute_instance_group_manager"
sidebar_current: "docs-google-compute-instance-group-manager"
description: |-
  Manages an Instance Group within GCE.
---

# google\_compute\_instance\_group\_manager

The Google Compute Engine Instance Group Manager API creates and manages pools
of homogeneous Compute Engine virtual machine instances from a common instance
template.  For more information, see [the official documentation](https://cloud.google.com/compute/docs/instance-groups/manager
and [API](https://cloud.google.com/compute/docs/instance-groups/manager/v1beta2/instanceGroupManagers)

## Example Usage

```
resource "google_compute_instance_group_manager" "foobar" {
	description = "Terraform test instance group manager"
	name = "terraform-test"
	instance_template = "${google_compute_instance_template.foobar.self_link}"
	update_strategy= "NONE"
	target_pools = ["${google_compute_target_pool.foobar.self_link}"]
	base_instance_name = "foobar"
	zone = "us-central1-a"
	target_size = 2
}
```

## Argument Reference

The following arguments are supported:

* `base_instance_name` - (Required) The base instance name to use for
instances in this group. The value must be a valid [RFC1035](https://www.ietf.org/rfc/rfc1035.txt) name.
Supported characters are lowercase letters, numbers, and hyphens (-). Instances
are named by appending a hyphen and a random four-character string to the base
instance name.

* `description` - (Optional) An optional textual description of the instance
group manager.

* `instance_template` - (Required) The full URL to an instance template from
which all new instances will be created. 

* `update_strategy` - (Optional, Default `"RESTART"`) If the `instance_template` resource is
modified, a value of `"NONE"` will prevent any of the managed instances from
being restarted by Terraform. A value of `"RESTART"` will restart all of the 
instances at once. In the future, as the GCE API matures we will support
`"ROLLING_UPDATE"` as well.

* `name` - (Required) The name of the instance group manager. Must be 1-63
characters long and comply with [RFC1035](https://www.ietf.org/rfc/rfc1035.txt).
Supported characters include lowercase letters, numbers, and hyphens.

* `target_size` - (Optional) If not given at creation time, this defaults to 1.  Do not specify this
  if you are managing the group with an autoscaler, as this will cause fighting.

* `target_pools` - (Optional) The full URL of all target pools to which new
instances in the group are added. Updating the target pools attribute does not
affect existing instances.

* `zone` - (Required) The zone that instances in this group should be created in.

## Attributes Reference

The following attributes are exported:

* `instance_group` - The full URL of the instance group created by the manager.

* `self_link` - The URL of the created resource.
