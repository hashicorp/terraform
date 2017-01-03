package alicloud

import (
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
)

type BandwidthPackageType struct {
	IpCount   int
	Bandwidth int
	Zone      string
}

type CreateNatGatewayArgs struct {
	RegionId         common.Region
	VpcId            string
	Spec             string
	BandwidthPackage []BandwidthPackageType
	Name             string
	Description      string
	ClientToken      string
}

type ForwardTableIdType struct {
	ForwardTableId []string
}

type BandwidthPackageIdType struct {
	BandwidthPackageId []string
}

type CreateNatGatewayResponse struct {
	common.Response
	NatGatewayId        string
	ForwardTableIds     ForwardTableIdType
	BandwidthPackageIds BandwidthPackageIdType
}

// CreateNatGateway creates Virtual Private Cloud
//
// You can read doc at http://docs.aliyun.com/#/pub/ecs/open-api/vpc&createvpc
func CreateNatGateway(client *ecs.Client, args *CreateNatGatewayArgs) (resp *CreateNatGatewayResponse, err error) {
	response := CreateNatGatewayResponse{}
	err = client.Invoke("CreateNatGateway", args, &response)
	if err != nil {
		return nil, err
	}
	return &response, err
}

type NatGatewaySetType struct {
	BusinessStatus      string
	Description         string
	BandwidthPackageIds BandwidthPackageIdType
	ForwardTableIds     ForwardTableIdType
	InstanceChargeType  string
	Name                string
	NatGatewayId        string
	RegionId            common.Region
	Spec                string
	Status              string
	VpcId               string
}

type DescribeNatGatewayResponse struct {
	common.Response
	common.PaginationResult
	NatGateways struct {
		NatGateway []NatGatewaySetType
	}
}

type DescribeNatGatewaysArgs struct {
	RegionId     common.Region
	NatGatewayId string
	VpcId        string
	common.Pagination
}

func DescribeNatGateways(client *ecs.Client, args *DescribeNatGatewaysArgs) (natGateways []NatGatewaySetType,
	pagination *common.PaginationResult, err error) {

	args.Validate()
	response := DescribeNatGatewayResponse{}

	err = client.Invoke("DescribeNatGateways", args, &response)

	if err == nil {
		return response.NatGateways.NatGateway, &response.PaginationResult, nil
	}

	return nil, nil, err
}

type ModifyNatGatewayAttributeArgs struct {
	RegionId     common.Region
	NatGatewayId string
	Name         string
	Description  string
}

type ModifyNatGatewayAttributeResponse struct {
	common.Response
}

func ModifyNatGatewayAttribute(client *ecs.Client, args *ModifyNatGatewayAttributeArgs) error {
	response := ModifyNatGatewayAttributeResponse{}
	return client.Invoke("ModifyNatGatewayAttribute", args, &response)
}

type ModifyNatGatewaySpecArgs struct {
	RegionId     common.Region
	NatGatewayId string
	Spec         NatGatewaySpec
}

func ModifyNatGatewaySpec(client *ecs.Client, args *ModifyNatGatewaySpecArgs) error {
	response := ModifyNatGatewayAttributeResponse{}
	return client.Invoke("ModifyNatGatewaySpec", args, &response)
}

type DeleteNatGatewayArgs struct {
	RegionId     common.Region
	NatGatewayId string
}

type DeleteNatGatewayResponse struct {
	common.Response
}

func DeleteNatGateway(client *ecs.Client, args *DeleteNatGatewayArgs) error {
	response := DeleteNatGatewayResponse{}
	err := client.Invoke("DeleteNatGateway", args, &response)
	return err
}

type DescribeBandwidthPackagesArgs struct {
	RegionId           common.Region
	BandwidthPackageId string
	NatGatewayId       string
}

type DescribeBandwidthPackageType struct {
	Bandwidth          string
	BandwidthPackageId string
	IpCount            string
}

type DescribeBandwidthPackagesResponse struct {
	common.Response
	BandwidthPackages struct {
		BandwidthPackage []DescribeBandwidthPackageType
	}
}

func DescribeBandwidthPackages(client *ecs.Client, args *DescribeBandwidthPackagesArgs) ([]DescribeBandwidthPackageType, error) {
	response := &DescribeBandwidthPackagesResponse{}
	err := client.Invoke("DescribeBandwidthPackages", args, response)
	if err != nil {
		return nil, err
	}
	return response.BandwidthPackages.BandwidthPackage, err
}

type DeleteBandwidthPackageArgs struct {
	RegionId           common.Region
	BandwidthPackageId string
}

type DeleteBandwidthPackageResponse struct {
	common.Response
}

func DeleteBandwidthPackage(client *ecs.Client, args *DeleteBandwidthPackageArgs) error {
	response := DeleteBandwidthPackageResponse{}
	err := client.Invoke("DeleteBandwidthPackage", args, &response)
	return err
}

type DescribeSnatTableEntriesArgs struct {
	RegionId common.Region
}

func DescribeSnatTableEntries(client *ecs.Client, args *DescribeSnatTableEntriesArgs) {

}

type NatGatewaySpec string

const (
	NatGatewaySmallSpec  = NatGatewaySpec("Small")
	NatGatewayMiddleSpec = NatGatewaySpec("Middle")
	NatGatewayLargeSpec  = NatGatewaySpec("Large")
)
