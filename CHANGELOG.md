## 0.2.1 (August 31, 2014)

IMPROVEMENTS:

  * core: Plugins are automatically discovered in the executable directory
      or pwd if named properly. [GH-190]
  * providers/mailgun: domain records are now saved to state

BUG FIXES:

  * core: Configuration parses when identifier and '=' have no space. [GH-243]
  * core: `depends_on` with `count` generates the proper graph. [GH-244]
  * core: Depending on a computed variable of a list type generates a
      plan without failure. i.e. `${type.name.foos.0.bar}` where `foos`
      is computed. [GH-247]
  * providers/aws: Route53 destroys in parallel work properly. [GH-183]

## 0.2.0 (August 28, 2014)

BACKWARDS INCOMPATIBILITIES:

  * We've replaced the configuration language in use from a C library to
    a pure-Go reimplementation. In the process, we removed some features
    of the language since it was too flexible:
    * Semicolons are no longer valid at the end of lines
    * Keys cannot be double-quoted strings: `"foo" = "bar"` is no longer
      valid.
    * JSON style maps `{ "foo": "bar" }` are no longer valid outside of JSON.
      Maps must be in the format of `{ foo = "bar" }` (like other objects
      in the config)
  * Heroku apps now require (will not validate without) `region` and
    `name` due to an upstream API change. [GH-239]

FEATURES:

  * **New Provider: `google`**: Manage Google Compute instances, disks,
      firewalls, and more.
  * **New Provider: `mailgun`**: Manage mailgun domains.
  * **New Function: `concat`**: Concatenate multiple strings together.
    Example: `concat(var.region, "-", var.channel)`.

IMPROVEMENTS:

  * core: "~/.terraformrc" (Unix) or "%APPDATA%/terraform.rc" (Windows)
    can be used to configure custom providers and provisioners. [GH-192]
  * providers/aws: EIPs now expose `allocation_id` and `public_ip`
      attributes.
  * providers/aws: Security group rules can be updated without a
      destroy/create.
  * providers/aws: You can enable and disable dns settings for VPCs. [GH-172]
  * providers/aws: Can specify a private IP address for `aws_instance` [GH-217]

BUG FIXES:

  * core: Variables are validated to not contain interpolations. [GH-180]
  * core: Key files for provisioning can now contain `~` and will be expanded
      to the user's home directory. [GH-179]
  * core: The `file()` function can load files in sub-directories. [GH-213]
  * core: Fix issue where some JSON structures didn't map properly into
     Terraform structures. [GH-177]
  * core: Resources with only `file()` calls will interpolate. [GH-159]
  * core: Variables work in block names. [GH-234]
  * core: Plugins are searched for in the same directory as the executable
      before the PATH. [GH-157]
  * command/apply: "tfvars" file no longer interferes with plan apply. [GH-153]
  * providers/aws: Fix issues around failing to read EIPs. [GH-122]
  * providers/aws: Autoscaling groups now register and export load
    balancers. [GH-207]
  * providers/aws: Ingress results are treated as a set, so order doesn't
      matter anymore. [GH-87]
  * providers/aws: Instance security groups treated as a set [GH-194]
  * providers/aws: Retry Route53 requests if operation failed because another
      operation is in progress [GH-183]
  * providers/aws: Route53 records with multiple record values work. [GH-221]
  * providers/aws: Changing AMI doesn't result in errors anymore. [GH-196]
  * providers/heroku: If you delete the `config_vars` block, config vars
      are properly nuked.
  * providers/heroku: Domains and drains are deleted before the app.
  * providers/heroku: Moved from the client library bgentry/heroku-go to
      cyberdelia/heroku-go [GH-239].
  * providers/heroku: Plans without a specific plan name for
      heroku\_addon work. [GH-198]

PLUGIN CHANGES:

  * **New Package:** `helper/schema`. This introduces a high-level framework
    for easily writing new providers and resources. The Heroku provider has
    been converted to this as an example.

## 0.1.1 (August 5, 2014)

FEATURES:

  * providers/heroku: Now supports creating Heroku Drains [GH-97]

IMPROVEMENTS:

  * providers/aws: Launch configurations accept user data [GH-94]
  * providers/aws: Regions are now validated [GH-96]
  * providers/aws: ELB now supports health check configurations [GH-109]

BUG FIXES:

  * core: Default variable file "terraform.tfvars" is auto-loaded. [GH-59]
  * core: Multi-variables (`foo.*.bar`) work even when `count = 1`. [GH-115]
  * core: `file()` function can have string literal arg [GH-145]
  * providers/cloudflare: Include the proper bins so the cloudflare
      provider is compiled
  * providers/aws: Engine version for RDS now properly set [GH-118]
  * providers/aws: Security groups now depend on each other and
  * providers/aws: DB instances now wait for destroys, have proper
      dependencies and allow passing skip_final_snapshot
  * providers/aws: Add associate_public_ip_address as an attribute on
      the aws_instance resource [GH-85]
  * providers/aws: Fix cidr blocks being updated [GH-65, GH-85]
  * providers/aws: Description is now required for security groups
  * providers/digitalocean: Private IP addresses are now a separate
      attribute
  * provisioner/all: If an SSH key is given with a password, a better
      error message is shown. [GH-73]

## 0.1.0 (July 28, 2014)

  * Initial release

