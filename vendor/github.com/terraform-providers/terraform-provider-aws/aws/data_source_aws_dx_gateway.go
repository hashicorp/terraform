package aws

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsDxGateway() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsDxGatewayRead,

		Schema: map[string]*schema.Schema{
			"amazon_side_asn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceAwsDxGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn
	name := d.Get("name").(string)

	gateways := make([]*directconnect.Gateway, 0)
	// DescribeDirectConnectGatewaysInput does not have a name parameter for filtering
	input := &directconnect.DescribeDirectConnectGatewaysInput{}
	for {
		output, err := conn.DescribeDirectConnectGateways(input)
		if err != nil {
			return fmt.Errorf("error reading Direct Connect Gateway: %s", err)
		}
		for _, gateway := range output.DirectConnectGateways {
			if aws.StringValue(gateway.DirectConnectGatewayName) == name {
				gateways = append(gateways, gateway)
			}
		}
		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	if len(gateways) == 0 {
		return fmt.Errorf("Direct Connect Gateway not found for name: %s", name)
	}

	if len(gateways) > 1 {
		return fmt.Errorf("Multiple Direct Connect Gateways found for name: %s", name)
	}

	gateway := gateways[0]

	d.SetId(aws.StringValue(gateway.DirectConnectGatewayId))
	d.Set("amazon_side_asn", strconv.FormatInt(aws.Int64Value(gateway.AmazonSideAsn), 10))

	return nil
}
