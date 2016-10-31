package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsElbAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElbAttachmentCreate,
		Read:   resourceAwsElbAttachmentRead,
		Delete: resourceAwsElbAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"elb": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"instance": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
		},
	}
}

func resourceAwsElbAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	elbName := d.Get("elb").(string)

	instance := d.Get("instance").(string)

	registerInstancesOpts := elb.RegisterInstancesWithLoadBalancerInput{
		LoadBalancerName: aws.String(elbName),
		Instances:        []*elb.Instance{{InstanceId: aws.String(instance)}},
	}

	log.Printf("[INFO] registering instance %s with ELB %s", instance, elbName)

	_, err := elbconn.RegisterInstancesWithLoadBalancer(&registerInstancesOpts)
	if err != nil {
		return fmt.Errorf("Failure registering instances with ELB: %s", err)
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", elbName)))

	return nil
}

func resourceAwsElbAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	elbName := d.Get("elb").(string)

	// only add the instance that was previously defined for this resource
	expected := d.Get("instance").(string)

	// Retrieve the ELB properties to get a list of attachments
	describeElbOpts := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String(elbName)},
	}

	resp, err := elbconn.DescribeLoadBalancers(describeElbOpts)
	if err != nil {
		if isLoadBalancerNotFound(err) {
			log.Printf("[ERROR] ELB %s not found", elbName)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving ELB: %s", err)
	}
	if len(resp.LoadBalancerDescriptions) != 1 {
		log.Printf("[ERROR] Unable to find ELB: %s", resp.LoadBalancerDescriptions)
		d.SetId("")
		return nil
	}

	// only set the instance Id that this resource manages
	found := false
	for _, i := range resp.LoadBalancerDescriptions[0].Instances {
		if expected == *i.InstanceId {
			d.Set("instance", expected)
			found = true
		}
	}

	if !found {
		log.Printf("[WARN] instance %s not found in elb attachments", expected)
		d.SetId("")
	}

	return nil
}

func resourceAwsElbAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	elbName := d.Get("elb").(string)

	instance := d.Get("instance").(string)

	log.Printf("[INFO] Deleting Attachment %s from: %s", instance, elbName)

	deRegisterInstancesOpts := elb.DeregisterInstancesFromLoadBalancerInput{
		LoadBalancerName: aws.String(elbName),
		Instances:        []*elb.Instance{{InstanceId: aws.String(instance)}},
	}

	_, err := elbconn.DeregisterInstancesFromLoadBalancer(&deRegisterInstancesOpts)
	if err != nil {
		return fmt.Errorf("Failure deregistering instances from ELB: %s", err)
	}

	return nil
}
