# CloudWatch Event sent to Kinesis Stream

This example sets up a CloudWatch Event Rule with a Target and IAM Role & Policy
to send all autoscaling events into Kinesis stream for further examination.

See more details about [CloudWatch Events](http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/WhatIsCloudWatchEvents.html)
in the official AWS docs.

## How to run the example

```
terraform apply \
	-var=aws_region=us-west-2
```
