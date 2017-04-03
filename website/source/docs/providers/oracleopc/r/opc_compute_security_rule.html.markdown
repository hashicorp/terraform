---
layout: "oracleopc"
page_title: "Oracle: opc_compute_security_rule"
sidebar_current: "docs-oracleopc-resource-security-rule"
description: |-
  Creates and manages a security rule in an OPC identity domain.
---

# opc\_compute\_ip\_reservation

The ``opc_compute_security_rule`` resource creates and manages a security rule in an OPC identity domain, which joins
together a source security list (or security IP list), a destination security list (or security IP list), and a security
application.

## Example Usage

```
resource "opc_compute_security_rule" "test_rule" {
	name = "test"
	source_list = "seclist:${opc_compute_security_list.sec-list1.name}"
	destination_list = "seciplist:${opc_compute_security_ip_list.sec-ip-list1.name}"
	action = "permit"
	application = "${opc_compute_security_application.spring-boot.name}"
	disabled = false
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique (within the identity domain) name of the security rule.

* `source_list` - (Required) The source security list (prefixed with `seclist:`), or security IP list (prefixed with
`seciplist:`).

 * `destination_list` - (Required) The destination security list (prefixed with `seclist:`), or security IP list (prefixed with
 `seciplist:`).

* `application` - (Required) The name of the application to which the rule applies.

* `action` - (Required) Whether to `permit`, `refuse` or `deny` packets to which this rule applies. This will ordinarily
be `permit`.

* `disabled` - (Required) Whether to disable this security rule. This is useful if you want to temporarily disable a rule
without removing it outright from your Terraform resource definition.
