package alicloud

import (
	"encoding/json"
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"strings"
)

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

func (client *AliyunClient) QueryInstancesByIds(ids []string) (instances []ecs.InstanceAttributesType, err error) {
	idsStr, jerr := json.Marshal(ids)
	if jerr != nil {
		return nil, jerr
	}

	args := ecs.DescribeInstancesArgs{
		RegionId:    client.Region,
		InstanceIds: string(idsStr),
	}

	instances, _, errs := client.ecsconn.DescribeInstances(&args)

	if errs != nil {
		return nil, errs
	}

	return instances, nil
}

func (client *AliyunClient) QueryInstancesById(id string) (instance *ecs.InstanceAttributesType, err error) {
	ids := []string{id}

	instances, errs := client.QueryInstancesByIds(ids)
	if errs != nil {
		return nil, errs
	}

	if len(instances) == 0 {
		return nil, common.GetClientErrorFromString(InstanceNotfound)
	}

	return &instances[0], nil
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

// todo: support syc
func (client *AliyunClient) JoinSecurityGroups(instanceId string, securityGroupIds []string) error {
	for _, sid := range securityGroupIds {
		err := client.ecsconn.JoinSecurityGroup(instanceId, sid)
		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code != InvalidInstanceIdAlreadyExists {
				return err
			}
		}
	}

	return nil
}

func (client *AliyunClient) LeaveSecurityGroups(instanceId string, securityGroupIds []string) error {
	for _, sid := range securityGroupIds {
		err := client.ecsconn.LeaveSecurityGroup(instanceId, sid)
		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code != InvalidSecurityGroupIdNotFound {
				return err
			}
		}
	}

	return nil
}

func (client *AliyunClient) DescribeSecurity(securityGroupId string) (*ecs.DescribeSecurityGroupAttributeResponse, error) {

	args := &ecs.DescribeSecurityGroupAttributeArgs{
		RegionId:        client.Region,
		SecurityGroupId: securityGroupId,
	}

	return client.ecsconn.DescribeSecurityGroupAttribute(args)
}

func (client *AliyunClient) DescribeSecurityGroupRule(securityGroupId, types, ip_protocol, port_range string) (*ecs.PermissionType, error) {

	sg, err := client.DescribeSecurity(securityGroupId)
	if err != nil {
		return nil, err
	}

	for _, p := range sg.Permissions.Permission {
		if strings.ToLower(string(p.IpProtocol)) == ip_protocol && p.PortRange == port_range {
			return &p, nil
		}
	}
	return nil, nil

}

func (client *AliyunClient) RevokeSecurityGroup(args *ecs.RevokeSecurityGroupArgs) error {
	//todo: handle the specal err
	return client.ecsconn.RevokeSecurityGroup(args)
}
