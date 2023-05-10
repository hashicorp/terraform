## 1.5.0 (Unreleased)

NEW FEATURES:

* `check` blocks for validating infrastructure: Module and configuration authors can now write independent check blocks within their configuration to validate assertions about their infrastructure.

    The new independent `check` blocks must specify at least one `assert` block, but possibly many, each one with a `condition` expression and an `error_message` expression matching the existing [Custom Condition Checks](https://developer.hashicorp.com/terraform/language/v1.4.x/expressions/custom-conditions). 
    Additionally, check blocks can optionally load a scoped [data source](https://developer.hashicorp.com/terraform/language/v1.4.x/data-sources). Scoped data sources match the existing data sources with the exception that they can only be referenced from within their check block.

    Unlike the existing `precondition` and `postcondition` blocks, Terraform will not halt execution should the scoped data block fail or error or if any of the assertions fail. 
    This allows practitioners to continually validate the state of their infrastructure outside the usual lifecycle management cycle. 

* Adds a new `plantimestamp` function that returns the timestamp at plan time. This is similar to the `timestamp` function which returns the timestamp at apply time.
* Adds a new `strcontains` function that checks whether a given string contains a given substring. [GH-33069]


UPGRADE NOTES:

* This is the last version of Terraform for which macOS 10.13 High Sierra or 10.14 Mojave are officially supported. Future Terraform versions may not function correctly on these older versions of macOS.
* This is the last version of Terraform for which Windows 7, 8, Server 2008, and Server 2012 are supported by Terraform's main implementation language, Go. We already ended explicit support for versions earlier than Windows 10 in Terraform v0.15.0, but future Terraform versions may malfunction in more significant ways on these older Windows versions.
* On Linux (and some other non-macOS Unix platforms we don't officially support), Terraform will now notice the `trust-ad` option in `/etc/resolv.conf` and, if set, will set the "authentic data" option in outgoing DNS requests in order to better match the behavior of the GNU libc resolver.

    Terraform does not pay any attention to the corresponding option in responses, but some DNSSEC-aware recursive resolvers return different responses when the request option isn't set. This should therefore avoid some potential situations where a DNS request from Terraform might get a different response than a similar request from other software on your system.

ENHANCEMENTS:

* Terraform CLI's local operations mode will now attempt to persist state snapshots to the state storage backend periodically during the apply step, thereby reducing the window for lost data if the Terraform process is aborted unexpectedly. [GH-32680]
* If Terraform CLI receives SIGINT (or its equivalent on non-Unix platforms) during the apply step then it will immediately try to persist the latest state snapshot to the state storage backend, with the assumption that a graceful shutdown request often typically followed by a hard abort some time later if the graceful shutdown doesn't complete fast enough. [GH-32680]
* `pg` backend: Now supports the `PG_CONN_STR`, `PG_SCHEMA_NAME`, `PG_SKIP_SCHEMA_CREATION`, `PG_SKIP_TABLE_CREATION` and `PG_SKIP_INDEX_CREATION` environment variables. [GH-33045]

BUG FIXES:

* `terraform init`: Fixed crash with invalid blank module name. [GH-32781]
* `moved` blocks: Fixed a typo in the error message that Terraform raises when you use `-target` to exclude an object that has been moved. [GH-33149]

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
