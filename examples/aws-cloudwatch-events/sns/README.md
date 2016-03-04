# CloudWatch Event sent to SNS Topic

This example sets up a CloudWatch Event Rule with a Target and SNS Topic
to send any CloudTrail API operation into that SNS topic. This allows you
to add SNS subscriptions which may notify you about suspicious activity.

See more details about [CloudWatch Events](http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/WhatIsCloudWatchEvents.html)
in the official AWS docs.

## How to run the example

```
terraform apply \
	-var=aws_region=us-west-2
```
