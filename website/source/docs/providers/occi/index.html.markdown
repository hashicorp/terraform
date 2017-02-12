---
layout: "occi"
page_title: "Provider: OCCI"
sidebar_current: "docs-occi-index"
description: |-
  The OCCI (Open Cloud Computing Interface) provider is used to interact with the resources supported by OCCI. The provider needs rOCCI-cli to be functional.
---

# OCCI Provider

The [OCCI](http://occi-wg.org/) (Open Cloud Computing Interface) provider is used to interact with the resources supported by OCCI. The provider needs [rOCCI-cli](https://github.com/EGI-FCTF/rOCCI-cli) to be fully functional.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Create a virtual machine
resource "occi_virtual_machine" "vm" {
	...
}
```