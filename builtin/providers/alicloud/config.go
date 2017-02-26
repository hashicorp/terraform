package alicloud

import (
	"fmt"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/denverdino/aliyungo/slb"
)

// Config of aliyun
type Config struct {
	AccessKey string
	SecretKey string
	Region    common.Region
}

// AliyunClient of aliyun
type AliyunClient struct {
	Region  common.Region
	ecsconn *ecs.Client
	vpcconn *ecs.Client
	slbconn *slb.Client
}

// Client for AliyunClient
func (c *Config) Client() (*AliyunClient, error) {
	err := c.loadAndValidate()
	if err != nil {
		return nil, err
	}

	ecsconn, err := c.ecsConn()
	if err != nil {
		return nil, err
	}

	slbconn, err := c.slbConn()
	if err != nil {
		return nil, err
	}

	vpcconn, err := c.vpcConn()
	if err != nil {
		return nil, err
	}

	return &AliyunClient{
		Region:  c.Region,
		ecsconn: ecsconn,
		vpcconn: vpcconn,
		slbconn: slbconn,
	}, nil
}

func (c *Config) loadAndValidate() error {
	err := c.validateRegion()
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) validateRegion() error {

	for _, valid := range common.ValidRegions {
		if c.Region == valid {
			return nil
		}
	}

	return fmt.Errorf("Not a valid region: %s", c.Region)
}

func (c *Config) ecsConn() (*ecs.Client, error) {
	client := ecs.NewClient(c.AccessKey, c.SecretKey)
	_, err := client.DescribeRegions()

	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Config) slbConn() (*slb.Client, error) {
	client := slb.NewClient(c.AccessKey, c.SecretKey)

	return client, nil
}

func (c *Config) vpcConn() (*ecs.Client, error) {
	_, err := c.ecsConn()

	if err != nil {
		return nil, err
	}

	client := &ecs.Client{}
	client.Init("https://vpc.aliyuncs.com/", "2016-04-28", c.AccessKey, c.SecretKey)
	return client, nil
}
