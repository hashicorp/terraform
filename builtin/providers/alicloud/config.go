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

func (client *AliyunClient) DescribeNatGateway(natGatewayId string) (*NatGatewaySetType, error) {

	args := &DescribeNetGatewaysArgs{
		RegionId:     client.Region,
		NatGatewayId: natGatewayId,
	}

	natGateways, _, err := DescribeNatGateways(client.ecsconn, args)
	if err != nil {
		return nil, err
	}

	if len(natGateways) == 0 {
		return nil, common.GetClientErrorFromString("Not found")
	}

	return &natGateways[0], nil
}

func (client *AliyunClient) DescribeImage(imageId string) (*ecs.ImageType, error) {

	pagination := common.Pagination{
		PageNumber: 1,
	}
	args := ecs.DescribeImagesArgs{
		Pagination: pagination,
		RegionId:   client.Region,
		Status:     ecs.ImageStatusAvailable,
	}

	var allImages []ecs.ImageType

	for {
		images, _, err := client.ecsconn.DescribeImages(&args)
		if err != nil {
			break
		}

		if len(images) == 0 {
			break
		}

		allImages = append(allImages, images...)

		args.Pagination.PageNumber++
	}

	if len(allImages) == 0 {
		return nil, common.GetClientErrorFromString("Not found")
	}

	var image *ecs.ImageType
	imageIds := []string{}
	for _, im := range allImages {
		if im.ImageId == imageId {
			image = &im
		}
		imageIds = append(imageIds, im.ImageId)
	}

	if image == nil {
		return nil, fmt.Errorf("image_id %s not exists in range %s, all images are %s", imageId, client.Region, imageIds)
	}

	return image, nil
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

// DescribeZone validate zoneId is valid in region
func (client *AliyunClient) DescribeZone(zoneID string) (*ecs.ZoneType, error) {
	zones, err := client.ecsconn.DescribeZones(client.Region)
	if err != nil {
		return nil, fmt.Errorf("error to list zones not found")
	}

	var zone *ecs.ZoneType
	zoneIds := []string{}
	for _, z := range zones {
		if z.ZoneId == zoneID {
			zone = &ecs.ZoneType{
				ZoneId:                    z.ZoneId,
				LocalName:                 z.LocalName,
				AvailableResourceCreation: z.AvailableResourceCreation,
				AvailableDiskCategories:   z.AvailableDiskCategories,
			}
		}
		zoneIds = append(zoneIds, z.ZoneId)
	}

	if zone == nil {
		return nil, fmt.Errorf("availability_zone not exists in range %s, all zones are %s", client.Region, zoneIds)
	}

	return zone, nil
}

// ResourceAvailable check resource available for zone
func (client *AliyunClient) ResourceAvailable(zone *ecs.ZoneType, resourceType ecs.ResourceType) error {
	available := false
	for _, res := range zone.AvailableResourceCreation.ResourceTypes {
		if res == resourceType {
			available = true
		}
	}
	if !available {
		return fmt.Errorf("%s is not available in %s zone of %s region", resourceType, zone.ZoneId, client.Region)
	}

	return nil
}

func (client *AliyunClient) DiskAvailable(zone *ecs.ZoneType, diskCategory ecs.DiskCategory) error {
	available := false
	for _, dist := range zone.AvailableDiskCategories.DiskCategories {
		if dist == diskCategory {
			available = true
		}
	}
	if !available {
		return fmt.Errorf("%s is not available in %s zone of %s region", diskCategory, zone.ZoneId, client.Region)
	}
	return nil
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
