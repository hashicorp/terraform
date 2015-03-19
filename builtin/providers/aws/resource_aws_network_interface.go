package aws

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNetworkInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNetworkInterfaceCreate,
		Read:   resourceAwsNetworkInterfaceRead,
		Update: resourceAwsNetworkInterfaceUpdate,
		Delete: resourceAwsNetworkInterfaceDelete,

		Schema: map[string]*schema.Schema{

			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"private_ips": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"attachment": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"device_index": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"attachment_id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: resourceAwsEniAttachmentHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsNetworkInterfaceCreate(d *schema.ResourceData, meta interface{}) error {

	ec2conn := meta.(*AWSClient).ec2conn

	request := &ec2.CreateNetworkInterfaceRequest{
		Groups:             expandStringList(d.Get("security_groups").(*schema.Set).List()),
		SubnetID:           aws.String(d.Get("subnet_id").(string)),
		PrivateIPAddresses: expandPrivateIPAddesses(d.Get("private_ips").(*schema.Set).List()),
	}

	log.Printf("[DEBUG] Creating network interface")
	resp, err := ec2conn.CreateNetworkInterface(request)
	if err != nil {
		return fmt.Errorf("Error creating ENI: %s", err)
	}

	d.SetId(*resp.NetworkInterface.NetworkInterfaceID)
	log.Printf("[INFO] ENI ID: %s", d.Id())

	return resourceAwsNetworkInterfaceUpdate(d, meta)
}

func resourceAwsNetworkInterfaceRead(d *schema.ResourceData, meta interface{}) error {

	ec2conn := meta.(*AWSClient).ec2conn
	describe_network_interfaces_request := &ec2.DescribeNetworkInterfacesRequest{
		NetworkInterfaceIDs: []string{d.Id()},
	}
	describeResp, err := ec2conn.DescribeNetworkInterfaces(describe_network_interfaces_request)

	if err != nil {
		if ec2err, ok := err.(aws.APIError); ok && ec2err.Code == "InvalidNetworkInterfaceID.NotFound" {
			// The ENI is gone now, so just remove it from the state
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving ENI: %s", err)
	}
	if len(describeResp.NetworkInterfaces) != 1 {
		return fmt.Errorf("Unable to find ENI: %#v", describeResp.NetworkInterfaces)
	}

	eni := describeResp.NetworkInterfaces[0]
	d.Set("subnet_id", eni.SubnetID)
	d.Set("private_ips", flattenNetworkInterfacesPrivateIPAddesses(eni.PrivateIPAddresses))
	d.Set("security_groups", flattenGroupIdentifiers(eni.Groups))

	// Tags
	d.Set("tags", tagsToMap(eni.TagSet))

	if eni.Attachment != nil {
		attachment := []map[string]interface{}{flattenAttachment(eni.Attachment)}
		d.Set("attachment", attachment)
	} else {
		d.Set("attachment", nil)
	}

	return nil
}

func networkInterfaceAttachmentRefreshFunc(ec2conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		describe_network_interfaces_request := &ec2.DescribeNetworkInterfacesRequest{
			NetworkInterfaceIDs: []string{id},
		}
		describeResp, err := ec2conn.DescribeNetworkInterfaces(describe_network_interfaces_request)

		if err != nil {
			log.Printf("[ERROR] Could not find network interface %s. %s", id, err)
			return nil, "", err
		}

		eni := describeResp.NetworkInterfaces[0]
		hasAttachment := strconv.FormatBool(eni.Attachment != nil)
		log.Printf("[DEBUG] ENI %s has attachment state %s", id, hasAttachment)
		return eni, hasAttachment, nil
	}
}

func resourceAwsNetworkInterfaceDetach(oa *schema.Set, meta interface{}, eniId string) error {
	// if there was an old attachment, remove it
	if oa != nil && len(oa.List()) > 0 {
		old_attachment := oa.List()[0].(map[string]interface{})
		detach_request := &ec2.DetachNetworkInterfaceRequest{
			AttachmentID: aws.String(old_attachment["attachment_id"].(string)),
			Force:        aws.Boolean(true),
		}
		ec2conn := meta.(*AWSClient).ec2conn
		detach_err := ec2conn.DetachNetworkInterface(detach_request)
		if detach_err != nil {
			return fmt.Errorf("Error detaching ENI: %s", detach_err)
		}

		log.Printf("[DEBUG] Waiting for ENI (%s) to become dettached", eniId)
		stateConf := &resource.StateChangeConf{
			Pending: []string{"true"},
			Target:  "false",
			Refresh: networkInterfaceAttachmentRefreshFunc(ec2conn, eniId),
			Timeout: 10 * time.Minute,
		}
		if _, err := stateConf.WaitForState(); err != nil {
			return fmt.Errorf(
				"Error waiting for ENI (%s) to become dettached: %s", eniId, err)
		}
	}

	return nil
}

func resourceAwsNetworkInterfaceUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn
	d.Partial(true)

	if d.HasChange("attachment") {
		ec2conn := meta.(*AWSClient).ec2conn
		oa, na := d.GetChange("attachment")

		detach_err := resourceAwsNetworkInterfaceDetach(oa.(*schema.Set), meta, d.Id())
		if detach_err != nil {
			return detach_err
		}

		// if there is a new attachment, attach it
		if na != nil && len(na.(*schema.Set).List()) > 0 {
			new_attachment := na.(*schema.Set).List()[0].(map[string]interface{})
			attach_request := &ec2.AttachNetworkInterfaceRequest{
				DeviceIndex:        aws.Integer(new_attachment["device_index"].(int)),
				InstanceID:         aws.String(new_attachment["instance"].(string)),
				NetworkInterfaceID: aws.String(d.Id()),
			}
			_, attach_err := ec2conn.AttachNetworkInterface(attach_request)
			if attach_err != nil {
				return fmt.Errorf("Error attaching ENI: %s", attach_err)
			}
		}

		d.SetPartial("attachment")
	}

	if d.HasChange("security_groups") {
		request := &ec2.ModifyNetworkInterfaceAttributeRequest{
			NetworkInterfaceID: aws.String(d.Id()),
			Groups:             expandStringList(d.Get("security_groups").(*schema.Set).List()),
		}

		err := ec2conn.ModifyNetworkInterfaceAttribute(request)
		if err != nil {
			return fmt.Errorf("Failure updating ENI: %s", err)
		}

		d.SetPartial("security_groups")
	}

	if err := setTags(ec2conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsNetworkInterfaceRead(d, meta)
}

func resourceAwsNetworkInterfaceDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Deleting ENI: %s", d.Id())

	detach_err := resourceAwsNetworkInterfaceDetach(d.Get("attachment").(*schema.Set), meta, d.Id())
	if detach_err != nil {
		return detach_err
	}

	deleteEniOpts := ec2.DeleteNetworkInterfaceRequest{
		NetworkInterfaceID: aws.String(d.Id()),
	}
	if err := ec2conn.DeleteNetworkInterface(&deleteEniOpts); err != nil {
		return fmt.Errorf("Error deleting ENI: %s", err)
	}

	return nil
}

func resourceAwsEniAttachmentHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["instance"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["device_index"].(int)))
	return hashcode.String(buf.String())
}
