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

const (
	GatewayAssociationStateDeleted = "deleted"
)

func resourceAwsDxGatewayAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxGatewayAssociationCreate,
		Read:   resourceAwsDxGatewayAssociationRead,
		Delete: resourceAwsDxGatewayAssociationDelete,

		Schema: map[string]*schema.Schema{
			"dx_gateway_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vpn_gateway_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(15 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceAwsDxGatewayAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	dxgwId := d.Get("dx_gateway_id").(string)
	vgwId := d.Get("vpn_gateway_id").(string)
	req := &directconnect.CreateDirectConnectGatewayAssociationInput{
		DirectConnectGatewayId: aws.String(dxgwId),
		VirtualGatewayId:       aws.String(vgwId),
	}

	log.Printf("[DEBUG] Creating Direct Connect gateway association: %#v", req)
	_, err := conn.CreateDirectConnectGatewayAssociation(req)
	if err != nil {
		return fmt.Errorf("Error creating Direct Connect gateway association: %s", err)
	}

	d.SetId(dxGatewayAssociationId(dxgwId, vgwId))

	stateConf := &resource.StateChangeConf{
		Pending:    []string{directconnect.GatewayAssociationStateAssociating},
		Target:     []string{directconnect.GatewayAssociationStateAssociated},
		Refresh:    dxGatewayAssociationStateRefresh(conn, dxgwId, vgwId),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Direct Connect gateway association (%s) to become available: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsDxGatewayAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	dxgwId := d.Get("dx_gateway_id").(string)
	vgwId := d.Get("vpn_gateway_id").(string)
	_, state, err := dxGatewayAssociationStateRefresh(conn, dxgwId, vgwId)()
	if err != nil {
		return fmt.Errorf("Error reading Direct Connect gateway association: %s", err)
	}
	if state == GatewayAssociationStateDeleted {
		log.Printf("[WARN] Direct Connect gateway association (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	return nil
}

func resourceAwsDxGatewayAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	dxgwId := d.Get("dx_gateway_id").(string)
	vgwId := d.Get("vpn_gateway_id").(string)

	log.Printf("[DEBUG] Deleting Direct Connect gateway association: %s", d.Id())

	_, err := conn.DeleteDirectConnectGatewayAssociation(&directconnect.DeleteDirectConnectGatewayAssociationInput{
		DirectConnectGatewayId: aws.String(dxgwId),
		VirtualGatewayId:       aws.String(vgwId),
	})
	if err != nil {
		if isAWSErr(err, "DirectConnectClientException", "No association exists") {
			return nil
		}
		return fmt.Errorf("Error deleting Direct Connect gateway association: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{directconnect.GatewayAssociationStateDisassociating},
		Target:     []string{directconnect.GatewayAssociationStateDisassociated, GatewayAssociationStateDeleted},
		Refresh:    dxGatewayAssociationStateRefresh(conn, dxgwId, vgwId),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Direct Connect gateway association (%s) to be deleted: %s", d.Id(), err.Error())
	}

	return nil
}

func dxGatewayAssociationStateRefresh(conn *directconnect.DirectConnect, dxgwId, vgwId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeDirectConnectGatewayAssociations(&directconnect.DescribeDirectConnectGatewayAssociationsInput{
			DirectConnectGatewayId: aws.String(dxgwId),
			VirtualGatewayId:       aws.String(vgwId),
		})
		if err != nil {
			return nil, "", err
		}

		n := len(resp.DirectConnectGatewayAssociations)
		switch n {
		case 0:
			return "", GatewayAssociationStateDeleted, nil

		case 1:
			assoc := resp.DirectConnectGatewayAssociations[0]
			return assoc, aws.StringValue(assoc.AssociationState), nil

		default:
			return nil, "", fmt.Errorf("Found %d Direct Connect gateway associations for %s, expected 1", n, dxGatewayAssociationId(dxgwId, vgwId))
		}
	}
}

func dxGatewayAssociationId(dxgwId, vgwId string) string {
	return fmt.Sprintf("ga-%s%s", dxgwId, vgwId)
}
