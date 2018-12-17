package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
)

func decodeEc2TransitGatewayRouteID(id string) (string, string, error) {
	parts := strings.Split(id, "_")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected format of ID (%q), expected tgw-rtb-ID_DESTINATION", id)
	}

	return parts[0], parts[1], nil
}

func decodeEc2TransitGatewayRouteTableAssociationID(id string) (string, string, error) {
	parts := strings.Split(id, "_")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected format of ID (%q), expected tgw-rtb-ID_tgw-attach-ID", id)
	}

	return parts[0], parts[1], nil
}

func decodeEc2TransitGatewayRouteTablePropagationID(id string) (string, string, error) {
	parts := strings.Split(id, "_")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected format of ID (%q), expected tgw-rtb-ID_tgw-attach-ID", id)
	}

	return parts[0], parts[1], nil
}

func ec2DescribeTransitGateway(conn *ec2.EC2, transitGatewayID string) (*ec2.TransitGateway, error) {
	input := &ec2.DescribeTransitGatewaysInput{
		TransitGatewayIds: []*string{aws.String(transitGatewayID)},
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateway (%s): %s", transitGatewayID, input)
	for {
		output, err := conn.DescribeTransitGateways(input)

		if err != nil {
			return nil, err
		}

		if output == nil || len(output.TransitGateways) == 0 {
			return nil, nil
		}

		for _, transitGateway := range output.TransitGateways {
			if transitGateway == nil {
				continue
			}

			if aws.StringValue(transitGateway.TransitGatewayId) == transitGatewayID {
				return transitGateway, nil
			}
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil, nil
}

func ec2DescribeTransitGatewayRoute(conn *ec2.EC2, transitGatewayRouteTableID, destination string) (*ec2.TransitGatewayRoute, error) {
	input := &ec2.SearchTransitGatewayRoutesInput{
		// As of the time of writing, the EC2 API reference documentation (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SearchTransitGatewayRoutes.html)
		// incorrectly states which filter Names are allowed. The below are example errors:
		// InvalidParameterValue: Value (transit-gateway-route-destination-cidr-block) for parameter Filters is invalid.
		// InvalidParameterValue: Value (transit-gateway-route-type) for parameter Filters is invalid.
		// InvalidParameterValue: Value (destination-cidr-block) for parameter Filters is invalid.
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("type"),
				Values: []*string{aws.String("static")},
			},
		},
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	log.Printf("[DEBUG] Searching EC2 Transit Gateway Route Table (%s): %s", transitGatewayRouteTableID, input)
	output, err := conn.SearchTransitGatewayRoutes(input)

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.Routes) == 0 {
		return nil, nil
	}

	for _, route := range output.Routes {
		if route == nil {
			continue
		}

		if aws.StringValue(route.DestinationCidrBlock) == destination {
			return route, nil
		}
	}

	return nil, nil
}

func ec2DescribeTransitGatewayRouteTable(conn *ec2.EC2, transitGatewayRouteTableID string) (*ec2.TransitGatewayRouteTable, error) {
	input := &ec2.DescribeTransitGatewayRouteTablesInput{
		TransitGatewayRouteTableIds: []*string{aws.String(transitGatewayRouteTableID)},
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateway Route Table (%s): %s", transitGatewayRouteTableID, input)
	for {
		output, err := conn.DescribeTransitGatewayRouteTables(input)

		if err != nil {
			return nil, err
		}

		if output == nil || len(output.TransitGatewayRouteTables) == 0 {
			return nil, nil
		}

		for _, transitGatewayRouteTable := range output.TransitGatewayRouteTables {
			if transitGatewayRouteTable == nil {
				continue
			}

			if aws.StringValue(transitGatewayRouteTable.TransitGatewayRouteTableId) == transitGatewayRouteTableID {
				return transitGatewayRouteTable, nil
			}
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil, nil
}

func ec2DescribeTransitGatewayRouteTableAssociation(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) (*ec2.TransitGatewayRouteTableAssociation, error) {
	if transitGatewayRouteTableID == "" {
		return nil, nil
	}

	input := &ec2.GetTransitGatewayRouteTableAssociationsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("transit-gateway-attachment-id"),
				Values: []*string{aws.String(transitGatewayAttachmentID)},
			},
		},
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	output, err := conn.GetTransitGatewayRouteTableAssociations(input)

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.Associations) == 0 {
		return nil, nil
	}

	return output.Associations[0], nil
}

func ec2DescribeTransitGatewayRouteTablePropagation(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) (*ec2.TransitGatewayRouteTablePropagation, error) {
	if transitGatewayRouteTableID == "" {
		return nil, nil
	}

	input := &ec2.GetTransitGatewayRouteTablePropagationsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("transit-gateway-attachment-id"),
				Values: []*string{aws.String(transitGatewayAttachmentID)},
			},
		},
		TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
	}

	output, err := conn.GetTransitGatewayRouteTablePropagations(input)

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.TransitGatewayRouteTablePropagations) == 0 {
		return nil, nil
	}

	return output.TransitGatewayRouteTablePropagations[0], nil
}

func ec2DescribeTransitGatewayVpcAttachment(conn *ec2.EC2, transitGatewayAttachmentID string) (*ec2.TransitGatewayVpcAttachment, error) {
	input := &ec2.DescribeTransitGatewayVpcAttachmentsInput{
		TransitGatewayAttachmentIds: []*string{aws.String(transitGatewayAttachmentID)},
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateway VPC Attachment (%s): %s", transitGatewayAttachmentID, input)
	for {
		output, err := conn.DescribeTransitGatewayVpcAttachments(input)

		if err != nil {
			return nil, err
		}

		if output == nil || len(output.TransitGatewayVpcAttachments) == 0 {
			return nil, nil
		}

		for _, transitGatewayVpcAttachment := range output.TransitGatewayVpcAttachments {
			if transitGatewayVpcAttachment == nil {
				continue
			}

			if aws.StringValue(transitGatewayVpcAttachment.TransitGatewayAttachmentId) == transitGatewayAttachmentID {
				return transitGatewayVpcAttachment, nil
			}
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil, nil
}

func ec2TransitGatewayRouteTableAssociationUpdate(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string, associate bool) error {
	transitGatewayAssociation, err := ec2DescribeTransitGatewayRouteTableAssociation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID)
	if err != nil {
		return fmt.Errorf("error determining EC2 Transit Gateway Attachment Route Table (%s) association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
	}

	if associate && transitGatewayAssociation == nil {
		input := &ec2.AssociateTransitGatewayRouteTableInput{
			TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
			TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
		}

		if _, err := conn.AssociateTransitGatewayRouteTable(input); err != nil {
			return fmt.Errorf("error associating EC2 Transit Gateway Route Table (%s) association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}

		if err := waitForEc2TransitGatewayRouteTableAssociationCreation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID); err != nil {
			return fmt.Errorf("error waiting for EC2 Transit Gateway Route Table (%s) association (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}
	} else if !associate && transitGatewayAssociation != nil {
		input := &ec2.DisassociateTransitGatewayRouteTableInput{
			TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
			TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
		}

		if _, err := conn.DisassociateTransitGatewayRouteTable(input); err != nil {
			return fmt.Errorf("error disassociating EC2 Transit Gateway Route Table (%s) disassociation (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}

		if err := waitForEc2TransitGatewayRouteTableAssociationDeletion(conn, transitGatewayRouteTableID, transitGatewayAttachmentID); err != nil {
			return fmt.Errorf("error waiting for EC2 Transit Gateway Route Table (%s) disassociation (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}
	}

	return nil
}

func ec2TransitGatewayRouteTablePropagationUpdate(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string, enablePropagation bool) error {
	transitGatewayRouteTablePropagation, err := ec2DescribeTransitGatewayRouteTablePropagation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID)
	if err != nil {
		return fmt.Errorf("error determining EC2 Transit Gateway Attachment (%s) propagation to Route Table (%s): %s", transitGatewayAttachmentID, transitGatewayRouteTableID, err)
	}

	if enablePropagation && transitGatewayRouteTablePropagation == nil {
		input := &ec2.EnableTransitGatewayRouteTablePropagationInput{
			TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
			TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
		}

		if _, err := conn.EnableTransitGatewayRouteTablePropagation(input); err != nil {
			return fmt.Errorf("error enabling EC2 Transit Gateway Attachment (%s) propagation to Route Table (%s): %s", transitGatewayAttachmentID, transitGatewayRouteTableID, err)
		}
	} else if !enablePropagation && transitGatewayRouteTablePropagation != nil {
		input := &ec2.DisableTransitGatewayRouteTablePropagationInput{
			TransitGatewayAttachmentId: aws.String(transitGatewayAttachmentID),
			TransitGatewayRouteTableId: aws.String(transitGatewayRouteTableID),
		}

		if _, err := conn.DisableTransitGatewayRouteTablePropagation(input); err != nil {
			return fmt.Errorf("error disabling EC2 Transit Gateway Attachment (%s) propagation to Route Table (%s): %s", transitGatewayAttachmentID, transitGatewayRouteTableID, err)
		}
	}

	return nil
}

func ec2TransitGatewayRefreshFunc(conn *ec2.EC2, transitGatewayID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGateway, err := ec2DescribeTransitGateway(conn, transitGatewayID)

		if isAWSErr(err, "InvalidTransitGatewayID.NotFound", "") {
			return nil, ec2.TransitGatewayStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway (%s): %s", transitGatewayID, err)
		}

		if transitGateway == nil {
			return nil, ec2.TransitGatewayStateDeleted, nil
		}

		return transitGateway, aws.StringValue(transitGateway.State), nil
	}
}

func ec2TransitGatewayRouteTableRefreshFunc(conn *ec2.EC2, transitGatewayRouteTableID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGatewayRouteTable, err := ec2DescribeTransitGatewayRouteTable(conn, transitGatewayRouteTableID)

		if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
			return nil, ec2.TransitGatewayRouteTableStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway Route Table (%s): %s", transitGatewayRouteTableID, err)
		}

		if transitGatewayRouteTable == nil {
			return nil, ec2.TransitGatewayRouteTableStateDeleted, nil
		}

		return transitGatewayRouteTable, aws.StringValue(transitGatewayRouteTable.State), nil
	}
}

func ec2TransitGatewayRouteTableAssociationRefreshFunc(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGatewayAssociation, err := ec2DescribeTransitGatewayRouteTableAssociation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID)

		if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
			return nil, ec2.TransitGatewayRouteTableStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway Route Table (%s) Association for (%s): %s", transitGatewayRouteTableID, transitGatewayAttachmentID, err)
		}

		if transitGatewayAssociation == nil {
			return nil, ec2.TransitGatewayRouteTableStateDeleted, nil
		}

		return transitGatewayAssociation, aws.StringValue(transitGatewayAssociation.State), nil
	}
}

func ec2TransitGatewayVpcAttachmentRefreshFunc(conn *ec2.EC2, transitGatewayAttachmentID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGatewayVpcAttachment, err := ec2DescribeTransitGatewayVpcAttachment(conn, transitGatewayAttachmentID)

		if isAWSErr(err, "InvalidTransitGatewayAttachmentID.NotFound", "") {
			return nil, ec2.TransitGatewayAttachmentStateDeleted, nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading EC2 Transit Gateway VPC Attachment (%s): %s", transitGatewayAttachmentID, err)
		}

		if transitGatewayVpcAttachment == nil {
			return nil, ec2.TransitGatewayAttachmentStateDeleted, nil
		}

		return transitGatewayVpcAttachment, aws.StringValue(transitGatewayVpcAttachment.State), nil
	}
}

func expandEc2TransitGatewayTagSpecifications(m map[string]interface{}) []*ec2.TagSpecification {
	if len(m) == 0 {
		return nil
	}

	return []*ec2.TagSpecification{
		{
			ResourceType: aws.String("transit-gateway"),
			Tags:         tagsFromMap(m),
		},
	}
}

func expandEc2TransitGatewayAttachmentTagSpecifications(m map[string]interface{}) []*ec2.TagSpecification {
	if len(m) == 0 {
		return nil
	}

	return []*ec2.TagSpecification{
		{
			ResourceType: aws.String("transit-gateway-attachment"),
			Tags:         tagsFromMap(m),
		},
	}
}

func expandEc2TransitGatewayRouteTableTagSpecifications(m map[string]interface{}) []*ec2.TagSpecification {
	if len(m) == 0 {
		return nil
	}

	return []*ec2.TagSpecification{
		{
			ResourceType: aws.String("transit-gateway-route-table"),
			Tags:         tagsFromMap(m),
		},
	}
}

func waitForEc2TransitGatewayCreation(conn *ec2.EC2, transitGatewayID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayStatePending},
		Target:  []string{ec2.TransitGatewayStateAvailable},
		Refresh: ec2TransitGatewayRefreshFunc(conn, transitGatewayID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway (%s) availability", transitGatewayID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayDeletion(conn *ec2.EC2, transitGatewayID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayStateAvailable,
			ec2.TransitGatewayStateDeleting,
		},
		Target:         []string{ec2.TransitGatewayStateDeleted},
		Refresh:        ec2TransitGatewayRefreshFunc(conn, transitGatewayID),
		Timeout:        10 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway (%s) deletion", transitGatewayID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayRouteTableCreation(conn *ec2.EC2, transitGatewayRouteTableID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayRouteTableStatePending},
		Target:  []string{ec2.TransitGatewayRouteTableStateAvailable},
		Refresh: ec2TransitGatewayRouteTableRefreshFunc(conn, transitGatewayRouteTableID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Route Table (%s) availability", transitGatewayRouteTableID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayRouteTableDeletion(conn *ec2.EC2, transitGatewayRouteTableID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayRouteTableStateAvailable,
			ec2.TransitGatewayRouteTableStateDeleting,
		},
		Target:         []string{ec2.TransitGatewayRouteTableStateDeleted},
		Refresh:        ec2TransitGatewayRouteTableRefreshFunc(conn, transitGatewayRouteTableID),
		Timeout:        10 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Route Table (%s) deletion", transitGatewayRouteTableID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayRouteTableAssociationCreation(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayAssociationStateAssociating},
		Target:  []string{ec2.TransitGatewayAssociationStateAssociated},
		Refresh: ec2TransitGatewayRouteTableAssociationRefreshFunc(conn, transitGatewayRouteTableID, transitGatewayAttachmentID),
		Timeout: 5 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Route Table (%s) association: %s", transitGatewayRouteTableID, transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayRouteTableAssociationDeletion(conn *ec2.EC2, transitGatewayRouteTableID, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayAssociationStateAssociated,
			ec2.TransitGatewayAssociationStateDisassociating,
		},
		Target:         []string{""},
		Refresh:        ec2TransitGatewayRouteTableAssociationRefreshFunc(conn, transitGatewayRouteTableID, transitGatewayAttachmentID),
		Timeout:        5 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway Route Table (%s) disassociation: %s", transitGatewayRouteTableID, transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayRouteTableAttachmentCreation(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayAttachmentStatePending},
		Target:  []string{ec2.TransitGatewayAttachmentStateAvailable},
		Refresh: ec2TransitGatewayVpcAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway VPC Attachment (%s) availability", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2TransitGatewayRouteTableAttachmentDeletion(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.TransitGatewayAttachmentStateAvailable,
			ec2.TransitGatewayAttachmentStateDeleting,
		},
		Target:         []string{ec2.TransitGatewayAttachmentStateDeleted},
		Refresh:        ec2TransitGatewayVpcAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout:        10 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway VPC Attachment (%s) deletion", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}

func waitForEc2TransitGatewayRouteTableAttachmentUpdate(conn *ec2.EC2, transitGatewayAttachmentID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{ec2.TransitGatewayAttachmentStateModifying},
		Target:  []string{ec2.TransitGatewayAttachmentStateAvailable},
		Refresh: ec2TransitGatewayVpcAttachmentRefreshFunc(conn, transitGatewayAttachmentID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for EC2 Transit Gateway VPC Attachment (%s) availability", transitGatewayAttachmentID)
	_, err := stateConf.WaitForState()

	return err
}
