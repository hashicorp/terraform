---
layout: "docs"
page_title: "Resource Templates"
sidebar_current: "docs-config-resource-templates"
---

# Resource Templates

A resource template is a generic construct which lives inside of a Terraform
configuration file, and holds key-value configuration pairs which can be applied
to any number of resources. The two most common use cases for resource templates
are:

* Reduce repetitive, constant configuration values
* Easily change common configuration across many resources

A resource template takes a single argument defining its unique name, and can
contain any number of configuration items to be applied to resources. A basic
resource template looks like this:

```
resource_template "aws-web" {
    ami = "ami-408c7f28"
    instance_type = "t1.micro"
    key_name = "web-key"
    availability_zone = "us-west-2"
    subnet_id = "subnet-9d4a7b6c"
}
```

As you can see, the resource template named "aws-web" holds many common
configuration items for a set of resources.

## Template Application

Applying a resource template to a resource is done using the `resource_template`
option within any resource. For example:

```
resource "aws_instance" "web" {
    resource_template = "aws-web"
}
```

By including the `aws-web` resource template, the per-resource configuration
becomes much more simple. The above example is the most basic usage of resource
templates.

You can also overwrite individual values from the template on a given resource.
This is done by simply adding the configuration value you want to overwrite into
the resource itself, like so:

```
resource "aws_instance" "web" {
    resource_template = "aws-web"
    instance_type = "m1.small"
}
```

Now with the above configuration, the `web` instance will be deployed exactly
like the template says, except that it will use an instance_type of `m1.small`
instead of `t1.micro`.
