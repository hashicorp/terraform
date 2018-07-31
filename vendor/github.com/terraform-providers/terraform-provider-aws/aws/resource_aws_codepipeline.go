package aws

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codepipeline"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsCodePipeline() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodePipelineCreate,
		Read:   resourceAwsCodePipelineRead,
		Update: resourceAwsCodePipelineUpdate,
		Delete: resourceAwsCodePipelineDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"role_arn": {
				Type:     schema.TypeString,
				Required: true,
			},

			"artifact_store": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"location": {
							Type:     schema.TypeString,
							Required: true,
						},

						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								codepipeline.ArtifactStoreTypeS3,
							}, false),
						},

						"encryption_key": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Required: true,
									},

									"type": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											codepipeline.EncryptionKeyTypeKms,
										}, false),
									},
								},
							},
						},
					},
				},
			},
			"stage": {
				Type:     schema.TypeList,
				MinItems: 2,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"action": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"configuration": {
										Type:     schema.TypeMap,
										Optional: true,
									},
									"category": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											codepipeline.ActionCategorySource,
											codepipeline.ActionCategoryBuild,
											codepipeline.ActionCategoryDeploy,
											codepipeline.ActionCategoryTest,
											codepipeline.ActionCategoryInvoke,
											codepipeline.ActionCategoryApproval,
										}, false),
									},
									"owner": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											codepipeline.ActionOwnerAws,
											codepipeline.ActionOwnerThirdParty,
											codepipeline.ActionOwnerCustom,
										}, false),
									},
									"provider": {
										Type:     schema.TypeString,
										Required: true,
									},
									"version": {
										Type:     schema.TypeString,
										Required: true,
									},
									"input_artifacts": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"output_artifacts": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"role_arn": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"run_order": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func validateAwsCodePipelineStageActionConfiguration(v interface{}, k string) (ws []string, errors []error) {
	for k := range v.(map[string]interface{}) {
		if k == "OAuthToken" {
			errors = append(errors, fmt.Errorf("CodePipeline: OAuthToken should be set as environment variable 'GITHUB_TOKEN'"))
		}
	}
	return
}

func resourceAwsCodePipelineCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codepipelineconn
	params := &codepipeline.CreatePipelineInput{
		Pipeline: expandAwsCodePipeline(d),
	}

	var resp *codepipeline.CreatePipelineOutput
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var err error

		resp, err = conn.CreatePipeline(params)

		if err != nil {
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
	if err != nil {
		return fmt.Errorf("[ERROR] Error creating CodePipeline: %s", err)
	}
	if resp.Pipeline == nil {
		return fmt.Errorf("[ERROR] Error creating CodePipeline: invalid response from AWS")
	}

	d.SetId(*resp.Pipeline.Name)
	return resourceAwsCodePipelineRead(d, meta)
}

func expandAwsCodePipeline(d *schema.ResourceData) *codepipeline.PipelineDeclaration {
	pipelineArtifactStore := expandAwsCodePipelineArtifactStore(d)
	pipelineStages := expandAwsCodePipelineStages(d)

	pipeline := codepipeline.PipelineDeclaration{
		Name:          aws.String(d.Get("name").(string)),
		RoleArn:       aws.String(d.Get("role_arn").(string)),
		ArtifactStore: pipelineArtifactStore,
		Stages:        pipelineStages,
	}
	return &pipeline
}
func expandAwsCodePipelineArtifactStore(d *schema.ResourceData) *codepipeline.ArtifactStore {
	configs := d.Get("artifact_store").([]interface{})
	data := configs[0].(map[string]interface{})
	pipelineArtifactStore := codepipeline.ArtifactStore{
		Location: aws.String(data["location"].(string)),
		Type:     aws.String(data["type"].(string)),
	}
	tek := data["encryption_key"].([]interface{})
	if len(tek) > 0 {
		vk := tek[0].(map[string]interface{})
		ek := codepipeline.EncryptionKey{
			Type: aws.String(vk["type"].(string)),
			Id:   aws.String(vk["id"].(string)),
		}
		pipelineArtifactStore.EncryptionKey = &ek
	}
	return &pipelineArtifactStore
}

func flattenAwsCodePipelineArtifactStore(artifactStore *codepipeline.ArtifactStore) []interface{} {
	values := map[string]interface{}{}
	values["type"] = *artifactStore.Type
	values["location"] = *artifactStore.Location
	if artifactStore.EncryptionKey != nil {
		as := map[string]interface{}{
			"id":   *artifactStore.EncryptionKey.Id,
			"type": *artifactStore.EncryptionKey.Type,
		}
		values["encryption_key"] = []interface{}{as}
	}
	return []interface{}{values}
}

func expandAwsCodePipelineStages(d *schema.ResourceData) []*codepipeline.StageDeclaration {
	configs := d.Get("stage").([]interface{})
	pipelineStages := []*codepipeline.StageDeclaration{}

	for _, stage := range configs {
		data := stage.(map[string]interface{})
		a := data["action"].([]interface{})
		actions := expandAwsCodePipelineActions(a)
		pipelineStages = append(pipelineStages, &codepipeline.StageDeclaration{
			Name:    aws.String(data["name"].(string)),
			Actions: actions,
		})
	}
	return pipelineStages
}

func flattenAwsCodePipelineStages(stages []*codepipeline.StageDeclaration) []interface{} {
	stagesList := []interface{}{}
	for _, stage := range stages {
		values := map[string]interface{}{}
		values["name"] = *stage.Name
		values["action"] = flattenAwsCodePipelineStageActions(stage.Actions)
		stagesList = append(stagesList, values)
	}
	return stagesList

}

func expandAwsCodePipelineActions(s []interface{}) []*codepipeline.ActionDeclaration {
	actions := []*codepipeline.ActionDeclaration{}
	for _, config := range s {
		data := config.(map[string]interface{})

		conf := expandAwsCodePipelineStageActionConfiguration(data["configuration"].(map[string]interface{}))
		if data["provider"].(string) == "GitHub" {
			githubToken := os.Getenv("GITHUB_TOKEN")
			if githubToken != "" {
				conf["OAuthToken"] = aws.String(githubToken)
			}

		}

		action := codepipeline.ActionDeclaration{
			ActionTypeId: &codepipeline.ActionTypeId{
				Category: aws.String(data["category"].(string)),
				Owner:    aws.String(data["owner"].(string)),

				Provider: aws.String(data["provider"].(string)),
				Version:  aws.String(data["version"].(string)),
			},
			Name:          aws.String(data["name"].(string)),
			Configuration: conf,
		}

		oa := data["output_artifacts"].([]interface{})
		if len(oa) > 0 {
			outputArtifacts := expandAwsCodePipelineActionsOutputArtifacts(oa)
			action.OutputArtifacts = outputArtifacts

		}
		ia := data["input_artifacts"].([]interface{})
		if len(ia) > 0 {
			inputArtifacts := expandAwsCodePipelineActionsInputArtifacts(ia)
			action.InputArtifacts = inputArtifacts

		}
		ra := data["role_arn"].(string)
		if ra != "" {
			action.RoleArn = aws.String(ra)
		}
		ro := data["run_order"].(int)
		if ro > 0 {
			action.RunOrder = aws.Int64(int64(ro))
		}
		actions = append(actions, &action)
	}
	return actions
}

func flattenAwsCodePipelineStageActions(actions []*codepipeline.ActionDeclaration) []interface{} {
	actionsList := []interface{}{}
	for _, action := range actions {
		values := map[string]interface{}{
			"category": *action.ActionTypeId.Category,
			"owner":    *action.ActionTypeId.Owner,
			"provider": *action.ActionTypeId.Provider,
			"version":  *action.ActionTypeId.Version,
			"name":     *action.Name,
		}
		if action.Configuration != nil {
			config := flattenAwsCodePipelineStageActionConfiguration(action.Configuration)
			_, ok := config["OAuthToken"]
			actionProvider := *action.ActionTypeId.Provider
			if ok && actionProvider == "GitHub" {
				delete(config, "OAuthToken")
			}
			values["configuration"] = config
		}

		if len(action.OutputArtifacts) > 0 {
			values["output_artifacts"] = flattenAwsCodePipelineActionsOutputArtifacts(action.OutputArtifacts)
		}

		if len(action.InputArtifacts) > 0 {
			values["input_artifacts"] = flattenAwsCodePipelineActionsInputArtifacts(action.InputArtifacts)
		}

		if action.RoleArn != nil {
			values["role_arn"] = *action.RoleArn
		}

		if action.RunOrder != nil {
			values["run_order"] = int(*action.RunOrder)
		}

		actionsList = append(actionsList, values)
	}
	return actionsList
}

func expandAwsCodePipelineStageActionConfiguration(config map[string]interface{}) map[string]*string {
	m := map[string]*string{}
	for k, v := range config {
		s := v.(string)
		m[k] = &s
	}
	return m
}

func flattenAwsCodePipelineStageActionConfiguration(config map[string]*string) map[string]string {
	m := map[string]string{}
	for k, v := range config {
		m[k] = *v
	}
	return m
}

func expandAwsCodePipelineActionsOutputArtifacts(s []interface{}) []*codepipeline.OutputArtifact {
	outputArtifacts := []*codepipeline.OutputArtifact{}
	for _, artifact := range s {
		if artifact == nil {
			continue
		}
		outputArtifacts = append(outputArtifacts, &codepipeline.OutputArtifact{
			Name: aws.String(artifact.(string)),
		})
	}
	return outputArtifacts
}

func flattenAwsCodePipelineActionsOutputArtifacts(artifacts []*codepipeline.OutputArtifact) []string {
	values := []string{}
	for _, artifact := range artifacts {
		values = append(values, *artifact.Name)
	}
	return values
}

func expandAwsCodePipelineActionsInputArtifacts(s []interface{}) []*codepipeline.InputArtifact {
	outputArtifacts := []*codepipeline.InputArtifact{}
	for _, artifact := range s {
		if artifact == nil {
			continue
		}
		outputArtifacts = append(outputArtifacts, &codepipeline.InputArtifact{
			Name: aws.String(artifact.(string)),
		})
	}
	return outputArtifacts
}

func flattenAwsCodePipelineActionsInputArtifacts(artifacts []*codepipeline.InputArtifact) []string {
	values := []string{}
	for _, artifact := range artifacts {
		values = append(values, *artifact.Name)
	}
	return values
}

func resourceAwsCodePipelineRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codepipelineconn
	resp, err := conn.GetPipeline(&codepipeline.GetPipelineInput{
		Name: aws.String(d.Id()),
	})

	if err != nil {
		pipelineerr, ok := err.(awserr.Error)
		if ok && pipelineerr.Code() == "PipelineNotFoundException" {
			log.Printf("[INFO] Codepipeline %q not found", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("[ERROR] Error retreiving Pipeline: %q", err)
	}
	metadata := resp.Metadata
	pipeline := resp.Pipeline

	if err := d.Set("artifact_store", flattenAwsCodePipelineArtifactStore(pipeline.ArtifactStore)); err != nil {
		return err
	}

	if err := d.Set("stage", flattenAwsCodePipelineStages(pipeline.Stages)); err != nil {
		return err
	}

	d.Set("arn", metadata.PipelineArn)
	d.Set("name", pipeline.Name)
	d.Set("role_arn", pipeline.RoleArn)
	return nil
}

func resourceAwsCodePipelineUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codepipelineconn

	pipeline := expandAwsCodePipeline(d)
	params := &codepipeline.UpdatePipelineInput{
		Pipeline: pipeline,
	}
	_, err := conn.UpdatePipeline(params)

	if err != nil {
		return fmt.Errorf(
			"[ERROR] Error updating CodePipeline (%s): %s",
			d.Id(), err)
	}

	return resourceAwsCodePipelineRead(d, meta)
}

func resourceAwsCodePipelineDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codepipelineconn

	_, err := conn.DeletePipeline(&codepipeline.DeletePipelineInput{
		Name: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	return nil
}
