## 0.13.0 (Unreleased)

BREAKING CHANGES:

* command/import: remove the deprecated -provider command line argument [GH-24090]
#22862 fixed a bug where the `import` command was not properly attaching the configured provider for a resource to be imported, making the `-provider` command line argument unnecessary. 

---
For information on prior major releases, see their changelogs:

* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
