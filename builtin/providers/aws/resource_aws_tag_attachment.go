package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsTagAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsTagAttachmentCreate,
		Read:   resourceAwsTagAttachmentRead,
		Update: resourceAwsTagAttachmentUpdate,
		Delete: resourceAwsTagAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"resource": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
			},
		},
	}
}

func resourceAwsTagAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	resource := d.Get("resource").(string)
	d.SetId(resource)
	return resourceAwsTagAttachmentUpdate(d, meta)
}

func resourceAwsTagAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	resource := d.Get("resource").(string)

	conn := meta.(*AWSClient).ec2conn
	result, err := conn.DescribeTags(&ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("resource-id"),
				Values: []*string{aws.String(resource)},
			},
		},
	})
	if err != nil {
		return err
	}

	configTags := d.Get("tags").(map[string]interface{})

	if len(result.Tags) > 0 {
		tags := make(map[string]string)
		for _, v := range result.Tags {
			if _, ok := configTags[*v.Key]; ok {
				tags[*v.Key] = *v.Value
			}
		}
		d.Set("tags", tags)
	}
	return nil
}

func resourceAwsTagAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	return setTags(conn, d)
}

func resourceAwsTagAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	tags := d.Get("tags").(map[string]interface{})
	remove := make([]*ec2.Tag, len(tags))
	for k, _ := range tags {
		remove = append(remove, &ec2.Tag{
			Key: aws.String(k),
		})
	}

	_, err := conn.DeleteTags(&ec2.DeleteTagsInput{
		Resources: []*string{aws.String(d.Id())},
		Tags:      remove,
	})
	return err
}
