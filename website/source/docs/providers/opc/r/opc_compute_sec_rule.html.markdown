---
layout: "opc"
page_title: "Oracle: opc_compute_sec_rule"
sidebar_current: "docs-opc-resource-sec-rule"
description: |-
  Creates and manages a sec rule in an OPC identity domain.
---

# opc\_compute\_sec\_rule

The ``opc_compute_sec_rule`` resource creates and manages a sec rule in an OPC identity domain, which joinstogether a source security list (or security IP list), a destination security list (or security IP list), and a security application.

## Example Usage

```hcl
resource "opc_compute_sec_rule" "test_rule" {
  name             = "test"
  source_list      = "seclist:${opc_compute_security_list.sec-list1.name}"
  destination_list = "seciplist:${opc_compute_security_ip_list.sec-ip-list1.name}"
  action           = "permit"
  application      = "${opc_compute_security_application.spring-boot.name}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique (within the identity domain) name of the security rule.

* `description` - (Optional) A description for this security rule.

* `source_list` - (Required) The source security list (prefixed with `seclist:`), or security IP list (prefixed with
`seciplist:`).

 * `destination_list` - (Required) The destination security list (prefixed with `seclist:`), or security IP list (prefixed with
 `seciplist:`).

* `application` - (Required) The name of the application to which the rule applies.

* `action` - (Required) Whether to `permit`, `refuse` or `deny` packets to which this rule applies. This will ordinarily
be `permit`.

* `disabled` - (Optional) Whether to disable this security rule. This is useful if you want to temporarily disable a rule
without removing it outright from your Terraform resource definition. Defaults to `false`.

In addition to the above, the following values are exported:

* `uri` - The Uniform Resource Identifier of the sec rule.

## Import

Sec Rule's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_sec_rule.rule1 example
```
