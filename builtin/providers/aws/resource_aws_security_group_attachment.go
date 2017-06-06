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
			"instance_id": {
				Type:     schema.TypeString,
				Optional: true,
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
	var err error
	switch {
	case d.Get("instance_id").(string) != "":
		err = attachSecurityGroupToInstance(d, meta)
	case d.Get("network_interface_id").(string) != "":
		err = attachSecurityGroupToInterface(d, meta)
	default:
		err = fmt.Errorf("one of instance_id or network_interface_id needs to be defined")
	}
	if err != nil {
		return err
	}

	return resourceAwsSecurityGroupAttachmentRead(d, meta)
}

func attachSecurityGroupToInstance(d *schema.ResourceData, meta interface{}) error {
	sgID := d.Get("security_group_id").(string)
	instanceID := d.Get("instance_id").(string)

	log.Printf("[INFO] Attaching security group %s to instance ID %s", sgID, instanceID)

	conn := meta.(*AWSClient).ec2conn

	iface, err := fetchPrimaryNetworkInterface(conn, instanceID)
	if err != nil {
		return err
	}

	return addSGToENI(conn, sgID, iface)
}

func fetchPrimaryNetworkInterface(conn *ec2.EC2, instanceID string) (*ec2.NetworkInterface, error) {
	diParams := &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	}

	diResp, err := conn.DescribeInstances(diParams)
	if err != nil {
		return nil, err
	}

	instance := diResp.Reservations[0].Instances[0]

	var primaryInterface ec2.InstanceNetworkInterface
	for _, ni := range instance.NetworkInterfaces {
		if *ni.Attachment.DeviceIndex == 0 {
			primaryInterface = *ni
		}
	}

	if primaryInterface.NetworkInterfaceId == nil {
		return nil, fmt.Errorf("instance ID %s, does not contain a primary network interface", instanceID)
	}

	return fetchNetworkInterface(conn, *primaryInterface.NetworkInterfaceId)
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

func addSGToENI(conn *ec2.EC2, sgID string, iface *ec2.NetworkInterface) error {
	if sgExistsInENI(conn, sgID, iface) {
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

func sgExistsInENI(conn *ec2.EC2, sgID string, iface *ec2.NetworkInterface) bool {
	for _, v := range iface.Groups {
		if *v.GroupId == sgID {
			return true
		}
	}
	return false
}

func resourceAwsSecurityGroupAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	switch {
	case d.Get("instance_id").(string) != "":
		return refreshSecurityGroupWithInstance(d, meta)
	case d.Get("network_interface_id").(string) != "":
		return refreshSecurityGroupWithInterface(d, meta)
	}
	return fmt.Errorf("one of instance_id or network_interface_id needs to be defined")
}

func refreshSecurityGroupWithInstance(d *schema.ResourceData, meta interface{}) error {
	sgID := d.Get("security_group_id").(string)
	instanceID := d.Get("instance_id").(string)

	log.Printf("[INFO] Checking association of security group %s to instance ID %s", sgID, instanceID)

	conn := meta.(*AWSClient).ec2conn

	iface, err := fetchPrimaryNetworkInterface(conn, instanceID)
	if err != nil {
		return err
	}

	if sgExistsInENI(conn, sgID, iface) {
		d.SetId(fmt.Sprintf("%s_%s", sgID, instanceID))
	} else {
		// The assocation does not exist when it should, taint this resource.
		log.Printf("[WARN] Security group %s not associated with instance ID %s, tainting", sgID, instanceID)
		d.SetId("")
	}
	return nil
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

	if sgExistsInENI(conn, sgID, iface) {
		d.SetId(fmt.Sprintf("%s_%s", sgID, interfaceID))
	} else {
		// The assocation does not exist when it should, taint this resource.
		log.Printf("[WARN] Security group %s not associated with network interface ID %s, tainting", sgID, interfaceID)
		d.SetId("")
	}
	return nil
}

func resourceAwsSecurityGroupAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	var err error
	switch {
	case d.Get("instance_id").(string) != "":
		err = detachSecurityGroupFromInstance(d, meta)
	case d.Get("network_interface_id").(string) != "":
		err = detachSecurityGroupFromInterface(d, meta)
	}
	if err != nil {
		return err
	}

	// We are done. It's possible that the resource had a broken config and the
	// switch fell through, but if that's the case, then there's nothing to do
	// here anyway.
	d.SetId("")
	return nil
}

func detachSecurityGroupFromInstance(d *schema.ResourceData, meta interface{}) error {
	sgID := d.Get("security_group_id").(string)
	instanceID := d.Get("instance_id").(string)

	log.Printf("[INFO] Removing security group %s from instance ID %s", sgID, instanceID)

	conn := meta.(*AWSClient).ec2conn

	iface, err := fetchPrimaryNetworkInterface(conn, instanceID)
	if err != nil {
		return err
	}

	return delSGFromENI(conn, sgID, iface)
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
