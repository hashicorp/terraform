package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/inspector"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAWSInspectorResourceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsInspectorResourceGroupCreate,
		Read:   resourceAwsInspectorResourceGroupRead,
		Delete: resourceAwsInspectorResourceGroupDelete,

		Schema: map[string]*schema.Schema{
			"tags": &schema.Schema{
				ForceNew: true,
				Type:     schema.TypeMap,
				Required: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsInspectorResourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	resp, err := conn.CreateResourceGroup(&inspector.CreateResourceGroupInput{
		ResourceGroupTags: tagsFromMapInspector(d.Get("tags").(map[string]interface{})),
	})

	if err != nil {
		return err
	}

	d.Set("arn", *resp.ResourceGroupArn)

	d.SetId(*resp.ResourceGroupArn)

	return resourceAwsInspectorResourceGroupRead(d, meta)
}

func resourceAwsInspectorResourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	_, err := conn.DescribeResourceGroups(&inspector.DescribeResourceGroupsInput{
		ResourceGroupArns: []*string{
			aws.String(d.Id()),
		},
	})

	if err != nil {
		if inspectorerr, ok := err.(awserr.Error); ok && inspectorerr.Code() == "InvalidInputException" {
			return nil
		} else {
			log.Printf("[ERROR] Error finding Inspector resource group: %s", err)
			return err
		}
	}

	return nil
}

func resourceAwsInspectorResourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	d.Set("arn", "")
	d.SetId("")

	return nil
}
