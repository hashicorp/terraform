package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func dxVirtualInterfaceRead(id string, conn *directconnect.DirectConnect) (*directconnect.VirtualInterface, error) {
	resp, state, err := dxVirtualInterfaceStateRefresh(conn, id)()
	if err != nil {
		return nil, fmt.Errorf("Error reading Direct Connect virtual interface: %s", err)
	}
	if state == directconnect.VirtualInterfaceStateDeleted {
		return nil, nil
	}

	return resp.(*directconnect.VirtualInterface), nil
}

func dxVirtualInterfaceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	if err := setTagsDX(conn, d, d.Get("arn").(string)); err != nil {
		return err
	}

	return nil
}

func dxVirtualInterfaceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	log.Printf("[DEBUG] Deleting Direct Connect virtual interface: %s", d.Id())
	_, err := conn.DeleteVirtualInterface(&directconnect.DeleteVirtualInterfaceInput{
		VirtualInterfaceId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, directconnect.ErrCodeClientException, "does not exist") {
			return nil
		}
		return fmt.Errorf("Error deleting Direct Connect virtual interface: %s", err)
	}

	deleteStateConf := &resource.StateChangeConf{
		Pending: []string{
			directconnect.VirtualInterfaceStateAvailable,
			directconnect.VirtualInterfaceStateConfirming,
			directconnect.VirtualInterfaceStateDeleting,
			directconnect.VirtualInterfaceStateDown,
			directconnect.VirtualInterfaceStatePending,
			directconnect.VirtualInterfaceStateRejected,
			directconnect.VirtualInterfaceStateVerifying,
		},
		Target: []string{
			directconnect.VirtualInterfaceStateDeleted,
		},
		Refresh:    dxVirtualInterfaceStateRefresh(conn, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = deleteStateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Direct Connect virtual interface (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func dxVirtualInterfaceStateRefresh(conn *directconnect.DirectConnect, vifId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeVirtualInterfaces(&directconnect.DescribeVirtualInterfacesInput{
			VirtualInterfaceId: aws.String(vifId),
		})
		if err != nil {
			return nil, "", err
		}

		n := len(resp.VirtualInterfaces)
		switch n {
		case 0:
			return "", directconnect.VirtualInterfaceStateDeleted, nil

		case 1:
			vif := resp.VirtualInterfaces[0]
			return vif, aws.StringValue(vif.VirtualInterfaceState), nil

		default:
			return nil, "", fmt.Errorf("Found %d Direct Connect virtual interfaces for %s, expected 1", n, vifId)
		}
	}
}

func dxVirtualInterfaceWaitUntilAvailable(d *schema.ResourceData, conn *directconnect.DirectConnect, pending, target []string) error {
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    dxVirtualInterfaceStateRefresh(conn, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Direct Connect virtual interface (%s) to become available: %s", d.Id(), err)
	}

	return nil
}
