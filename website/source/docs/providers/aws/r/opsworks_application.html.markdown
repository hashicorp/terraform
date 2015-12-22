---
layout: "aws"
page_title: "AWS: aws_opsworks_aplication"
sidebar_current: "docs-aws-resource-opsworks-application"
description: |-
  Provides an OpsWorks application resource.
---

# aws\_opsworks\_application

Provides an OpsWorks application resource.

## Example Usage

```
resource "aws_opsworks_application" "foo-app" {
  name = "foobar application"
  short_name = "foobar"
  stack_id = "${aws_opsworks_stack.stack.id}"
  type = "rails"
  description = "This is a Rails application"
  domains = [
    "example.com",
    "sub.example.com"
  ]
  environment = {
    key = "key"
    value = "value"
    secure = false
  }
  app_source = {
    type = "git"
    revision = "master"
    url = "https://github.com/example.git"
  }
  enable_ssl = true
  ssl_configuration = {
    private_key = "${file("./foobar.key")}"
    certificate = "${file("./foobar.crt")}"
  }
  document_root = "public"
  auto_bundle_on_deploy = true
  rails_env = "staging"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A human-readable name for the application.
* `short_name` - (Required) A short, machine-readable name for the application. This can only be defined on resource creation and ignored on resource update.
* `stack_id` - (Required) The id of the stack the application will belong to.
* `type` - (Required) Opsworks application type. One of `aws-flow-ruby`, `java`, `rails`, `php`, `nodejs`, `static` or `other`.
* `description` - (Optional) A description of the app.
* `environment` - (Optional) Object to define environment variables.  Object is described below.
* `enable_ssl` - (Optional) Whether to enable SSL for the app. This must be set in order to let `ssl_configuration.private_key`, `ssl_configuration.certificate` and `ssl_configuration.chain` take effect.
* `ssl_configuration` - (Optional) The SSL configuration of the app. Object is described below.
* `app_source` - (Optional) SCM configuration of the app as described below.
* `data_source_arn` - (Optional) The data source's ARN.
* `data_source_type` - (Optional) The data source's type one of `AutoSelectOpsworksMysqlInstance`, `OpsworksMysqlInstance`, or `RdsDbInstance`.
* `data_source_database_name` - (Optional) The database name.
* `domains` -  (Optional) A list of virtual host alias.
* `document_root` - (Optional) Subfolder for the document root for application of type `rails`.
* `auto_bundle_on_deploy` - (Optional) Run bundle install when deploying for application of type `rails`.
* `rails_env` - (Required if `type` = `rails`) The name of the Rails environment for application of type `rails`.
* `aws_flow_ruby_settings` - (Optional) Specify activity and workflow workers for your app using the aws-flow gem.

An `app_source` block supports the following arguments (can only be defined once per resource):

* `type` - (Required) The type of source to use. For example, "archive".
* `url` - (Required) The URL where the app resource can be found.
* `username` - (Optional) Username to use when authenticating to the source.
* `password` - (Optional) Password to use when authenticating to the source.
* `ssh_key` - (Optional) SSH key to use when authenticating to the source.
* `revision` - (Optional) For sources that are version-aware, the revision to use.

An `environment` block supports the following arguments:

* `key` - (Required) Variable name.
* `value` - (Required) Variable value.
* `secret` - (Optional) Set visibility of the variable value to `true` or `false`.

A `ssl_configuration` block supports the following arguments (can only be defined once per resource):

* `private_key` - (Required) The private key; the contents of the certificate's domain.key file.
* `certificate` - (Required) The contents of the certificate's domain.crt file.
* `chain` - (Optional)  Can be used to specify an intermediate certificate authority key or client authentication.

## Attributes Reference

The following attributes are exported:

* `id` - The id of the application.
