package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEip() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEipRead,

		Schema: map[string]*schema.Schema{
			"association_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"domain": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"filter": ec2CustomFiltersSchema(),
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"instance_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"network_interface_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"network_interface_owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"private_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_ip": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"public_ipv4_pool": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsEipRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeAddressesInput{}

	if v, ok := d.GetOk("id"); ok {
		req.AllocationIds = []*string{aws.String(v.(string))}
	}

	if v, ok := d.GetOk("public_ip"); ok {
		req.PublicIps = []*string{aws.String(v.(string))}
	}

	req.Filters = []*ec2.Filter{}

	req.Filters = append(req.Filters, buildEC2CustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)

	req.Filters = append(req.Filters, buildEC2TagFilterList(
		tagsFromMap(d.Get("tags").(map[string]interface{})),
	)...)

	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] Reading EIP: %s", req)
	resp, err := conn.DescribeAddresses(req)
	if err != nil {
		return fmt.Errorf("error describing EC2 Address: %s", err)
	}
	if resp == nil || len(resp.Addresses) == 0 {
		return fmt.Errorf("no matching Elastic IP found")
	}
	if len(resp.Addresses) > 1 {
		return fmt.Errorf("multiple Elastic IPs matched; use additional constraints to reduce matches to a single Elastic IP")
	}

	eip := resp.Addresses[0]

	if aws.StringValue(eip.Domain) == ec2.DomainTypeVpc {
		d.SetId(aws.StringValue(eip.AllocationId))
	} else {
		log.Printf("[DEBUG] Reading EIP, has no AllocationId, this means we have a Classic EIP, the id will also be the public ip : %s", req)
		d.SetId(aws.StringValue(eip.PublicIp))
	}

	d.Set("association_id", eip.AssociationId)
	d.Set("domain", eip.Domain)
	d.Set("instance_id", eip.InstanceId)
	d.Set("network_interface_id", eip.NetworkInterfaceId)
	d.Set("network_interface_owner_id", eip.NetworkInterfaceOwnerId)
	d.Set("private_ip", eip.PrivateIpAddress)
	d.Set("public_ip", eip.PublicIp)
	d.Set("public_ipv4_pool", eip.PublicIpv4Pool)
	d.Set("tags", tagsToMap(eip.Tags))

	return nil
}
