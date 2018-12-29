package ecs

import (
	"os"

	"github.com/denverdino/aliyungo/common"
)

// Interval for checking status in WaitForXXX method
const DefaultWaitForInterval = 5

// Default timeout value for WaitForXXX method
const DefaultTimeout = 60

type Client struct {
	common.Client
}

const (
	// ECSDefaultEndpoint is the default API endpoint of ECS services
	ECSDefaultEndpoint = "https://ecs-cn-hangzhou.aliyuncs.com"
	ECSAPIVersion      = "2014-05-26"

	ECSServiceCode = "ecs"

	VPCDefaultEndpoint = "https://vpc.aliyuncs.com"
	VPCAPIVersion      = "2016-04-28"
	VPCServiceCode     = "vpc"
)

// NewClient creates a new instance of ECS client
func NewClient(accessKeyId, accessKeySecret string) *Client {
	endpoint := os.Getenv("ECS_ENDPOINT")
	if endpoint == "" {
		endpoint = ECSDefaultEndpoint
	}
	return NewClientWithEndpoint(endpoint, accessKeyId, accessKeySecret)
}

func NewECSClient(accessKeyId, accessKeySecret string, regionID common.Region) *Client {
	endpoint := os.Getenv("ECS_ENDPOINT")
	if endpoint == "" {
		endpoint = ECSDefaultEndpoint
	}

	return NewClientWithRegion(endpoint, accessKeyId, accessKeySecret, regionID)
}

func NewClientWithRegion(endpoint string, accessKeyId, accessKeySecret string, regionID common.Region) *Client {
	client := &Client{}
	client.NewInit(endpoint, ECSAPIVersion, accessKeyId, accessKeySecret, ECSServiceCode, regionID)
	return client
}

func NewClientWithEndpoint(endpoint string, accessKeyId, accessKeySecret string) *Client {
	client := &Client{}
	client.Init(endpoint, ECSAPIVersion, accessKeyId, accessKeySecret)
	return client
}

func NewVPCClient(accessKeyId, accessKeySecret string, regionID common.Region) *Client {
	endpoint := os.Getenv("VPC_ENDPOINT")
	if endpoint == "" {
		endpoint = VPCDefaultEndpoint
	}

	return NewVPCClientWithRegion(endpoint, accessKeyId, accessKeySecret, regionID)
}

func NewVPCClientWithRegion(endpoint string, accessKeyId, accessKeySecret string, regionID common.Region) *Client {
	client := &Client{}
	client.NewInit(endpoint, VPCAPIVersion, accessKeyId, accessKeySecret, VPCServiceCode, regionID)
	return client
}
