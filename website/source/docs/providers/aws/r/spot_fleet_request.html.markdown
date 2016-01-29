---
layout: "aws"
page_title: "AWS: aws_spot_fleet_request"
sidebar_current: "docs-aws-resource-spot-fleet-request"
description: |-
  Provides a Spot Fleet Request resource.
---

# aws\_spot\_fleet\_request

Provides an EC2 Spot Fleet Request resource. This allows a fleet of spot
instances to be requested on the spot market.

## Example Usage

# Request a simple spot fleet
```
resource "aws_spot_fleet_request" "spotfleettest" {
    iam_fleet_role = "arn:aws:iam::1234:role/spot-fleet"
    spot_price = "0.03"
    target_capacity = 3
    valid_until = "2019-11-04T20:44:20.000Z"
    launch_specification {
        instance_type = "m1.large"
        ami = "ami-abc"
        spot_price = "0.01"
        availability_zone = "us-west-1a"
        weighted_capacity = 75
    }
     launch_specification {
        instance_type = "m1.xlarge"
        ami = "ami-abc"
        spot_price = "0.01"
        availability_zone = "us-west-1b"
        weighted_capacity = 25
    }
}
```

## Argument Reference

Most of these arguments directly correspond to the
[offical API](http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SpotFleetRequestConfigData.html).

* `iam_fleet_role` - (required) Grants the Spot fleet permission to terminate
  Spot instances on your behalf when you cancel its Spot fleet request using
CancelSpotFleetRequests or when the Spot fleet request expires, if you set
terminateInstancesWithExpiration.
* `launch_specification` - Used to define the launch configuration of the
  spot-fleet request. Can be specified multiple times to define different bids
across different markets and instance types. Note: takes in similar but not
idential inputs as [`aws_instance`](instance.html).  There are limitations on
what you can specify, however. (tags, for example, are not supported) See the
list of officially supported inputs in the
[reference documentation](http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SpotFleetLaunchSpecification.html),
any normal [`aws_instance`](instance.html) parameter that corresponds to those
inputs may be used.
* `spot_price` - (required) The bid price per unit hour.
* `target_capacity` - The number of units to request. You can choose to set the
  target capacity in terms of instances or a performance characteristic that is
important to your application workload, such as vCPUs, memory, or I/O.
* `allocation_strategy` - Indicates how to allocate the target capacity across
  the Spot pools specified by the Spot fleet request. The default is
lowestPrice.
* `excess_capacity_termination_policy` - Indicates whether running Spot
  instances should be terminated if the target capacity of the Spot fleet
request is decreased below the current size of the Spot fleet. 
* `terminate_instances_with_expiration` -Indicates whether running Spot
  instances should be terminated when the Spot fleet request expires.
* `valid_until` - The end date and time of the request, in UTC ISO8601 format
  (for example, YYYY-MM-DDTHH:MM:SSZ). At this point, no new Spot instance
requests are placed or enabled to fulfill the request. Defaults to 24 hours.


## Attributes Reference

The following attributes are exported:

* `id` - The Spot Instance Request ID.
* `spot_request_state` - The Spot Instance Request ID.
