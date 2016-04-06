---
layout: "consul"
page_title: "Consul: consul_key_prefix"
sidebar_current: "docs-consul-resource-key-prefix"
description: |-
  Allows Terraform to manage a namespace of Consul keys that share a
  common name prefix.
---

# consul\_key\_prefix

Allows Terraform to manage a "namespace" of Consul keys that share a common
name prefix.

Like `consul_keys`, this resource can write values into the Consul key/value
store, but *unlike* `consul_keys` this resource can detect and remove extra
keys that have been added some other way, thus ensuring that rogue data
added outside of Terraform will be removed on the next run.

This resource is thus useful in the case where Terraform is exclusively
managing a set of related keys.

To avoid accidentally clobbering matching data that existed in Consul before
a `consul_key_prefix` resource was created, creation of a key prefix instance
will fail if any matching keys are already present in the key/value store.
If any conflicting data is present, you must first delete it manually.

~> **Warning** After this resource is instantiated, Terraform takes control
over *all* keys with the given path prefix, and will remove any matching keys
that are not present in the configuration. It will also delete *all* keys under
the given prefix when a `consul_key_prefix` resource is destroyed, even if
those keys were created outside of Terraform.

## Example Usage

```
resource "consul_key_prefix" "myapp_config" {
    datacenter = "nyc1"
    token = "abcd"

    # Prefix to add to prepend to all of the subkey names below.
    path_prefix = "myapp/config/"

    subkeys = {
        "elb_cname" = "${aws_elb.app.dns_name}"
        "s3_bucket_name" = "${aws_s3_bucket.app.bucket}"
        "database/hostname" = "${aws_db_instance.app.address}"
        "database/port" = "${aws_db_instance.app.port}"
        "database/username" = "${aws_db_instance.app.username}"
        "database/password" = "${aws_db_instance.app.password}"
        "database/name" = "${aws_db_instance.app.name}"
    }
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) The datacenter to use. This overrides the
  datacenter in the provider setup and the agent's default datacenter.

* `token` - (Optional) The ACL token to use. This overrides the
  token that the agent provides by default.

* `path_prefix` - (Required) Specifies the common prefix shared by all keys
  that will be managed by this resource instance. In most cases this will
  end with a slash, to manage a "folder" of keys.

* `subkeys` - (Required) A mapping from subkey name (which will be appended
  to the give `path_prefix`) to the value that should be stored at that key.
  Use slashes as shown in the above example to create "sub-folders" under
  the given path prefix.

## Attributes Reference

The following attributes are exported:

* `datacenter` - The datacenter the keys are being read/written to.
