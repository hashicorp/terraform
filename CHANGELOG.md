## 0.13.0 (Unreleased)

BREAKING CHANGES:

* command/import: remove the deprecated `-provider` command line argument [GH-24090]
#22862 fixed a bug where the `import` command was not properly attaching the configured provider for a resource to be imported, making the `-provider` command line argument unnecessary. 
* config: Inside `provisioner` blocks that have `when = destroy` set, and inside any `connection` blocks that are used by such `provisioner` blocks, it is now an error to refer to any objects other than `self`, `count`, or `each` [GH-24083]
* config: The `merge` function now returns more precise type information, making it usable for values passed to `for_each` [GH-24032]


BUG FIXES: 
* cli: The `terraform plan` command (and the implied plan run by `terraform apply` with no arguments) will now print any warnings that were generated even if there are no changes to be made. [GH-24095]
* core: Instances are now destroyed only using their stored state, removing many cycle errors [GH-24083]

---
For information on prior major releases, see their changelogs:

* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
