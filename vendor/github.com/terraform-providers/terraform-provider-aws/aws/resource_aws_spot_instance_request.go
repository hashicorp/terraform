package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSpotInstanceRequest() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSpotInstanceRequestCreate,
		Read:   resourceAwsSpotInstanceRequestRead,
		Delete: resourceAwsSpotInstanceRequestDelete,
		Update: resourceAwsSpotInstanceRequestUpdate,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: func() map[string]*schema.Schema {
			// The Spot Instance Request Schema is based on the AWS Instance schema.
			s := resourceAwsInstance().Schema

			// Everything on a spot instance is ForceNew except tags
			for k, v := range s {
				if k == "tags" {
					continue
				}
				v.ForceNew = true
			}

			s["volume_tags"] = &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			}

			s["spot_price"] = &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			}
			s["spot_type"] = &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "persistent",
			}
			s["wait_for_fulfillment"] = &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			}
			s["launch_group"] = &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			}
			s["spot_bid_status"] = &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			}
			s["spot_request_state"] = &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			}
			s["spot_instance_id"] = &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			}
			s["block_duration_minutes"] = &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			}
			s["instance_interruption_behaviour"] = &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "terminate",
				ForceNew: true,
			}
			return s
		}(),
	}
}

func resourceAwsSpotInstanceRequestCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	instanceOpts, err := buildAwsInstanceOpts(d, meta)
	if err != nil {
		return err
	}

	spotOpts := &ec2.RequestSpotInstancesInput{
		SpotPrice: aws.String(d.Get("spot_price").(string)),
		Type:      aws.String(d.Get("spot_type").(string)),
		InstanceInterruptionBehavior: aws.String(d.Get("instance_interruption_behaviour").(string)),

		// Though the AWS API supports creating spot instance requests for multiple
		// instances, for TF purposes we fix this to one instance per request.
		// Users can get equivalent behavior out of TF's "count" meta-parameter.
		InstanceCount: aws.Int64(1),

		LaunchSpecification: &ec2.RequestSpotLaunchSpecification{
			BlockDeviceMappings: instanceOpts.BlockDeviceMappings,
			EbsOptimized:        instanceOpts.EBSOptimized,
			Monitoring:          instanceOpts.Monitoring,
			IamInstanceProfile:  instanceOpts.IAMInstanceProfile,
			ImageId:             instanceOpts.ImageID,
			InstanceType:        instanceOpts.InstanceType,
			KeyName:             instanceOpts.KeyName,
			Placement:           instanceOpts.SpotPlacement,
			SecurityGroupIds:    instanceOpts.SecurityGroupIDs,
			SecurityGroups:      instanceOpts.SecurityGroups,
			SubnetId:            instanceOpts.SubnetID,
			UserData:            instanceOpts.UserData64,
			NetworkInterfaces:   instanceOpts.NetworkInterfaces,
		},
	}

	if v, ok := d.GetOk("block_duration_minutes"); ok {
		spotOpts.BlockDurationMinutes = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("launch_group"); ok {
		spotOpts.LaunchGroup = aws.String(v.(string))
	}

	// Make the spot instance request
	log.Printf("[DEBUG] Requesting spot bid opts: %s", spotOpts)

	var resp *ec2.RequestSpotInstancesOutput
	err = resource.Retry(15*time.Second, func() *resource.RetryError {
		var err error
		resp, err = conn.RequestSpotInstances(spotOpts)
		// IAM instance profiles can take ~10 seconds to propagate in AWS:
		// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
		if isAWSErr(err, "InvalidParameterValue", "Invalid IAM Instance Profile") {
			log.Printf("[DEBUG] Invalid IAM Instance Profile referenced, retrying...")
			return resource.RetryableError(err)
		}
		// IAM roles can also take time to propagate in AWS:
		if isAWSErr(err, "InvalidParameterValue", " has no associated IAM Roles") {
			log.Printf("[DEBUG] IAM Instance Profile appears to have no IAM roles, retrying...")
			return resource.RetryableError(err)
		}
		return resource.NonRetryableError(err)
	})

	if err != nil {
		return fmt.Errorf("Error requesting spot instances: %s", err)
	}
	if len(resp.SpotInstanceRequests) != 1 {
		return fmt.Errorf(
			"Expected response with length 1, got: %s", resp)
	}

	sir := *resp.SpotInstanceRequests[0]
	d.SetId(*sir.SpotInstanceRequestId)

	if d.Get("wait_for_fulfillment").(bool) {
		spotStateConf := &resource.StateChangeConf{
			// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-bid-status.html
			Pending:    []string{"start", "pending-evaluation", "pending-fulfillment"},
			Target:     []string{"fulfilled"},
			Refresh:    SpotInstanceStateRefreshFunc(conn, sir),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		log.Printf("[DEBUG] waiting for spot bid to resolve... this may take several minutes.")
		_, err = spotStateConf.WaitForState()

		if err != nil {
			return fmt.Errorf("Error while waiting for spot request (%s) to resolve: %s", sir, err)
		}
	}

	return resourceAwsSpotInstanceRequestUpdate(d, meta)
}

// Update spot state, etc
func resourceAwsSpotInstanceRequestRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []*string{aws.String(d.Id())},
	}
	resp, err := conn.DescribeSpotInstanceRequests(req)

	if err != nil {
		// If the spot request was not found, return nil so that we can show
		// that it is gone.
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidSpotInstanceRequestID.NotFound" {
			d.SetId("")
			return nil
		}

		// Some other error, report it
		return err
	}

	// If nothing was found, then return no state
	if len(resp.SpotInstanceRequests) == 0 {
		d.SetId("")
		return nil
	}

	request := resp.SpotInstanceRequests[0]

	// if the request is cancelled or closed, then it is gone
	if *request.State == "cancelled" || *request.State == "closed" {
		d.SetId("")
		return nil
	}

	d.Set("spot_bid_status", *request.Status.Code)
	// Instance ID is not set if the request is still pending
	if request.InstanceId != nil {
		d.Set("spot_instance_id", *request.InstanceId)
		// Read the instance data, setting up connection information
		if err := readInstance(d, meta); err != nil {
			return fmt.Errorf("[ERR] Error reading Spot Instance Data: %s", err)
		}
	}

	d.Set("spot_request_state", request.State)
	d.Set("launch_group", request.LaunchGroup)
	d.Set("block_duration_minutes", request.BlockDurationMinutes)
	d.Set("tags", tagsToMap(request.Tags))
	d.Set("instance_interruption_behaviour", request.InstanceInterruptionBehavior)

	return nil
}

func readInstance(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(d.Get("spot_instance_id").(string))},
	})
	if err != nil {
		// If the instance was not found, return nil so that we can show
		// that the instance is gone.
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidInstanceID.NotFound" {
			return fmt.Errorf("no instance found")
		}

		// Some other error, report it
		return err
	}

	// If nothing was found, then return no state
	if len(resp.Reservations) == 0 {
		return fmt.Errorf("no instances found")
	}

	instance := resp.Reservations[0].Instances[0]

	// Set these fields for connection information
	if instance != nil {
		d.Set("public_dns", instance.PublicDnsName)
		d.Set("public_ip", instance.PublicIpAddress)
		d.Set("private_dns", instance.PrivateDnsName)
		d.Set("private_ip", instance.PrivateIpAddress)

		// set connection information
		if instance.PublicIpAddress != nil {
			d.SetConnInfo(map[string]string{
				"type": "ssh",
				"host": *instance.PublicIpAddress,
			})
		} else if instance.PrivateIpAddress != nil {
			d.SetConnInfo(map[string]string{
				"type": "ssh",
				"host": *instance.PrivateIpAddress,
			})
		}
		if err := readBlockDevices(d, instance, conn); err != nil {
			return err
		}

		var ipv6Addresses []string
		if len(instance.NetworkInterfaces) > 0 {
			for _, ni := range instance.NetworkInterfaces {
				if *ni.Attachment.DeviceIndex == 0 {
					d.Set("subnet_id", ni.SubnetId)
					d.Set("network_interface_id", ni.NetworkInterfaceId)
					d.Set("associate_public_ip_address", ni.Association != nil)
					d.Set("ipv6_address_count", len(ni.Ipv6Addresses))

					for _, address := range ni.Ipv6Addresses {
						ipv6Addresses = append(ipv6Addresses, *address.Ipv6Address)
					}
				}
			}
		} else {
			d.Set("subnet_id", instance.SubnetId)
			d.Set("network_interface_id", "")
		}

		if err := d.Set("ipv6_addresses", ipv6Addresses); err != nil {
			log.Printf("[WARN] Error setting ipv6_addresses for AWS Spot Instance (%s): %s", d.Id(), err)
		}
	}

	return nil
}

func resourceAwsSpotInstanceRequestUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	d.Partial(true)
	if err := setTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsSpotInstanceRequestRead(d, meta)
}

func resourceAwsSpotInstanceRequestDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Cancelling spot request: %s", d.Id())
	_, err := conn.CancelSpotInstanceRequests(&ec2.CancelSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []*string{aws.String(d.Id())},
	})

	if err != nil {
		return fmt.Errorf("Error cancelling spot request (%s): %s", d.Id(), err)
	}

	if instanceId := d.Get("spot_instance_id").(string); instanceId != "" {
		log.Printf("[INFO] Terminating instance: %s", instanceId)
		if err := awsTerminateInstance(conn, instanceId, d); err != nil {
			return fmt.Errorf("Error terminating spot instance: %s", err)
		}
	}

	return nil
}

// SpotInstanceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an EC2 spot instance request
func SpotInstanceStateRefreshFunc(
	conn *ec2.EC2, sir ec2.SpotInstanceRequest) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		resp, err := conn.DescribeSpotInstanceRequests(&ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: []*string{sir.SpotInstanceRequestId},
		})

		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidSpotInstanceRequestID.NotFound" {
				// Set this to nil as if we didn't find anything.
				resp = nil
			} else {
				log.Printf("Error on StateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil || len(resp.SpotInstanceRequests) == 0 {
			// Sometimes AWS just has consistency issues and doesn't see
			// our request yet. Return an empty state.
			return nil, "", nil
		}

		req := resp.SpotInstanceRequests[0]
		return req, *req.Status.Code, nil
	}
}
