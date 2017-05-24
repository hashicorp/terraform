package alicloud

import (
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
)

type Tag struct {
	Key   string
	Value string
}

type AddTagsArgs struct {
	ResourceId   string
	ResourceType ecs.TagResourceType //image, instance, snapshot or disk
	RegionId     common.Region
	Tag          []Tag
}

type RemoveTagsArgs struct {
	ResourceId   string
	ResourceType ecs.TagResourceType //image, instance, snapshot or disk
	RegionId     common.Region
	Tag          []Tag
}

func AddTags(client *ecs.Client, args *AddTagsArgs) error {
	response := ecs.AddTagsResponse{}
	err := client.Invoke("AddTags", args, &response)
	if err != nil {
		return err
	}
	return err
}

func RemoveTags(client *ecs.Client, args *RemoveTagsArgs) error {
	response := ecs.RemoveTagsResponse{}
	err := client.Invoke("RemoveTags", args, &response)
	if err != nil {
		return err
	}
	return err
}
