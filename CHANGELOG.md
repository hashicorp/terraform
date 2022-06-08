## 1.3.0 (Unreleased)

UPGRADE NOTES:

* Module variable type constraints now support an `optional()` modifier for object attribute types. Optional attributes may be omitted from the variable value, and will be replaced by a default value (or `null` if no default is specified). For example:

    ```terraform
    variable "with_optional_attribute" {
      type = object({
        a = string                # a required attribute
        b = optional(string)      # an optional attribute
        c = optional(number, 127) # an optional attribute with default value
      })
    }
    ```

   Assigning `{ a = "foo" }` to this variable will result in the value `{ a = "foo", b = null, c = 127 }`.

    This functionality was introduced as an experiment in Terraform 0.14. This release removes the experimental `defaults` function. ([#31154](https://github.com/hashicorp/terraform/issues/31154))

BUG FIXES:

* Made `terraform output` CLI help documentation consistent with web-based documentation ([#29354](https://github.com/hashicorp/terraform/issues/29354))

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.2](https://github.com/hashicorp/terraform/blob/v1.2/CHANGELOG.md)
* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
