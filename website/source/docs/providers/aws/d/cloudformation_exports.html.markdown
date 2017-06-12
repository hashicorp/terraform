---
layout: "aws"
page_title: "AWS: aws_cloudformation_exports"
sidebar_current: "docs-aws-datasource-cloudformation-exports"
description: |-
    Provides metadata of a CloudFormation Exports (e.g. Cross Stack References)
---

# aws\_cloudformation\_exports

The CloudFormation Exports data source allows access to stack
exports specified in the [Output](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/outputs-section-structure.html) section of the Cloudformation Template using the optional Export Property. 

 -> Note: The data resource runs before the Terraform `resource` declarations. For this reason, the exports must have been created prior to the Terraform `apply`. If you are trying to use a value from a Cloudformation Stack in the same Terraform run please use normal interpolation or Cloudformation Outputs. 

## Example Usage

```hcl
data "aws_cloudformation_exports" "us_west_2" { }

resource "aws_instance" "web" {
  ami           = "ami-abb07bcb"
  instance_type = "t1.micro"
  subnet_id     = "${data.aws_cloudformation_exports.us_west_2.values["MyVpcSubnetId"]}"

  tags {
    Name = "HelloWorld"
    DependsOnStack = "${data.aws_cloudformation_exports.us_west_2.stack_ids["MyVpcSubnetId"]}"
  }
}
```

## Argument Reference

 There are no arguments

## Attributes Reference

The following attributes are exported:

* `values` - A map of values from Cloudformation exports keyed by the name equivalent of creating a map using `Name:Value` from [list-exports](http://docs.aws.amazon.com/cli/latest/reference/cloudformation/list-exports.html)
* `stack_ids` - A map of stack Ids (AWS ARNs) equivalent of creating a map using
    `Name:ExportingStackId` from [list-exports](http://docs.aws.amazon.com/cli/latest/reference/cloudformation/list-exports.html) 
