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
