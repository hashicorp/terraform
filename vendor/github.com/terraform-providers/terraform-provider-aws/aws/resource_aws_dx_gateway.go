package aws

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDxGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxGatewayCreate,
		Read:   resourceAwsDxGatewayRead,
		Delete: resourceAwsDxGatewayDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsDxGatewayImportState,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"amazon_side_asn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAmazonSideAsn,
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceAwsDxGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	req := &directconnect.CreateDirectConnectGatewayInput{
		DirectConnectGatewayName: aws.String(d.Get("name").(string)),
	}
	if asn, ok := d.GetOk("amazon_side_asn"); ok {
		i, err := strconv.ParseInt(asn.(string), 10, 64)
		if err != nil {
			return err
		}
		req.AmazonSideAsn = aws.Int64(i)
	}

	log.Printf("[DEBUG] Creating Direct Connect gateway: %#v", req)
	resp, err := conn.CreateDirectConnectGateway(req)
	if err != nil {
		return fmt.Errorf("Error creating Direct Connect gateway: %s", err)
	}

	d.SetId(aws.StringValue(resp.DirectConnectGateway.DirectConnectGatewayId))

	stateConf := &resource.StateChangeConf{
		Pending:    []string{directconnect.GatewayStatePending},
		Target:     []string{directconnect.GatewayStateAvailable},
		Refresh:    dxGatewayStateRefresh(conn, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Direct Connect gateway (%s) to become available: %s", d.Id(), err)
	}

	return resourceAwsDxGatewayRead(d, meta)
}

func resourceAwsDxGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	dxGwRaw, state, err := dxGatewayStateRefresh(conn, d.Id())()
	if err != nil {
		return fmt.Errorf("Error reading Direct Connect gateway: %s", err)
	}
	if state == directconnect.GatewayStateDeleted {
		log.Printf("[WARN] Direct Connect gateway (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	dxGw := dxGwRaw.(*directconnect.Gateway)
	d.Set("name", aws.StringValue(dxGw.DirectConnectGatewayName))
	d.Set("amazon_side_asn", strconv.FormatInt(aws.Int64Value(dxGw.AmazonSideAsn), 10))

	return nil
}

func resourceAwsDxGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	_, err := conn.DeleteDirectConnectGateway(&directconnect.DeleteDirectConnectGatewayInput{
		DirectConnectGatewayId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, "DirectConnectClientException", "does not exist") {
			return nil
		}
		return fmt.Errorf("Error deleting Direct Connect gateway: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{directconnect.GatewayStatePending, directconnect.GatewayStateAvailable, directconnect.GatewayStateDeleting},
		Target:     []string{directconnect.GatewayStateDeleted},
		Refresh:    dxGatewayStateRefresh(conn, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Direct Connect gateway (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func dxGatewayStateRefresh(conn *directconnect.DirectConnect, dxgwId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeDirectConnectGateways(&directconnect.DescribeDirectConnectGatewaysInput{
			DirectConnectGatewayId: aws.String(dxgwId),
		})
		if err != nil {
			return nil, "", err
		}

		n := len(resp.DirectConnectGateways)
		switch n {
		case 0:
			return "", directconnect.GatewayStateDeleted, nil

		case 1:
			dxgw := resp.DirectConnectGateways[0]
			return dxgw, aws.StringValue(dxgw.DirectConnectGatewayState), nil

		default:
			return nil, "", fmt.Errorf("Found %d Direct Connect gateways for %s, expected 1", n, dxgwId)
		}
	}
}
