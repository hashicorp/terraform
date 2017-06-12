package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/terraform/helper/schema"
)

func dmsTagsToMap(tags []*dms.Tag) map[string]string {
	result := make(map[string]string)

	for _, tag := range tags {
		result[*tag.Key] = *tag.Value
	}

	return result
}

func dmsTagsFromMap(m map[string]interface{}) []*dms.Tag {
	result := make([]*dms.Tag, 0, len(m))

	for k, v := range m {
		result = append(result, &dms.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func dmsDiffTags(oldTags, newTags []*dms.Tag) ([]*dms.Tag, []*dms.Tag) {
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	remove := []*dms.Tag{}
	for _, t := range oldTags {
		v, ok := create[*t.Key]
		if !ok || v != *t.Value {
			remove = append(remove, t)
		}
	}

	return dmsTagsFromMap(create), remove
}

func dmsGetTagKeys(tags []*dms.Tag) []*string {
	keys := []*string{}

	for _, tag := range tags {
		keys = append(keys, tag.Key)
	}

	return keys
}

func dmsSetTags(arn string, d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})

		add, remove := dmsDiffTags(dmsTagsFromMap(o), dmsTagsFromMap(n))

		if len(remove) > 0 {
			_, err := conn.RemoveTagsFromResource(&dms.RemoveTagsFromResourceInput{
				ResourceArn: aws.String(arn),
				TagKeys:     dmsGetTagKeys(remove),
			})
			if err != nil {
				return err
			}
		}

		if len(add) > 0 {
			_, err := conn.AddTagsToResource(&dms.AddTagsToResourceInput{
				ResourceArn: aws.String(arn),
				Tags:        add,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
