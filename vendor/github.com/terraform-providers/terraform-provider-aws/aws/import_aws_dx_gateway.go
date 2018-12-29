package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/schema"
)

// Direct Connect Gateway import also imports all assocations
func resourceAwsDxGatewayImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	conn := meta.(*AWSClient).dxconn

	id := d.Id()
	resp, err := conn.DescribeDirectConnectGateways(&directconnect.DescribeDirectConnectGatewaysInput{
		DirectConnectGatewayId: aws.String(id),
	})
	if err != nil {
		return nil, err
	}
	if len(resp.DirectConnectGateways) < 1 || resp.DirectConnectGateways[0] == nil {
		return nil, fmt.Errorf("Direct Connect Gateway %s was not found", id)
	}
	results := make([]*schema.ResourceData, 1)
	results[0] = d

	{
		subResource := resourceAwsDxGatewayAssociation()
		resp, err := conn.DescribeDirectConnectGatewayAssociations(&directconnect.DescribeDirectConnectGatewayAssociationsInput{
			DirectConnectGatewayId: aws.String(id),
		})
		if err != nil {
			return nil, err
		}

		for _, assoc := range resp.DirectConnectGatewayAssociations {
			d := subResource.Data(nil)
			d.SetType("aws_dx_gateway_association")
			d.Set("dx_gateway_id", assoc.DirectConnectGatewayId)
			d.Set("vpn_gateway_id", assoc.VirtualGatewayId)
			d.SetId(dxGatewayAssociationId(aws.StringValue(assoc.DirectConnectGatewayId), aws.StringValue(assoc.VirtualGatewayId)))
			results = append(results, d)
		}
	}

	return results, nil
}
