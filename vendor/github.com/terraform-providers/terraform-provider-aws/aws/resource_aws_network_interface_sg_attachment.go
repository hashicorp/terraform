package aws

import (
	"fmt"
	"log"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNetworkInterfaceSGAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNetworkInterfaceSGAttachmentCreate,
		Read:   resourceAwsNetworkInterfaceSGAttachmentRead,
		Delete: resourceAwsNetworkInterfaceSGAttachmentDelete,
		Schema: map[string]*schema.Schema{
			"security_group_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"network_interface_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsNetworkInterfaceSGAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	mk := "network_interface_sg_attachment_" + d.Get("network_interface_id").(string)
	awsMutexKV.Lock(mk)
	defer awsMutexKV.Unlock(mk)

	sgID := d.Get("security_group_id").(string)
	interfaceID := d.Get("network_interface_id").(string)

	conn := meta.(*AWSClient).ec2conn

	// Fetch the network interface we will be working with.
	iface, err := fetchNetworkInterface(conn, interfaceID)
	if err != nil {
		return err
	}

	// Add the security group to the network interface.
	log.Printf("[DEBUG] Attaching security group %s to network interface ID %s", sgID, interfaceID)

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

	_, err = conn.ModifyNetworkInterfaceAttribute(params)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Successful attachment of security group %s to network interface ID %s", sgID, interfaceID)

	return resourceAwsNetworkInterfaceSGAttachmentRead(d, meta)
}

func resourceAwsNetworkInterfaceSGAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	sgID := d.Get("security_group_id").(string)
	interfaceID := d.Get("network_interface_id").(string)

	log.Printf("[DEBUG] Checking association of security group %s to network interface ID %s", sgID, interfaceID)

	conn := meta.(*AWSClient).ec2conn

	iface, err := fetchNetworkInterface(conn, interfaceID)

	if isAWSErr(err, "InvalidNetworkInterfaceID.NotFound", "") {
		log.Printf("[WARN] EC2 Network Interface (%s) not found, removing from state", interfaceID)
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	if sgExistsInENI(sgID, iface) {
		d.SetId(fmt.Sprintf("%s_%s", sgID, interfaceID))
	} else {
		// The association does not exist when it should, taint this resource.
		log.Printf("[WARN] Security group %s not associated with network interface ID %s, tainting", sgID, interfaceID)
		d.SetId("")
	}
	return nil
}

func resourceAwsNetworkInterfaceSGAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	mk := "network_interface_sg_attachment_" + d.Get("network_interface_id").(string)
	awsMutexKV.Lock(mk)
	defer awsMutexKV.Unlock(mk)

	sgID := d.Get("security_group_id").(string)
	interfaceID := d.Get("network_interface_id").(string)

	log.Printf("[DEBUG] Removing security group %s from interface ID %s", sgID, interfaceID)

	conn := meta.(*AWSClient).ec2conn

	iface, err := fetchNetworkInterface(conn, interfaceID)

	if isAWSErr(err, "InvalidNetworkInterfaceID.NotFound", "") {
		return nil
	}

	if err != nil {
		return err
	}

	return delSGFromENI(conn, sgID, iface)
}

// fetchNetworkInterface is a utility function used by Read and Delete to fetch
// the full ENI details for a specific interface ID.
func fetchNetworkInterface(conn *ec2.EC2, ifaceID string) (*ec2.NetworkInterface, error) {
	log.Printf("[DEBUG] Fetching information for interface ID %s", ifaceID)
	dniParams := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: aws.StringSlice([]string{ifaceID}),
	}

	dniResp, err := conn.DescribeNetworkInterfaces(dniParams)
	if err != nil {
		return nil, err
	}
	return dniResp.NetworkInterfaces[0], nil
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

	if isAWSErr(err, "InvalidNetworkInterfaceID.NotFound", "") {
		return nil
	}

	return err
}

// sgExistsInENI  is a utility function that can be used to quickly check to
// see if a security group exists in an *ec2.NetworkInterface.
func sgExistsInENI(sgID string, iface *ec2.NetworkInterface) bool {
	for _, v := range iface.Groups {
		if *v.GroupId == sgID {
			return true
		}
	}
	return false
}
