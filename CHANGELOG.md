## 0.12.1 (Unreleased)

BUG FIXES:

core: Always try to select a workspace after initialization [GH-21234]

## 0.12.0 (May 22, 2019)

---

This is the aggregated summary of changes compared to v0.11.14. If you'd like to see the incremental changelog through each of the v0.12.0 prereleases, please refer to [the v0.12.0-rc1 changelog](https://github.com/hashicorp/terraform/blob/v0.12.0-rc1/CHANGELOG.md).

---

The focus of v0.12.0 was on improvements to the Terraform language made in response to all of the feedback and experience gathered on prior versions. We hope that these language improvements will help to make configurations for more complex situations more readable, and improve the usability of re-usable modules.

However, an overhaul of this kind inevitably means that 100% compatibility is not possible. The updated language is designed to be broadly compatible with the 0.11 language as documented, but some of the improvements required a slightly stricter parser and language model in order to resolve ambiguity or to give better feedback in error messages.

If you are upgrading to v0.12.0, we strongly recommend reading [the upgrade guide](https://www.terraform.io/upgrade-guides/0-12.html) to learn the recommended upgrade process, which includes a tool to automatically upgrade many improved language constructs and to indicate situations where human intuition is required to complete the upgrade.

### Incompatibilities and Notes

* As noted above, the language overhaul means that several aspects of the language are now parsed or evaluated more strictly than before, so configurations that employ workarounds for prior version limitations or that followed conventions other than what was shown in documentation may require some updates. For more information, please refer to [the upgrade guide](https://www.terraform.io/upgrade-guides/0-12.html).

* In order to give better feedback about mistakes, Terraform now validates that all variable names set via `-var` and `-var-file` options correspond to declared variables, generating errors or warnings if not. In situations where automation is providing a fixed set of variables to all configurations (whether they are using them or not), use [`TF_VAR_` environment variables](https://www.terraform.io/docs/commands/environment-variables.html#tf_var_name) instead, which are ignored if they do not correspond to a declared variable.

* The wire protocol for provider and provisioner plugins has changed, so plugins built against prior versions of Terraform are not compatible with Terraform v0.12. The most commonly-downloaded providers already had v0.12-compatible releases at the time of v0.12.0 release, but some other providers (particularly those distributed independently of the `terraform init` installation mechanism) will need to make new releases before they can be used with Terraform v0.12 or later.

* The index API for automatic provider installation in `terraform init` is now provided by the Terraform Registry at `registry.terraform.io`, rather than the indexes directly on `releases.hashicorp.com`. The "releases" server is still currently the distribution source for the release archives themselves at the time of writing, but that may change over time.

* The serialization formats for persisted state snapshots and saved plans have changed. Third-party tools that parse these artifacts will need to be updated to support these new serialization formats.

  For most use-cases, we recommend instead using [`terraform show -json`](https://www.terraform.io/docs/commands/show.html#json-output) to read the content of state or plan, in a form that is less likely to see significant breaking changes in future releases.

* [`terraform validate`](https://www.terraform.io/docs/commands/validate.html) now has a slightly smaller scope than before, focusing only on configuration syntax and type/value checking. This makes it safe to run in unattended scenarios, such as on save in a text editor.

### New Features

The full set of language improvements is too large to list them all out exhaustively, so the list below covers some highlights:

* **First-class expressions:** Prior to v0.12, expressions could be used only via string interpolation, like `"${var.foo}"`. Expressions are now fully integrated into the language, allowing them to be used directly as argument values, like `ami = var.ami`.

* **`for` expressions:** This new expression construct allows the construction of a list or map by transforming and filtering elements from another list or map. For more information, refer to [the _`for` expressions_ documentation](https://www.terraform.io/docs/configuration/expressions.html#for-expressions).

* **Dynamic configuration blocks:** For nested configuration blocks accepted as part of a resource configuration, it is now possible to dynamically generate zero or more blocks corresponding to items in a list or map using the special new `dynamic` block construct. This is the official replacement for the common (but buggy) unofficial workaround of treating a block type name as if it were an attribute expecting a list of maps value, which worked sometimes before as a result of some unintended coincidences in the implementation.

* **Generalised "splat" operator:** The `aws_instance.foo.*.id` syntax was previously a special case only for resources with `count` set. It is now an operator within the expression language that can be applied to any list value. There is also an optional new splat variant that allows both index and attribute access operations on each item in the list. For more information, refer to [the _Splat Expressions_ documentation](https://www.terraform.io/docs/configuration/expressions.html#splat-expressions).

* **Nullable argument values:** It is now possible to use a conditional expression like `var.foo != "" ? var.foo : null` to conditionally leave an argument value unset, whereas before Terraform required the configuration author to provide a specific default value in this case. Assigning `null` to an argument is equivalent to omitting that argument entirely.

* **Rich types in module inputs variables and output values:** Terraform v0.7 added support for returning flat lists and maps of strings, but this is now generalized to allow returning arbitrary nested data structures with mixed types. Module authors can specify an expected [type constraint](https://www.terraform.io/docs/configuration/types.html) for each input variable to allow early type checking of arguments.

* **Resource and module object values:** An entire resource or module can now be treated as an object value within expressions, including passing them through input variables and output values to other modules, using an attribute-less reference syntax, like `aws_instance.foo`.

* **Extended template syntax:** The simple interpolation syntax from prior versions is extended to become a simple template language, with support for conditional interpolations and repeated interpolations through iteration. For more information, see [the _String Templates_ documentation](https://www.terraform.io/configuration/expressions.html#string-templates).

* **`jsondecode` and `csvdecode` interpolation functions:** Due to the richer type system in the new configuration language implementation, we can now offer functions for decoding serialization formats. [`jsondecode`](https://www.terraform.io/docs/configuration/functions/jsondecode.html) is the opposite of [`jsonencode`](https://www.terraform.io/docs/configuration/functions/jsonencode.html), while [`csvdecode`](https://www.terraform.io/docs/configuration/functions/csvdecode.html) provides a way to load in lists of maps from a compact tabular representation.

* **Revamped error messages:** Error messages relating to configuration now always include information about where in the configuration the problem was found, along with other contextual information. We have also revisited many of the most common error messages to reword them for clarity, consistency, and actionability.

* **Structual plan output:** When Terraform renders the set of changes it plans to make, it will now use formatting designed to be similar to the input configuration language, including nested rendering of individual changes within multi-line strings, JSON strings, and nested collections.

### Other Improvements

* `terraform validate` now accepts an argument `-json` which produces machine-readable output. Please refer to the documentation for this command for details on the format and some caveats that consumers must consider when using this interface. ([#17539](https://github.com/hashicorp/terraform/issues/17539))

* The JSON-based variant of the Terraform language now has a more tightly-specified and reliable mapping to the native syntax variant. In prior versions, certain Terraform configuration features did not function as expected or were not usable via the JSON-based forms. For more information, see [the _JSON Configuration Syntax_ documentation](https://www.terraform.io/docs/configuration/syntax-json.html).

* The new built-in function [`templatefile`](https://www.terraform.io/docs/configuration/functions/templatefile.html) allows rendering a template from a file directly in the language, without installing the separate Template provider and using the `template_file` data source.

* The new built-in function [`formatdate`](https://www.terraform.io/docs/configuration/functions/formatdate.html), which is a specialized string formatting function for creating machine-oriented timestamp strings in various formats.

* The new built-in functions [`reverse`](https://www.terraform.io/docs/configuration/functions/reverse.html), which reverses the order of items in a list, and [`strrev`](https://www.terraform.io/docs/configuration/functions/strrev.html), which reverses the order of Unicode characters in a string.

* A new `pg` state storage backend allows storing state in a PostgreSQL database.

* The `azurerm` state storage backend supports new authentication mechanisms, custom resource manager endpoints, and HTTP proxies.

* The `s3` state storage backend now supports `credential_source` in AWS configuration files, support for the new AWS regions `eu-north-1` and `ap-east-1`, and several other improvements previously made in the `aws` provider.

* The `swift` state storage backend now supports locking and workspaces.

### Bug Fixes

Quite a few bugs were fixed indirectly as a result of improvements to the underlying language engine, so a fully-comprehensive list of fixed bugs is not possible, but some of the more commonly-encountered bugs that are fixed in this release include:

* config: The conditional operator `... ? ... : ...` now works with result values of any type and only returns evaluation errors for the chosen result expression, as those familiar with this operator in other languages might expect.

* config: Accept and ignore UTF-8 byte-order mark for configuration files ([#19715](https://github.com/hashicorp/terraform/issues/19715))

* config: When using a splat expression like `aws_instance.foo.*.id`, the addition of a new instance to the set (whose `id` is therefore not known until after apply) will no longer cause all of the other ids in the resulting list to appear unknown.

* config: The `jsonencode` function now preserves the types of values passed to it, even inside nested structures, whereas before it had a tendency to convert primitive-typed values to string representations.

* config: The `format` and `formatlist` functions now attempt automatic type conversions when the given values do not match the "verbs" in the format string, rather than producing a result with error placeholders in it.

* config: Assigning a list containing one or more unknown values to an argument expecting a list no longer produces the incorrect error message "should be a list", because Terraform is now able to track the individual elements as being unknown rather than the list as a whole, and to track the type of each unknown value. (This also avoids any need to place seemingly-redundant list brackets around values that are already lists, which would now be interpreted as a list of lists.)

* cli: When `create_before_destroy` is enabled for a resource, replacement actions are reflected correctly in rendered plans as `+/-` rather than `-/+`, and described as such in the UI messages.

* core: Various root causes of the "diffs didn't match during apply" class of error are now checked at their source, allowing Terraform to either avoid the problem occurring altogether (ideally) or to provide a more actionable error message to help with reporting, finding, and fixing the bug.

---

For information on v0.11 and prior releases, please see [the v0.11 branch changelog](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md).
