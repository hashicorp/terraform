package ecs

import "github.com/denverdino/aliyungo/common"

type DescribeInstanceTypesArgs struct {
	InstanceTypeFamily string
}

//
// You can read doc at http://docs.aliyun.com/#/pub/ecs/open-api/datatype&instancetypeitemtype
type InstanceTypeItemType struct {
	InstanceTypeId     string
	CpuCoreCount       int
	MemorySize         float64
	InstanceTypeFamily string
}

type DescribeInstanceTypesResponse struct {
	common.Response
	InstanceTypes struct {
		InstanceType []InstanceTypeItemType
	}
}

// DescribeInstanceTypes describes all instance types
//
// You can read doc at http://docs.aliyun.com/#/pub/ecs/open-api/other&describeinstancetypes
func (client *Client) DescribeInstanceTypes() (instanceTypes []InstanceTypeItemType, err error) {
	response := DescribeInstanceTypesResponse{}

	err = client.Invoke("DescribeInstanceTypes", &DescribeInstanceTypesArgs{}, &response)

	if err != nil {
		return []InstanceTypeItemType{}, err
	}
	return response.InstanceTypes.InstanceType, nil

}

// support user args
func (client *Client) DescribeInstanceTypesNew(args *DescribeInstanceTypesArgs) (instanceTypes []InstanceTypeItemType, err error) {
	response := DescribeInstanceTypesResponse{}

	err = client.Invoke("DescribeInstanceTypes", args, &response)

	if err != nil {
		return []InstanceTypeItemType{}, err
	}
	return response.InstanceTypes.InstanceType, nil

}
