package aws

import (
	"fmt"
	"log"
	"time"
	//"strings"

	"github.com/aws/aws-sdk-go/aws"
	//"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCodeBuildProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeBuildProjectCreate,
		Read:   resourceAwsCodeBuildProjectRead,
		Update: resourceAwsCodeBuildProjectUpdate,
		Delete: resourceAwsCodeBuildProjectDelete,

		Schema: map[string]*schema.Schema{
			"artifacts": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"location": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"namespace_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"packaging": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"encryption_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"environment": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"compute_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"environment_variable": &schema.Schema{
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"value": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							Optional: true,
						},
						"image": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"service_role": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"source": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auth": &schema.Schema{
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resource": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							Optional: true,
						},
						"buildspec": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"location": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Required: true,
			},
			"timeout": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsCodeBuildProjectCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codebuildconn

	log.Printf("[DEBUG] CodeBuild Create Project: %s", d.Id())

	params := &codebuild.CreateProjectInput{
		Environment: expandProjectEnvironment(d.Get("environment").(*schema.Set).List()[0].(map[string]interface{})),
		Name:        aws.String(d.Get("name").(string)),
		Source:      expandProjectSource(d.Get("source").(*schema.Set).List()[0].(map[string]interface{})),
		Artifacts:   expandProjectArtifacts(d.Get("artifacts").(*schema.Set).List()[0].(map[string]interface{})),
	}

	if v, ok := d.GetOk("description"); ok {
		params.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("encryption_key"); ok {
		params.EncryptionKey = aws.String(v.(string))
	}

	if v, ok := d.GetOk("service_role"); ok {
		params.ServiceRole = aws.String(v.(string))
	}

	if v, ok := d.GetOk("timeout"); ok {
		params.TimeoutInMinutes = aws.Int64(int64(v.(int)))
	}

	var resp *codebuild.CreateProjectOutput
	err := resource.Retry(30*time.Second, func() *resource.RetryError {
		var err error

		resp, err = conn.CreateProject(params)

		if err != nil {
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})

	if err != nil {
		return fmt.Errorf("[ERROR] Error creating CodeBuild project: %s", err)
	}

	if resp.Project.Name == nil {
		return fmt.Errorf("[ERROR] Project name was nil")
	}

	d.SetId(*resp.Project.Name)

	return resourceAwsCodeBuildProjectUpdate(d, meta)
}

func expandProjectArtifacts(m map[string]interface{}) *codebuild.ProjectArtifacts {

	projectArtifacts := &codebuild.ProjectArtifacts{
		Type: aws.String(m["type"].(string)),
	}

	if len(m["location"].(string)) > 0 {
		projectArtifacts.Location = aws.String(m["location"].(string))
	}

	if len(m["name"].(string)) > 0 {
		projectArtifacts.Name = aws.String(m["name"].(string))
	}

	if len(m["namespace_type"].(string)) > 0 {
		projectArtifacts.NamespaceType = aws.String(m["namespace_type"].(string))
	}

	if len(m["packaging"].(string)) > 0 {
		projectArtifacts.Packaging = aws.String(m["packaging"].(string))
	}

	if len(m["path"].(string)) > 0 {
		projectArtifacts.Path = aws.String(m["path"].(string))
	}

	return projectArtifacts
}

func expandProjectEnvironment(m map[string]interface{}) *codebuild.ProjectEnvironment {
	projectEnv := &codebuild.ProjectEnvironment{
		ComputeType: aws.String(m["compute_type"].(string)),
		Image:       aws.String(m["image"].(string)),
		Type:        aws.String(m["type"].(string)),
	}

	envVariables := m["environment_variable"].(*schema.Set).List()
	projectEnv.EnvironmentVariables = make([]*codebuild.EnvironmentVariable, len(envVariables))

	for i := 0; i < len(envVariables); i++ {
		v := envVariables[i].(map[string]interface{})
		projectEnv.EnvironmentVariables[i] = &codebuild.EnvironmentVariable{
			Name:  aws.String(v["name"].(string)),
			Value: aws.String(v["value"].(string)),
		}
	}

	return projectEnv
}

func expandProjectSource(m map[string]interface{}) *codebuild.ProjectSource {

	projectSource := &codebuild.ProjectSource{
		Type:      aws.String(m["type"].(string)),
		Location:  aws.String(m["location"].(string)),
		Buildspec: aws.String(m["buildspec"].(string)),
	}

	if v, ok := m["auth"]; ok {
		if len(v.(*schema.Set).List()) > 0 {
			auth := v.(*schema.Set).List()[0].(map[string]interface{})

			projectSource.Auth = &codebuild.SourceAuth{
				Type:     aws.String(auth["type"].(string)),
				Resource: aws.String(auth["resource"].(string)),
			}
		}
	}

	return projectSource
}

func resourceAwsCodeBuildProjectRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codebuildconn

	log.Printf("[DEBUG] CodeBuild Read Project: %s", d.Id())

	resp, err := conn.BatchGetProjects(&codebuild.BatchGetProjectsInput{
		Names: []*string{
			aws.String(d.Get("name").(string)),
		},
	})

	if err != nil {
		return fmt.Errorf("[ERROR] Error retreiving Projects: %q", err)
	}

	// if nothing was found, then return no state
	if len(resp.Projects) == 0 {
		log.Printf("[INFO]: No projects were found, removing from state")
		d.SetId("")
		return nil
	}

	project := resp.Projects[0]

	if err := d.Set("artifacts", artifactsConfigToMap(project.Artifacts)); err != nil {
		return err
	}

	if err := d.Set("environment", environmentConfigToMap(project.Environment)); err != nil {
		return err
	}

	if err := d.Set("source", sourceConfigToMap(project.Source)); err != nil {
		return err
	}

	d.Set("description", project.Description)
	d.Set("encryption_key", project.EncryptionKey)
	d.Set("name", project.Name)
	d.Set("service_role", project.ServiceRole)
	d.Set("timeout", project.TimeoutInMinutes)

	if err := d.Set("tags", tagsToMapCodeBuild(project.Tags)); err != nil {
		return err
	}

	return nil
}

func resourceAwsCodeBuildProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codebuildconn

	log.Printf("[DEBUG] CodeBuild Update Project: %s", d.Id())

	params := &codebuild.UpdateProjectInput{
		Environment: expandProjectEnvironment(d.Get("environment").(*schema.Set).List()[0].(map[string]interface{})),
		Name:        aws.String(d.Get("name").(string)),
		Source:      expandProjectSource(d.Get("source").(*schema.Set).List()[0].(map[string]interface{})),
		Artifacts:   expandProjectArtifacts(d.Get("artifacts").(*schema.Set).List()[0].(map[string]interface{})),
	}

	if v, ok := d.GetOk("description"); ok {
		params.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("encryption_key"); ok {
		params.EncryptionKey = aws.String(v.(string))
	}

	if v, ok := d.GetOk("service_role"); ok {
		params.ServiceRole = aws.String(v.(string))
	}

	if v, ok := d.GetOk("timeout"); ok {
		params.TimeoutInMinutes = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("tags"); ok {
		params.Tags = tagsFromMapCodeBuild(v.(map[string]interface{}))
	}

	_, err := conn.UpdateProject(params)

	if err != nil {
		return fmt.Errorf(
			"[ERROR] Error updating CodeBuild project (%s): %s",
			d.Id(), err)
	}

	return resourceAwsCodeBuildProjectRead(d, meta)
}

func resourceAwsCodeBuildProjectDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codebuildconn

	_, err := conn.DeleteProject(&codebuild.DeleteProjectInput{
		Name: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func artifactsConfigToMap(artifacts *codebuild.ProjectArtifacts) []map[string]string {

	item := make(map[string]string)
	item["type"] = *artifacts.Type

	if artifacts.Location != nil {
		item["location"] = *artifacts.Location
	}

	if artifacts.Name != nil {
		item["name"] = *artifacts.Name
	}

	if artifacts.NamespaceType != nil {
		item["namespace_type"] = *artifacts.NamespaceType
	}

	if artifacts.Packaging != nil {
		item["packaging"] = *artifacts.Packaging
	}

	if artifacts.Path != nil {
		item["path"] = *artifacts.Path
	}

	result := make([]map[string]string, 0, 1)
	result = append(result, item)

	return result
}

func environmentConfigToMap(environment *codebuild.ProjectEnvironment) []map[string]interface{} {

	item := make(map[string]interface{})
	item["type"] = *environment.Type
	item["compute_type"] = *environment.ComputeType
	item["image"] = *environment.Image

	//TODO:
	// environmentVariables := environment.EnvironmentVariables
	// envVariables := make(map[string]string)
	// if len(environmentVariables) > 0 {
	// 	for i := 0; i < len(environmentVariables); i++ {
	// 		env := environmentVariables[i]
	//
	// 		envVariables["name"] = *env.Name
	// 		envVariables["value"] = *env.Value
	// 	}
	// }

	result := make([]map[string]interface{}, 0, 1)
	result = append(result, item)

	return result

}

func sourceConfigToMap(source *codebuild.ProjectSource) []map[string]string {

	item := make(map[string]string)
	item["type"] = *source.Type

	//TODO:
	//if source.Auth != nil {
	//	item["auth"] = *source.Auth
	//}

	if source.Buildspec != nil {
		item["buildspec"] = *source.Buildspec
	}

	if source.Location != nil {
		item["location"] = *source.Location
	}

	result := make([]map[string]string, 0, 1)
	result = append(result, item)

	return result

}
