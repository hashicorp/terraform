# AWS Networking Example

This example creates AWS VPC resources, making a VPC in each of two regions and
then two subnets in each VPC in two different availability zones.

This example also demonstrates the use of modules to create several copies of
the same resource set with different arguments. The child modules in this
directory are:

* `region`: container module for all of the network resources within a region. This is instantiated once per region.
* `subnet`: represents a subnet within a given availability zone. This is instantiated twice per region, using the first two availability zones supported within the target AWS account.
