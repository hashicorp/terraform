package alicloud

import (
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"strings"
)

func (client *AliyunClient) DescribeEipAddress(allocationId string) (*ecs.EipAddressSetType, error) {

	args := ecs.DescribeEipAddressesArgs{
		RegionId:     client.Region,
		AllocationId: allocationId,
	}

	eips, _, err := client.ecsconn.DescribeEipAddresses(&args)
	if err != nil {
		return nil, err
	}
	if len(eips) == 0 {
		return nil, common.GetClientErrorFromString("Not found")
	}

	return &eips[0], nil
}

func (client *AliyunClient) DescribeNatGateway(natGatewayId string) (*ecs.NatGatewaySetType, error) {

	args := &ecs.DescribeNatGatewaysArgs{
		RegionId:     client.Region,
		NatGatewayId: natGatewayId,
	}

	natGateways, _, err := client.vpcconn.DescribeNatGateways(args)
	if err != nil {
		return nil, err
	}

	if len(natGateways) == 0 {
		return nil, common.GetClientErrorFromString("Not found")
	}

	return &natGateways[0], nil
}

func (client *AliyunClient) DescribeVpc(vpcId string) (*ecs.VpcSetType, error) {
	args := ecs.DescribeVpcsArgs{
		RegionId: client.Region,
		VpcId:    vpcId,
	}

	vpcs, _, err := client.ecsconn.DescribeVpcs(&args)
	if err != nil {
		if notFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	if len(vpcs) == 0 {
		return nil, nil
	}

	return &vpcs[0], nil
}

// describe vswitch by param filters
func (client *AliyunClient) QueryVswitches(args *ecs.DescribeVSwitchesArgs) (vswitches []ecs.VSwitchSetType, err error) {
	vsws, _, err := client.ecsconn.DescribeVSwitches(args)
	if err != nil {
		if notFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return vsws, nil
}

func (client *AliyunClient) QueryVswitchById(vpcId, vswitchId string) (vsw *ecs.VSwitchSetType, err error) {
	args := &ecs.DescribeVSwitchesArgs{
		VpcId:     vpcId,
		VSwitchId: vswitchId,
	}
	vsws, err := client.QueryVswitches(args)
	if err != nil {
		return nil, err
	}

	if len(vsws) == 0 {
		return nil, nil
	}

	return &vsws[0], nil
}

func (client *AliyunClient) QueryRouteTables(args *ecs.DescribeRouteTablesArgs) (routeTables []ecs.RouteTableSetType, err error) {
	rts, _, err := client.ecsconn.DescribeRouteTables(args)
	if err != nil {
		return nil, err
	}

	return rts, nil
}

func (client *AliyunClient) QueryRouteTableById(routeTableId string) (rt *ecs.RouteTableSetType, err error) {
	args := &ecs.DescribeRouteTablesArgs{
		RouteTableId: routeTableId,
	}
	rts, err := client.QueryRouteTables(args)
	if err != nil {
		return nil, err
	}

	if len(rts) == 0 {
		return nil, &common.Error{ErrorResponse: common.ErrorResponse{Message: Notfound}}
	}

	return &rts[0], nil
}

func (client *AliyunClient) QueryRouteEntry(routeTableId, cidrBlock, nextHopType, nextHopId string) (rn *ecs.RouteEntrySetType, err error) {
	rt, errs := client.QueryRouteTableById(routeTableId)
	if errs != nil {
		return nil, errs
	}

	for _, e := range rt.RouteEntrys.RouteEntry {
		if strings.ToLower(string(e.DestinationCidrBlock)) == cidrBlock {
			return &e, nil
		}
	}
	return nil, nil
}

func (client *AliyunClient) GetVpcIdByVSwitchId(vswitchId string) (vpcId string, err error) {

	vs, _, err := client.ecsconn.DescribeVpcs(&ecs.DescribeVpcsArgs{
		RegionId: client.Region,
	})
	if err != nil {
		return "", err
	}

	for _, v := range vs {
		for _, sw := range v.VSwitchIds.VSwitchId {
			if sw == vswitchId {
				return v.VpcId, nil
			}
		}
	}

	return "", &common.Error{ErrorResponse: common.ErrorResponse{Message: Notfound}}
}
