package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datapipeline"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDataPipeline() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDataPipelineCreate,
		Read:   resourceAwsDataPipelineRead,
		Update: resourceAwsDataPipelineUpdate,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": tagsSchema(),
			"uniqueId": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsDataPipelineCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datapipeline

	input := datapipeline.CreatePipelineInput{
		Name:        aws.String(d.Get("name").(string)),
		Description: aws.String(d.Get("description").(string)),
		UniqueId:    aws.String(d.Get("uniqueId").(string)),
	}

	req, err := conn.CreatePipeline(&input)

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Data Pipeline created %s", req)

	d.SetId(*req.PipelineId)

	return resourceAwsDataPipelineRead(d, meta)
}

func resourceAwsDataPipelineRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datapipeline

	input := datapipeline.DescribePipelinesInput{
		PipelineIds: []*string{
			aws.String(d.Id()),
		},
	}

	resp, err := conn.DescribePipelines(&input)

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] CloudTrail received: %s", resp)

	return nil
}

func resourceAwsDataPipelineUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datapipeline

	input := datapipeline.PutPipelineDefinitionInput{
		PipelineId: aws.String(d.Id()),
	}

	resp, err := conn.PutPipelineDefinition(&input)

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] CloudTrail received: %s", resp)

	return resourceAwsDataPipelineRead(d, meta)
}
