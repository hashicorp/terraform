## 0.13.0 (Unreleased)

BREAKING CHANGES:

* command/import: remove the deprecated -provider command line argument [GH-24090]
#22862 fixed a bug where the `import` command was not properly attaching the configured provider for a resource to be imported, making the `-provider` command line argument unnecessary. 

---

For information on v0.12, please see [the v0.12 branch changelog](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md).

For information on v0.11 and prior releases, please see [the v0.11 branch changelog](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md).
