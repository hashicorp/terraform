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
		Delete: resourceAwsDataPipelineDelete,
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
			"unique_id": &schema.Schema{
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
		UniqueId:    aws.String(d.Get("unique_id").(string)),
	}

	req, err := conn.CreatePipeline(&input)

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Data Pipeline created %s", req)

	d.SetId(*req.PipelineId)

	return resourceAwsDataPipelineUpdate(d, meta)
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

	log.Printf("[DEBUG] DataPipeline received: %s", resp)

	return nil
}

func resourceAwsDataPipelineUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datapipeline

	input := datapipeline.PutPipelineDefinitionInput{
		PipelineId: aws.String(d.Id()),
		PipelineObjects: []*datapipeline.PipelineObject{
			{
				Fields: []*datapipeline.Field{
					{
						Key:         aws.String("startDateTime"),
						StringValue: aws.String("2012-09-25T17:00:00"),
					},
					{
						Key:         aws.String("type"),
						StringValue: aws.String("Schedule"),
					},
					{
						Key:         aws.String("period"),
						StringValue: aws.String("1 hour"),
					},
					{
						Key:         aws.String("endDateTime"),
						StringValue: aws.String("2012-09-25T18:00:00"),
					},
				},
				Id:   aws.String("Schedule"),
				Name: aws.String("Schedule"),
			},
		},
	}

	if d.HasChange("tags") {
		err := setTagsDatapipeline(conn, d)
		if err != nil {
			return err
		}
	}

	resp, err := conn.PutPipelineDefinition(&input)

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] DataPipeline received: %s", resp)

	return resourceAwsDataPipelineRead(d, meta)
}

func resourceAwsDataPipelineDelete(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).datapipeline

	log.Printf("[DEBUG] Deleting DataPipeline: %q", d.Id())

	input := datapipeline.DeletePipelineInput{
		PipelineId: aws.String(d.Id()),
	}

	_, err := conn.DeletePipeline(&input)

	if err != nil {
		return err
	}

	return nil
}
