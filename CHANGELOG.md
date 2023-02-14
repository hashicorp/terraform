## 1.5.0 (Unreleased)

ENHANCEMENTS:
* Terraform CLI's local operations mode will now attempt to persist state snapshots to the state storage backend periodically during the apply step, thereby reducing the window for lost data if the Terraform process is aborted unexpectedly. [GH-32680]
* If Terraform CLI recieves SIGINT (or its equivalent on non-Unix platforms) during the apply step then it will immediately try to persist the latest state snapshot to the state storage backend, with the assumption that a graceful shutdown request often typically followed by a hard abort some time later if the graceful shutdown doesn't complete fast enough. [GH-32680]

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.4](https://github.com/hashicorp/terraform/blob/v1.4/CHANGELOG.md)
* [v1.3](https://github.com/hashicorp/terraform/blob/v1.3/CHANGELOG.md)
* [v1.2](https://github.com/hashicorp/terraform/blob/v1.2/CHANGELOG.md)
* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
