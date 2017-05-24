---
layout: "rancher"
page_title: "Rancher: rancher_certificate"
sidebar_current: "docs-rancher-resource-certificate"
description: |-
  Provides a Rancher Certificate resource. This can be used to create certificates for rancher environments and retrieve their information.
---

# rancher\_certificate

Provides a Rancher Certificate resource. This can be used to create certificates for rancher environments and retrieve their information.

## Example Usage

```hcl
# Create a new Rancher Certificate
resource rancher_certificate "foo" {
  name           = "foo"
  description    = "my foo certificate"
  environment_id = "${rancher_environment.test.id}"
  cert = "${file("server.crt")}"
  key = "${file("server.key")}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the registry credential.
* `description` - (Optional) A registry credential description.
* `environment_id` - (Required) The ID of the environment to create the certificate for.
* `cert` - (Required) The certificate content.
* `cert_chain` - (Optional) The certificate chain.
* `key` - (Required) The certificate key.

## Attributes Reference

The following attributes are exported:

* `cn` - The certificate CN.
* `algorithm` - The certificate algorithm.
* `cert_fingerprint` - The certificate fingerprint.
* `expires_at` - The certificate expiration date.
* `issued_at` - The certificate creation date.
* `issuer` - The certificate issuer.
* `key_size` - The certificate key size.
* `serial_number` - The certificate serial number.
* `subject_alternative_names` - The list of certificate Subject Alternative Names.
* `version` - The certificate version.

## Import

Registry credentials can be imported using the Registry and credentials
IDs in the format `<environment_id>/<certificate_id>`

```
$ terraform import rancher_certificate.mycert 1sp31/1c605
```

If the credentials for the Rancher provider have access to the global API,
then `environment_id` can be omitted e.g.

```
$ terraform import rancher_certificate.mycert 1c605
```
