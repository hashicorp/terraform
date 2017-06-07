package aws

import (
	"fmt"
	"log"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSecurityGroupAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityGroupAttachmentCreate,
		Read:   resourceAwsSecurityGroupAttachmentRead,
		Delete: resourceAwsSecurityGroupAttachmentDelete,
		Schema: map[string]*schema.Schema{
			"security_group_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"network_interface_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSecurityGroupAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	if err := attachSecurityGroupToInterface(d, meta); err != nil {
		return err
	}

	return resourceAwsSecurityGroupAttachmentRead(d, meta)
}

func attachSecurityGroupToInterface(d *schema.ResourceData, meta interface{}) error {
	sgID := d.Get("security_group_id").(string)
	interfaceID := d.Get("network_interface_id").(string)

	log.Printf("[INFO] Attaching security group %s to network interface ID %s", sgID, interfaceID)

	conn := meta.(*AWSClient).ec2conn

	dniParams := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: aws.StringSlice([]string{interfaceID}),
	}

	dniResp, err := conn.DescribeNetworkInterfaces(dniParams)
	if err != nil {
		return err
	}

	return addSGToENI(conn, sgID, dniResp.NetworkInterfaces[0])
}

func fetchNetworkInterface(conn *ec2.EC2, ifaceID string) (*ec2.NetworkInterface, error) {
	dniParams := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: aws.StringSlice([]string{ifaceID}),
	}

	dniResp, err := conn.DescribeNetworkInterfaces(dniParams)
	if err != nil {
		return nil, err
	}
	return dniResp.NetworkInterfaces[0], nil
}

func addSGToENI(conn *ec2.EC2, sgID string, iface *ec2.NetworkInterface) error {
	if sgExistsInENI(sgID, iface) {
		return fmt.Errorf("security group %s already attached to interface ID %s", sgID, *iface.NetworkInterfaceId)
	}
	var groupIDs []string
	for _, v := range iface.Groups {
		groupIDs = append(groupIDs, *v.GroupId)
	}
	groupIDs = append(groupIDs, sgID)
	params := &ec2.ModifyNetworkInterfaceAttributeInput{
		NetworkInterfaceId: iface.NetworkInterfaceId,
		Groups:             aws.StringSlice(groupIDs),
	}

	_, err := conn.ModifyNetworkInterfaceAttribute(params)
	return err
}

func sgExistsInENI(sgID string, iface *ec2.NetworkInterface) bool {
	for _, v := range iface.Groups {
		if *v.GroupId == sgID {
			return true
		}
	}
	return false
}

func resourceAwsSecurityGroupAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	return refreshSecurityGroupWithInterface(d, meta)
}

func refreshSecurityGroupWithInterface(d *schema.ResourceData, meta interface{}) error {
	sgID := d.Get("security_group_id").(string)
	interfaceID := d.Get("network_interface_id").(string)

	log.Printf("[INFO] Checking association of security group %s to network interface ID %s", sgID, interfaceID)

	conn := meta.(*AWSClient).ec2conn

	iface, err := fetchNetworkInterface(conn, interfaceID)
	if err != nil {
		return err
	}

	if sgExistsInENI(sgID, iface) {
		d.SetId(fmt.Sprintf("%s_%s", sgID, interfaceID))
	} else {
		// The assocation does not exist when it should, taint this resource.
		log.Printf("[WARN] Security group %s not associated with network interface ID %s, tainting", sgID, interfaceID)
		d.SetId("")
	}
	return nil
}

func resourceAwsSecurityGroupAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	if err := detachSecurityGroupFromInterface(d, meta); err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func detachSecurityGroupFromInterface(d *schema.ResourceData, meta interface{}) error {
	sgID := d.Get("security_group_id").(string)
	interfaceID := d.Get("network_interface_id").(string)

	log.Printf("[INFO] Removing security group %s from instance ID %s", sgID, interfaceID)

	conn := meta.(*AWSClient).ec2conn

	iface, err := fetchNetworkInterface(conn, interfaceID)
	if err != nil {
		return err
	}

	return delSGFromENI(conn, sgID, iface)
}

func delSGFromENI(conn *ec2.EC2, sgID string, iface *ec2.NetworkInterface) error {
	old := iface.Groups
	var new []*string
	for _, v := range iface.Groups {
		if *v.GroupId == sgID {
			continue
		}
		new = append(new, v.GroupId)
	}
	if reflect.DeepEqual(old, new) {
		// The interface already didn't have the security group, nothing to do
		return nil
	}

	params := &ec2.ModifyNetworkInterfaceAttributeInput{
		NetworkInterfaceId: iface.NetworkInterfaceId,
		Groups:             new,
	}

	_, err := conn.ModifyNetworkInterfaceAttribute(params)
	return err
}
