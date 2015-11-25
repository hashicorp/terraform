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

						"StageDeclaration": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"Name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},

									"ActionDeclaration": &schema.Schema{
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
													Optional: true,
												},

												"RunOrder": &schema.Schema{
													Type:     schema.TypeInt,
													Optional: true,
												},

												"Configuration": &schema.Schema{
													Type:     schema.TypeMap,
													Required: true,
												},

												"InputArtifact": &schema.Schema{
													Type:     schema.TypeList,
													Required: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"Name": &schema.Schema{
																Type:     schema.TypeString,
																Required: true,
															},
														}, //InputArtifact schema
													}, // InputArtifact schema resource
												}, //InputArtifact

												"OutputArtifact": &schema.Schema{
													Type:     schema.TypeList,
													Required: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"Name": &schema.Schema{
																Type:     schema.TypeString,
																Required: true,
															},
														}, //OutputArtifact schema
													}, // OutputArtifact schema resource
												}, //OutputArtifact

												"ActionTypeId": &schema.Schema{
													Type:     schema.TypeList,
													Required: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"Category": &schema.Schema{
																Type:     schema.TypeString,
																Required: true,
															},

															"Owner": &schema.Schema{
																Type:     schema.TypeString,
																Required: true,
															},

															"Provider": &schema.Schema{
																Type:     schema.TypeString,
																Required: true,
															},

															"Version": &schema.Schema{
																Type:     schema.TypeString,
																Required: true,
															},
														}, //ActionTypeId schema
													}, // ActionTypeId schema resource
												}, //ActionTypeId
											}, //ActionDeclaration schema
										}, // ActionDeclaration schema resource
									}, //ActionDeclaration
								}, //schema under Resource
							}, //StageDeclaration - Resource
						}, //StageDeclaration - Schema
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
