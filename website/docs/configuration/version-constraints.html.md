---
layout: "docs"
page_title: "Version Constraints - Configuration Language"
---


## from modules page

The `version` attribute value may either be a single explicit version or
a version constraint expression. Constraint expressions use the following
syntax to specify a _range_ of versions that are acceptable:

* `>= 1.2.0`: version 1.2.0 or newer
* `<= 1.2.0`: version 1.2.0 or older
* `~> 1.2.0`: any non-beta version `>= 1.2.0` and `< 1.3.0`, e.g. `1.2.X`
* `~> 1.2`: any non-beta version `>= 1.2.0` and `< 2.0.0`, e.g. `1.X.Y`
* `>= 1.0.0, <= 2.0.0`: any version between 1.0.0 and 2.0.0 inclusive

When depending on third-party modules, references to specific versions are
recommended since this ensures that updates only happen when convenient to you.

For modules maintained within your organization, a version range strategy
may be appropriate if a semantic versioning methodology is used consistently
or if there is a well-defined release process that avoids unwanted updates.


## from terraform core version

The value for `required_version` is a string containing a comma-separated
list of constraints. Each constraint is an operator followed by a version
number, such as `> 0.12.0`. The following constraint operators are allowed:

* `=` (or no operator): exact version equality

* `!=`: version not equal

* `>`, `>=`, `<`, `<=`: version comparison, where "greater than" is a larger
  version number

* `~>`: pessimistic constraint operator, constraining both the oldest and
  newest version allowed. For example, `~> 0.9` is equivalent to
  `>= 0.9, < 1.0`, and `~> 0.8.4`, is equivalent to `>= 0.8.4, < 0.9`

Re-usable modules should constrain only the minimum allowed version, such
as `>= 0.12.0`. This specifies the earliest version that the module is
compatible with while leaving the user of the module flexibility to upgrade
to newer versions of Terraform without altering the module.

## from required_providers

Version constraint strings within the `required_providers` block use the
same version constraint syntax as for
[the `required_version` argument](#specifying-a-required-terraform-version)
described above.

When a configuration contains multiple version constraints for a single
provider -- for example, if you're using multiple modules and each one has
its own constraint -- _all_ of the constraints must hold to select a single
provider version for the whole configuration.

Re-usable modules should constrain only the minimum allowed version, such
as `>= 1.0.0`. This specifies the earliest version that the module is
compatible with while leaving the user of the module flexibility to upgrade
to newer versions of the provider without altering the module.

Root modules should use a `~>` constraint to set both a lower and upper bound
on versions for each provider they depend on, as described in
[Provider Versions](providers.html#provider-versions).

