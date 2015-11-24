package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCodePipeline() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodePipelineCreate,
		Read:   resourceAwsCodePipelineRead,
		Update: resourceAwsCodePipelineUpdate,
		Delete: resourceAwsCodePipelineDelete,

		Schema: map[string]*schema.Schema{
			"PipelineDeclaration": pipelineDeclarationSchema(),
		},
	}
}

// tagsSchema returns the schema to use for tags.
func pipelineDeclarationSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"key": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"value": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"propagate_at_launch": &schema.Schema{
					Type:     schema.TypeBool,
					Required: true,
				},
			},
		},
		Set: autoscalingTagsToHash,
	}
}

func resourceAwsCodePipelineCreate(d *schema.ResourceData, meta interface{}) error {
	//conn := meta.(*AWSClient).codepipelineconn
	return fmt.Errorf("CodePipelineCreate Not implemented")

	//return resourceAwsCodePipelineRead(d, meta)
}

func resourceAwsCodePipelineRead(d *schema.ResourceData, meta interface{}) error {
	//conn := meta.(*AWSClient).codepipelineconn

	return fmt.Errorf("resourceAwsCodePipelineRead Not implemented")
	//	return nil
}

func resourceAwsCodePipelineUpdate(d *schema.ResourceData, meta interface{}) error {
	//conn := meta.(*AWSClient).codepipelineconn
	return fmt.Errorf("resourceAwsCodePipelineUpdate Not implemented")

	//	return nil
}

func resourceAwsCodePipelineDelete(d *schema.ResourceData, meta interface{}) error {
	//conn := meta.(*AWSClient).codepipelineconn

	return fmt.Errorf("resourceAwsCodePipelineDelete Not implemented")
	//return nil
}
