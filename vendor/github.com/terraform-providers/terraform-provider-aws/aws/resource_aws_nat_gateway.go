package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNatGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNatGatewayCreate,
		Read:   resourceAwsNatGatewayRead,
		Update: resourceAwsNatGatewayUpdate,
		Delete: resourceAwsNatGatewayDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"allocation_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network_interface_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"private_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"public_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsNatGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Create the NAT Gateway
	createOpts := &ec2.CreateNatGatewayInput{
		AllocationId: aws.String(d.Get("allocation_id").(string)),
		SubnetId:     aws.String(d.Get("subnet_id").(string)),
	}

	log.Printf("[DEBUG] Create NAT Gateway: %s", *createOpts)
	natResp, err := conn.CreateNatGateway(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating NAT Gateway: %s", err)
	}

	// Get the ID and store it
	ng := natResp.NatGateway
	d.SetId(*ng.NatGatewayId)
	log.Printf("[INFO] NAT Gateway ID: %s", d.Id())

	// Wait for the NAT Gateway to become available
	log.Printf("[DEBUG] Waiting for NAT Gateway (%s) to become available", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  []string{"available"},
		Refresh: NGStateRefreshFunc(conn, d.Id()),
		Timeout: 10 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for NAT Gateway (%s) to become available: %s", d.Id(), err)
	}

	// Update our attributes and return
	return resourceAwsNatGatewayUpdate(d, meta)
}

func resourceAwsNatGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Refresh the NAT Gateway state
	ngRaw, state, err := NGStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}

	status := map[string]bool{
		"deleted":  true,
		"deleting": true,
		"failed":   true,
	}

	if _, ok := status[strings.ToLower(state)]; ngRaw == nil || ok {
		log.Printf("[INFO] Removing %s from Terraform state as it is not found or in the deleted state.", d.Id())
		d.SetId("")
		return nil
	}

	// Set NAT Gateway attributes
	ng := ngRaw.(*ec2.NatGateway)
	d.Set("subnet_id", ng.SubnetId)

	// Address
	address := ng.NatGatewayAddresses[0]
	d.Set("allocation_id", address.AllocationId)
	d.Set("network_interface_id", address.NetworkInterfaceId)
	d.Set("private_ip", address.PrivateIp)
	d.Set("public_ip", address.PublicIp)

	// Tags
	d.Set("tags", tagsToMap(ng.Tags))

	return nil
}

func resourceAwsNatGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Turn on partial mode
	d.Partial(true)

	if err := setTags(conn, d); err != nil {
		return err
	}
	d.SetPartial("tags")

	d.Partial(false)
	return resourceAwsNatGatewayRead(d, meta)
}

func resourceAwsNatGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	deleteOpts := &ec2.DeleteNatGatewayInput{
		NatGatewayId: aws.String(d.Id()),
	}
	log.Printf("[INFO] Deleting NAT Gateway: %s", d.Id())

	_, err := conn.DeleteNatGateway(deleteOpts)
	if err != nil {
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}

		if ec2err.Code() == "NatGatewayNotFound" {
			return nil
		}

		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{"deleted"},
		Refresh:    NGStateRefreshFunc(conn, d.Id()),
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf("Error waiting for NAT Gateway (%s) to delete: %s", d.Id(), err)
	}

	return nil
}

// NGStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a NAT Gateway.
func NGStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		opts := &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []*string{aws.String(id)},
		}
		resp, err := conn.DescribeNatGateways(opts)
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "NatGatewayNotFound" {
				resp = nil
			} else {
				log.Printf("Error on NGStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		ng := resp.NatGateways[0]
		return ng, *ng.State, nil
	}
}
