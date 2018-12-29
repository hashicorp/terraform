package rds

import (
	"github.com/denverdino/aliyungo/common"

	"os"
)

type Client struct {
	common.Client
}

const (
	// ECSDefaultEndpoint is the default API endpoint of RDS services
	RDSDefaultEndpoint = "https://rds.aliyuncs.com"
	RDSAPIVersion      = "2014-08-15"
	RDSServiceCode     = "rds"
)

// NewClient creates a new instance of RDS client
func NewClient(accessKeyId, accessKeySecret string) *Client {
	endpoint := os.Getenv("RDS_ENDPOINT")
	if endpoint == "" {
		endpoint = RDSDefaultEndpoint
	}
	return NewClientWithEndpoint(endpoint, accessKeyId, accessKeySecret)
}

func NewClientWithEndpoint(endpoint string, accessKeyId, accessKeySecret string) *Client {
	client := &Client{}
	client.Init(endpoint, RDSAPIVersion, accessKeyId, accessKeySecret)
	return client
}

func NewRDSClient(accessKeyId, accessKeySecret string, regionID common.Region) *Client {
	endpoint := os.Getenv("RDS_ENDPOINT")
	if endpoint == "" {
		endpoint = RDSDefaultEndpoint
	}

	return NewClientWithRegion(endpoint, accessKeyId, accessKeySecret, regionID)
}

func NewClientWithRegion(endpoint string, accessKeyId, accessKeySecret string, regionID common.Region) *Client {
	client := &Client{}
	client.NewInit(endpoint, RDSAPIVersion, accessKeyId, accessKeySecret, RDSServiceCode, regionID)
	return client
}
