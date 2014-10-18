## 0.3.1 (unreleased)

BUG FIXES:

  * core: Remove panic case when applying with a plan that generates no
      new state. [GH-403]
  * core: Fix a hang that can occur with enough resources. [GH-410]
  * core: Config validation will not error if the field is being
      computed so the value is still unknown.
  * core: If a resource fails to create and has provisioners, it is
      marked as tainted. [GH-434]
  * core: Set types are validated to be sets. [GH-413]
  * core: Fix crash case when destroying with tainted resources. [GH-412]
  * core: Don't execute provisioners in some cases on destroy.
  * core: Inherited provider configurations will be properly interpolated. [GH-418]
  * providers/aws: Refresh of launch configs and autoscale groups load
      the correct data and don't incorrectly recreate themselves. [GH-425]
  * providers/aws: Fix case where ELB would incorrectly plan to modify
      listeners (with the same data) in some cases.
  * providers/aws: Retry destroying internet gateway for some amount of time
      if there is a dependency violation since it is probably just eventual
      consistency (public facing resources being destroyed). [GH-447]
  * providers/aws: Retry deleting security groups for some amount of time
      if there is a dependency violation since it is probably just eventual
      consistency. [GH-436]
  * providers/aws: Retry deleting subnet for some amount of time if there is a
      dependency violation since probably asynchronous destroy events take
      place still. [GH-449]
  * providers/aws: Drain autoscale groups before deleting. [GH-435]
  * providers/aws: Fix crash case if launch config is manually deleted. [GH-421]
  * providers/aws: Disassociate EIP before destroying.

## 0.3.0 (October 14, 2014)

FEATURES:

  * **Modules**: Configuration can now be modularized. Modules can live on
    GitHub, BitBucket, Git/Hg repos, HTTP URLs, and file paths. Terraform
    automatically downloads/updates modules for you on request.
  * **New Command: `init`**. This command initializes a Terraform configuration
    from an existing Terraform module (also new in 0.3).
  * **New Command: `destroy`**. This command destroys infrastructure
    created with `apply`.
  * Terraform will ask for user input to fill in required variables and
    provider configurations if they aren't set.
  * `terraform apply MODULE` can be used as a shorthand to quickly build
    infrastructure from a module.
  * The state file format is now JSON rather than binary. This allows for
    easier machine and human read/write. Old binary state files will be
    automatically upgraded.
  * You can now specify `create_before_destroy` as an option for replacement
    so that new resources are created before the old ones are destroyed.
  * The `count` metaparameter can now contain interpolations (such as
    variables).
  * The current index for a resource with a `count` set can be interpolated
    using `${count.index}`.
  * Various paths can be interpolated with the `path.X` variables. For example,
    the path to the current module can be interpolated using `${path.module}`.

IMPROVEMENTS:

  * config: Trailing commas are now allowed for the final elements of lists.
  * core: Plugins are loaded from `~/.terraform.d/plugins` (Unix) or
    `%USERDATA%/terraform.d/plugins` (Windows).
  * command/show: With no arguments, it will show the default state. [GH-349]
  * helper/schema: Can now have default values. [GH-245]
  * providers/aws: Tag support for most resources.
  * providers/aws: New resource `db_subnet_group`. [GH-295]
  * providers/aws: Add `map_public_ip_on_launch` for subnets. [GH-285]
  * providers/aws: Add `iam_instance_profile` for instances. [GH-319]
  * providers/aws: Add `internal` option for ELBs. [GH-303]
  * providers/aws: Add `ssl_certificate_id` for ELB listeners. [GH-350]
  * providers/aws: Add `self` option for security groups for ingress
      rules with self as source. [GH-303]
  * providers/aws: Add `iam_instance_profile` option to
      `aws_launch_configuration`. [GH-371]
  * providers/aws: Non-destructive update of `desired_capacity` for
      autoscale groups.
  * providers/aws: Add `main_route_table_id` attribute to VPCs. [GH-193]
  * providers/consul: Support tokens. [GH-396]
  * providers/google: Support `target_tags` for firewalls. [GH-324]
  * providers/google: `google_compute_instance` supports `can_ip_forward` [GH-375]
  * providers/google: `google_compute_disk` supports `type` to support disks
      such as SSDs. [GH-351]
  * provisioners/local-exec: Output from command is shown in CLI output. [GH-311]
  * provisioners/remote-exec: Output from command is shown in CLI output. [GH-311]

BUG FIXES:

  * core: Providers are validated even without a `provider` block. [GH-284]
  * core: In the case of error, walk all non-dependent trees.
  * core: Plugin loading from CWD works properly.
  * core: Fix many edge cases surrounding the `count` meta-parameter.
  * core: Strings in the configuration can escape double-quotes with the
      standard `\"` syntax.
  * core: Error parsing CLI config will show properly. [GH-288]
  * core: More than one Ctrl-C will exit immediately.
  * providers/aws: autoscaling_group can be launched into a vpc [GH-259]
  * providers/aws: not an error when RDS instance is deleted manually. [GH-307]
  * providers/aws: Retry deleting subnet for some time while AWS eventually
      destroys dependencies. [GH-357]
  * providers/aws: More robust destroy for route53 records. [GH-342]
  * providers/aws: ELB generates much more correct plans without extranneous
      data.
  * providers/aws: ELB works properly with dynamically changing
      count of instances.
  * providers/aws: Terraform can handle ELBs deleted manually. [GH-304]
  * providers/aws: Report errors properly if RDS fails to delete. [GH-310]
  * providers/aws: Wait for launch configuration to exist after creation
      (AWS eventual consistency) [GH-302]

## 0.2.2 (September 9, 2014)

IMPROVEMENTS:

  * providers/amazon: Add `ebs_optimized` flag. [GH-260]
  * providers/digitalocean: Handle 404 on delete
  * providers/digitalocean: Add `user_data` argument for creating droplets
  * providers/google: Disks can be marked `auto_delete`. [GH-254]

BUG FIXES:

  * core: Fix certain syntax of configuration that could cause hang. [GH-261]
  * core: `-no-color` flag properly disables color. [GH-250]
  * core: "~" is expanded in `-var-file` flags. [GH-273]
  * core: Errors with tfvars are shown in console. [GH-269]
  * core: Interpolation function calls with more than two args parse. [GH-282]
  * providers/aws: Refreshing EIP from pre-0.2 state file won't error. [GH-258]
  * providers/aws: Creating EIP without an instance/network won't fail.
  * providers/aws: Refreshing EIP manually deleted works.
  * providers/aws: Retry EIP delete to allow AWS eventual consistency to
      detect it isn't attached. [GH-276]
  * providers/digitalocean: Handle situations when resource was destroyed
      manually. [GH-279]
  * providers/digitalocean: Fix a couple scenarios where the diff was
      incorrect (and therefore the execution as well).
  * providers/google: Attaching a disk source (not an image) works
      properly. [GH-254]

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

