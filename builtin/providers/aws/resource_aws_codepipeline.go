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
			"PipelineDeclaration": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"Name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"RoleArn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"Version": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"ArtifactStore": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"Location": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},

									"Type": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},

									"EncryptionKey": &schema.Schema{
										Type:     schema.TypeList,
										Required: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"Location": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},

												"Type": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},
											}, //EncryptionKey schema
										}, // EncryptionKey schema resource
									}, //EncryptionKey
								}, //schema under Resource
							}, //ArtifactStore - Resource
						}, //ArtifactStore - Schema
					}, //schema PipelineDeclaration resource
				}, //PipelineDeclaration - Resource
			}, //PipelineDeclaration
		}, //Schema
	} //return
} //func

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
