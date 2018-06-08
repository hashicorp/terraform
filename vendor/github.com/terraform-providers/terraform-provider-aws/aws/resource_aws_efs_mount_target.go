package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEfsMountTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEfsMountTargetCreate,
		Read:   resourceAwsEfsMountTargetRead,
		Update: resourceAwsEfsMountTargetUpdate,
		Delete: resourceAwsEfsMountTargetDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"file_system_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},

			"security_groups": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Computed: true,
				Optional: true,
			},

			"subnet_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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

func resourceAwsEfsMountTargetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	fsId := d.Get("file_system_id").(string)
	subnetId := d.Get("subnet_id").(string)

	// CreateMountTarget would return the same Mount Target ID
	// to parallel requests if they both include the same AZ
	// and we would end up managing the same MT as 2 resources.
	// So we make it fail by calling 1 request per AZ at a time.
	az, err := getAzFromSubnetId(subnetId, meta.(*AWSClient).ec2conn)
	if err != nil {
		return fmt.Errorf("Failed getting Availability Zone from subnet ID (%s): %s", subnetId, err)
	}
	mtKey := "efs-mt-" + fsId + "-" + az
	awsMutexKV.Lock(mtKey)
	defer awsMutexKV.Unlock(mtKey)

	input := efs.CreateMountTargetInput{
		FileSystemId: aws.String(fsId),
		SubnetId:     aws.String(subnetId),
	}

	if v, ok := d.GetOk("ip_address"); ok {
		input.IpAddress = aws.String(v.(string))
	}
	if v, ok := d.GetOk("security_groups"); ok {
		input.SecurityGroups = expandStringList(v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] Creating EFS mount target: %#v", input)

	mt, err := conn.CreateMountTarget(&input)
	if err != nil {
		return err
	}

	d.SetId(*mt.MountTargetId)
	log.Printf("[INFO] EFS mount target ID: %s", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending: []string{"creating"},
		Target:  []string{"available"},
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeMountTargets(&efs.DescribeMountTargetsInput{
				MountTargetId: aws.String(d.Id()),
			})
			if err != nil {
				return nil, "error", err
			}

			if hasEmptyMountTargets(resp) {
				return nil, "error", fmt.Errorf("EFS mount target %q could not be found.", d.Id())
			}

			mt := resp.MountTargets[0]

			log.Printf("[DEBUG] Current status of %q: %q", *mt.MountTargetId, *mt.LifeCycleState)
			return mt, *mt.LifeCycleState, nil
		},
		Timeout:    10 * time.Minute,
		Delay:      2 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for EFS mount target (%s) to create: %s", d.Id(), err)
	}

	log.Printf("[DEBUG] EFS mount target created: %s", *mt.MountTargetId)

	return resourceAwsEfsMountTargetRead(d, meta)
}

func resourceAwsEfsMountTargetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	if d.HasChange("security_groups") {
		input := efs.ModifyMountTargetSecurityGroupsInput{
			MountTargetId:  aws.String(d.Id()),
			SecurityGroups: expandStringList(d.Get("security_groups").(*schema.Set).List()),
		}
		_, err := conn.ModifyMountTargetSecurityGroups(&input)
		if err != nil {
			return err
		}
	}

	return resourceAwsEfsMountTargetRead(d, meta)
}

func resourceAwsEfsMountTargetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn
	resp, err := conn.DescribeMountTargets(&efs.DescribeMountTargetsInput{
		MountTargetId: aws.String(d.Id()),
	})
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "MountTargetNotFound" {
			// The EFS mount target could not be found,
			// which would indicate that it might be
			// already deleted.
			log.Printf("[WARN] EFS mount target %q could not be found.", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading EFS mount target %s: %s", d.Id(), err)
	}

	if hasEmptyMountTargets(resp) {
		return fmt.Errorf("EFS mount target %q could not be found.", d.Id())
	}

	mt := resp.MountTargets[0]

	log.Printf("[DEBUG] Found EFS mount target: %#v", mt)

	d.SetId(*mt.MountTargetId)
	d.Set("file_system_id", mt.FileSystemId)
	d.Set("ip_address", mt.IpAddress)
	d.Set("subnet_id", mt.SubnetId)
	d.Set("network_interface_id", mt.NetworkInterfaceId)

	sgResp, err := conn.DescribeMountTargetSecurityGroups(&efs.DescribeMountTargetSecurityGroupsInput{
		MountTargetId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	err = d.Set("security_groups", schema.NewSet(schema.HashString, flattenStringList(sgResp.SecurityGroups)))
	if err != nil {
		return err
	}

	// DNS name per http://docs.aws.amazon.com/efs/latest/ug/mounting-fs-mount-cmd-dns-name.html
	_, err = getAzFromSubnetId(*mt.SubnetId, meta.(*AWSClient).ec2conn)
	if err != nil {
		return fmt.Errorf("Failed getting Availability Zone from subnet ID (%s): %s", *mt.SubnetId, err)
	}

	region := meta.(*AWSClient).region
	err = d.Set("dns_name", resourceAwsEfsMountTargetDnsName(*mt.FileSystemId, region))
	if err != nil {
		return err
	}

	return nil
}

func getAzFromSubnetId(subnetId string, conn *ec2.EC2) (string, error) {
	input := ec2.DescribeSubnetsInput{
		SubnetIds: []*string{aws.String(subnetId)},
	}
	out, err := conn.DescribeSubnets(&input)
	if err != nil {
		return "", err
	}

	if l := len(out.Subnets); l != 1 {
		return "", fmt.Errorf("Expected exactly 1 subnet returned for %q, got: %d", subnetId, l)
	}

	return *out.Subnets[0].AvailabilityZone, nil
}

func resourceAwsEfsMountTargetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	log.Printf("[DEBUG] Deleting EFS mount target %q", d.Id())
	_, err := conn.DeleteMountTarget(&efs.DeleteMountTargetInput{
		MountTargetId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{"available", "deleting", "deleted"},
		Target:  []string{},
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeMountTargets(&efs.DescribeMountTargetsInput{
				MountTargetId: aws.String(d.Id()),
			})
			if err != nil {
				awsErr, ok := err.(awserr.Error)
				if !ok {
					return nil, "error", err
				}

				if awsErr.Code() == "MountTargetNotFound" {
					return nil, "", nil
				}

				return nil, "error", awsErr
			}

			if hasEmptyMountTargets(resp) {
				return nil, "", nil
			}

			mt := resp.MountTargets[0]

			log.Printf("[DEBUG] Current status of %q: %q", *mt.MountTargetId, *mt.LifeCycleState)
			return mt, *mt.LifeCycleState, nil
		},
		Timeout:    10 * time.Minute,
		Delay:      2 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for EFS mount target (%q) to delete: %s",
			d.Id(), err.Error())
	}

	log.Printf("[DEBUG] EFS mount target %q deleted.", d.Id())

	return nil
}

func resourceAwsEfsMountTargetDnsName(fileSystemId, region string) string {
	return fmt.Sprintf("%s.efs.%s.amazonaws.com", fileSystemId, region)
}

func hasEmptyMountTargets(mto *efs.DescribeMountTargetsOutput) bool {
	if mto != nil && len(mto.MountTargets) > 0 {
		return false
	}
	return true
}
