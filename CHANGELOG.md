## 0.2.0 (unreleased)



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

