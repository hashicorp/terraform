---
layout: "aws"
page_title: "AWS: aws_elastic_beanstalk_solution_stack"
sidebar_current: "docs-aws-datasource-elastic-beanstalk-solution-stack"
description: |-
  Get an elastic beanstalk solution stack.
---

# aws\_elastic\_beanstalk\_solution\_stack

Use this data source to get the name of a elastic beanstalk solution stack.

## Example Usage

```hcl
data "aws_elastic_beanstalk_solution_stack" "multi_docker" {
  most_recent   = true

  name_regex    = "^64bit Amazon Linux (.*) Multi-container Docker (.*)$"
}
```

## Argument Reference

* `most_recent` - (Optional) If more than one result is returned, use the most
recent solution stack.

* `name_regex` - A regex string to apply to the solution stack list returned
by AWS.

~> **NOTE:** If more or less than a single match is returned by the search,
Terraform will fail. Ensure that your search is specific enough to return
a single solution stack, or use `most_recent` to choose the most recent one.

## Attributes Reference

* `name` - The name of the solution stack.
