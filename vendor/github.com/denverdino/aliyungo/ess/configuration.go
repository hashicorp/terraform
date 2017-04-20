package ess

import (
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
)

type CreateScalingConfigurationArgs struct {
	ScalingGroupId           string
	ImageId                  string
	InstanceType             string
	IoOptimized              ecs.IoOptimized
	SecurityGroupId          string
	ScalingConfigurationName string
	InternetChargeType       common.InternetChargeType
	InternetMaxBandwidthIn   int
	InternetMaxBandwidthOut  int
	SystemDisk_Category      common.UnderlineString
	SystemDisk_Size          common.UnderlineString
	DataDisk                 []DataDiskType
}

type DataDiskType struct {
	Category   string
	SnapshotId string
	Device     string
	Size       int
}

type CreateScalingConfigurationResponse struct {
	ScalingConfigurationId string
	common.Response
}

// CreateScalingConfiguration create scaling configuration
//
// You can read doc at https://help.aliyun.com/document_detail/25944.html?spm=5176.doc25942.6.625.KcE5ir
func (client *Client) CreateScalingConfiguration(args *CreateScalingConfigurationArgs) (resp *CreateScalingConfigurationResponse, err error) {
	response := CreateScalingConfigurationResponse{}
	err = client.InvokeByFlattenMethod("CreateScalingConfiguration", args, &response)

	if err != nil {
		return nil, err
	}
	return &response, nil
}

type DescribeScalingConfigurationsArgs struct {
	RegionId                 common.Region
	ScalingGroupId           string
	ScalingConfigurationId   common.FlattenArray
	ScalingConfigurationName common.FlattenArray
	common.Pagination
}

type DescribeScalingConfigurationsResponse struct {
	common.Response
	common.PaginationResult
	ScalingConfigurations struct {
		ScalingConfiguration []ScalingConfigurationItemType
	}
}

type ScalingConfigurationItemType struct {
	ScalingConfigurationId   string
	ScalingConfigurationName string
	ScalingGroupId           string
	ImageId                  string
	InstanceType             string
	IoOptimized              string
	SecurityGroupId          string
	InternetChargeType       string
	LifecycleState           LifecycleState
	CreationTime             string
	InternetMaxBandwidthIn   int
	InternetMaxBandwidthOut  int
	SystemDiskCategory       string
	DataDisks                struct {
		DataDisk []DataDiskItemType
	}
}

type DataDiskItemType struct {
	Size       int
	Category   string
	SnapshotId string
	Device     string
}

// DescribeScalingConfigurations describes scaling configuration
//
// You can read doc at https://help.aliyun.com/document_detail/25945.html?spm=5176.doc25944.6.626.knG0zz
func (client *Client) DescribeScalingConfigurations(args *DescribeScalingConfigurationsArgs) (configs []ScalingConfigurationItemType, pagination *common.PaginationResult, err error) {
	args.Validate()
	response := DescribeScalingConfigurationsResponse{}

	err = client.InvokeByFlattenMethod("DescribeScalingConfigurations", args, &response)

	if err == nil {
		return response.ScalingConfigurations.ScalingConfiguration, &response.PaginationResult, nil
	}

	return nil, nil, err
}

type DeleteScalingConfigurationArgs struct {
	ScalingConfigurationId string
	ScalingGroupId         string
	ImageId                string
}

type DeleteScalingConfigurationResponse struct {
	common.Response
}

// DeleteScalingConfiguration delete scaling configuration
//
// You can read doc at https://help.aliyun.com/document_detail/25946.html?spm=5176.doc25944.6.627.MjkuuL
func (client *Client) DeleteScalingConfiguration(args *DeleteScalingConfigurationArgs) (resp *DeleteScalingConfigurationResponse, err error) {
	response := DeleteScalingConfigurationResponse{}
	err = client.InvokeByFlattenMethod("DeleteScalingConfiguration", args, &response)

	if err != nil {
		return nil, err
	}
	return &response, nil
}
