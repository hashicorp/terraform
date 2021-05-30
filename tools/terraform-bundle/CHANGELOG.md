## 0.14.0

BUG FIXES: 
* fix packaging for custom plugins ([#26394](https://github.com/hashicorp/terraform/pull/26394))

## 0.13.0 (August 10, 2020)

> This is a list of changes relative to terraform-bundle tagged v0.12.

Breaking Changes: 
* Terraform v0.13.0 has introduced a new hierarchical namespace for providers. Terraform v0.13 requires a new directory layout in order to discover locally-installed provider plugins, and terraform-bundle has been updated to match. Please see the [README](README.md) to learn more about the new directory layout.
