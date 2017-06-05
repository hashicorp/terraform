---
layout: "aws"
page_title: "AWS: aws_opsworks_chef_server"
sidebar_current: "docs-aws-resource-opsworks-chef-server"
description: |-
  Provides an OpsWorks for Chef Automate server resource.
---

# aws\_opsworks\_chef\_server

Provides an OpsWorks for Chef Automate server resource.

~> **Note:** This resource is only available in a limited selection of AWS
regions:

- `us-east-1`: US East (N. Virginia)
- `us-west-2`: US West (Oregon)
- `eu-west-1`: EU (Ireland)

## Example Usage

```hcl
resource "aws_opsworks_chef_server" "my-chef" {
  name = "my-chef"

  associate_public_ip_address = true
  backup_automatically        = true

  instance_type        = "t2.medium"
  instance_profile_arn = "${aws_iam_instance_profile.my-chef.arn}"
  service_role_arn     = "${aws_iam_role.my-chef-ServiceRole.arn}"
  subnet_ids           = ["${aws_subnet.my-chef.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional, Forces new resource) The name of the Chef Server.
   Conflicts with `name_prefix`.

* `name_prefix` - (Optional, Forces new resource) Creates a unique name
   beginning with the specified prefix.

* `instance_type` - (Required, Forces new resource) The EC2 instance type to
   use. Valid values are:

    - `t2.medium` (supports up to 50 nodes)
    - `m4.large` (supports up to 200 nodes)
    - `m4.2xlarge` (supports 200+ nodes)

* `security_group_ids` - (Optional, Forces new resource) A list of security
   group IDs to attach to the Amazon EC2 instance. If you add this parameter,
   the specified security groups must be within the VPC that is specified by
   `subnet_ids`. If you do not specify this parameter, AWS OpsWorks for Chef
   Automate creates one new security group that uses TCP ports 22 and 443, open
   to 0.0.0.0/0 (everyone).

   **Note:** The default security group opens the Chef server to the world on
   TCP port 443. If a `key_name` is present, AWS OpsWorks enables SSH access.
   SSH is also open to the world on TCP port 22. By default, the Chef Server
   is accessible from any IP address. We recommend that you update your
   security group rules to allow access from known IP addresses and address
   ranges only.

* `subnet_ids` - (Optional, Forces new resource) The IDs of subnets in which to
   launch the server EC2 instance.
    - **Amazon EC2-Classic customers**: This field is required. All servers
      must run within a VPC. The VPC must have "Auto Assign Public IP" enabled.
    - **EC2-VPC customers**: This field is optional. If you do not specify
      subnet IDs, your EC2 instances are created in a default subnet that is
      selected by Amazon EC2. If you specify subnet IDs, the VPC must have
      "Auto Assign Public IP" enabled.

* `associate_public_ip_address` - (Optional, Forces new resource) Associate a
   public ip address with an instance in a VPC.  Boolean value.

* `service_role_arn` - (Optional, Forces new resource) The service role that the
  AWS OpsWorks for Chef Automate service backend uses to work with your account.

* `instance_profile_arn` - (Optional, Forces new resource) The ARN of the
  instance profile that your Amazon EC2 instances use.

* `key_pair_name` - (Optional, Forces new resource) The Amazon EC2 key pair to
  set for the instance.

* `engine` - (Optional, Forces new resource) The configuration management engine
  to use. Defaults to "Chef".

* `engine_model` - (Optional, Forces new resource) The engine model, or option.
  Defaults to "Single".

* `engine_version` - (Optional, Forces new resource) The major release version
  of the engine that you want to use. Values depend on the engine that you
  choose. Defaults to "12".

* `chef_delivery_admin_password` - (Optional, Forces new resource) The password
  for the administrative user in the Chef Automate GUI. When no password is set,
  one is generated and returned in the response.

* `chef_pivotal_key` - (Optional, Forces new resource) An RSA private key that
  is not stored by AWS OpsWorks for Chef. This private key is required to access
  the Chef API. When no key is set, one is generated and returned in the
  response.

* `backup_automatically` - (Optional) Enable or disable scheduled backups. The
  default value is `true`.

* `backup_id` - (Optional, Forces new resource) If you specify this field, AWS
  OpsWorks for Chef Automate creates the server by using the backup with this
  ID.

* `backup_retention_count` - (Optional) The number of automated backups that
  you want to keep. Whenever a new backup is created, AWS OpsWorks for Chef
  Automate deletes the oldest backups if this number is exceeded. The default
  value is 1.

* `preferred_backup_window` - (Optional) The start time for a one-hour period
  during which AWS OpsWorks for Chef Automate backs up application-level data
  on your server if automated backups are enabled. Valid values must be
  specified in one of the following formats:

    - `HH:MM` for daily backups
    - `DDD:HH:MM` for weekly backups

  The specified time is in coordinated universal time (UTC). The default value
  is a random, daily start time.

* `preferred_maintenance_window` - (Optional) The start time for a one-hour
  period each week during which AWS OpsWorks for Chef Automate performs
  maintenance on the instance. Valid values must be specified in the following
  format: `DDD:HH:MM`. The specified time is in coordinated universal time
  (UTC). The default value is a random one-hour period on Tuesday, Wednesday,
  or Friday.



## Attribute Reference

The following attributes are exported:

* `arn` - The ARN of the Chef Server.

* `chef_starter_kit` - A Base64-encoded starter kit to connect to Chef.

* `cloudformation_stack_arn` - The ARN of the CloudFormation stack created
  as part of the Chef Server.

* `endpoint` - The DNS name of the Chef server.
