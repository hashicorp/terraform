package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEfsMountTarget() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEfsMountTargetRead,

		Schema: map[string]*schema.Schema{
			"mount_target_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"file_system_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"security_groups": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Computed: true,
			},
			"subnet_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"network_interface_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsEfsMountTargetRead(d *schema.ResourceData, meta interface{}) error {
	efsconn := meta.(*AWSClient).efsconn

	describeEfsOpts := &efs.DescribeMountTargetsInput{
		MountTargetId: aws.String(d.Get("mount_target_id").(string)),
	}

	log.Printf("[DEBUG] Reading EFS Mount Target: %s", describeEfsOpts)
	resp, err := efsconn.DescribeMountTargets(describeEfsOpts)
	if err != nil {
		return fmt.Errorf("Error retrieving EFS Mount Target: %s", err)
	}
	if len(resp.MountTargets) != 1 {
		return fmt.Errorf("Search returned %d results, please revise so only one is returned", len(resp.MountTargets))
	}

	mt := resp.MountTargets[0]

	log.Printf("[DEBUG] Found EFS mount target: %#v", mt)

	d.SetId(*mt.MountTargetId)
	d.Set("file_system_id", mt.FileSystemId)
	d.Set("ip_address", mt.IpAddress)
	d.Set("subnet_id", mt.SubnetId)
	d.Set("network_interface_id", mt.NetworkInterfaceId)

	sgResp, err := efsconn.DescribeMountTargetSecurityGroups(&efs.DescribeMountTargetSecurityGroupsInput{
		MountTargetId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	err = d.Set("security_groups", schema.NewSet(schema.HashString, flattenStringList(sgResp.SecurityGroups)))
	if err != nil {
		return err
	}

	if err := d.Set("dns_name", resourceAwsEfsMountTargetDnsName(*mt.FileSystemId, meta.(*AWSClient).region)); err != nil {
		return fmt.Errorf("Error setting dns_name error: %#v", err)
	}

	return nil
}
