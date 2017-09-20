package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/hashicorp/terraform/helper/hashcode"
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
			"artifacts": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
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
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateAwsCodeBuildArifactsNamespaceType,
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
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateAwsCodeBuildArifactsType,
						},
					},
				},
				Set: resourceAwsCodeBuildProjectArtifactsHash,
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateAwsCodeBuildProjectDescription,
			},
			"encryption_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"environment": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"compute_type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateAwsCodeBuildEnvironmentComputeType,
						},
						"environment_variable": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
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
						},
						"image": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateAwsCodeBuildEnvironmentType,
						},
						"privileged_mode": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
				Set: resourceAwsCodeBuildProjectEnvironmentHash,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsCodeBuildProjectName,
			},
			"service_role": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"source": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auth": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resource": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateAwsCodeBuildSourceAuthType,
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
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateAwsCodeBuildSourceType,
						},
					},
				},
				Required: true,
				MaxItems: 1,
			},
			"timeout": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validateAwsCodeBuildTimeout,
				Removed:      "This field has been removed. Please use build_timeout instead",
			},
			"build_timeout": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      "60",
				ValidateFunc: validateAwsCodeBuildTimeout,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsCodeBuildProjectCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codebuildconn

	projectEnv := expandProjectEnvironment(d)
	projectSource := expandProjectSource(d)
	projectArtifacts := expandProjectArtifacts(d)

	params := &codebuild.CreateProjectInput{
		Environment: projectEnv,
		Name:        aws.String(d.Get("name").(string)),
		Source:      &projectSource,
		Artifacts:   &projectArtifacts,
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

	if v, ok := d.GetOk("build_timeout"); ok {
		params.TimeoutInMinutes = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("tags"); ok {
		params.Tags = tagsFromMapCodeBuild(v.(map[string]interface{}))
	}

	var resp *codebuild.CreateProjectOutput
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		var err error

		resp, err = conn.CreateProject(params)
		if err != nil {
			// Work around eventual consistency of IAM
			if isAWSErr(err, "InvalidInputException", "CodeBuild is not authorized to perform") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil

	})

	if err != nil {
		return fmt.Errorf("[ERROR] Error creating CodeBuild project: %s", err)
	}

	d.SetId(*resp.Project.Arn)

	return resourceAwsCodeBuildProjectUpdate(d, meta)
}

func expandProjectArtifacts(d *schema.ResourceData) codebuild.ProjectArtifacts {
	configs := d.Get("artifacts").(*schema.Set).List()
	data := configs[0].(map[string]interface{})

	projectArtifacts := codebuild.ProjectArtifacts{
		Type: aws.String(data["type"].(string)),
	}

	if data["location"].(string) != "" {
		projectArtifacts.Location = aws.String(data["location"].(string))
	}

	if data["name"].(string) != "" {
		projectArtifacts.Name = aws.String(data["name"].(string))
	}

	if data["namespace_type"].(string) != "" {
		projectArtifacts.NamespaceType = aws.String(data["namespace_type"].(string))
	}

	if data["packaging"].(string) != "" {
		projectArtifacts.Packaging = aws.String(data["packaging"].(string))
	}

	if data["path"].(string) != "" {
		projectArtifacts.Path = aws.String(data["path"].(string))
	}

	return projectArtifacts
}

func expandProjectEnvironment(d *schema.ResourceData) *codebuild.ProjectEnvironment {
	configs := d.Get("environment").(*schema.Set).List()

	envConfig := configs[0].(map[string]interface{})

	projectEnv := &codebuild.ProjectEnvironment{
		PrivilegedMode: aws.Bool(envConfig["privileged_mode"].(bool)),
	}

	if v := envConfig["compute_type"]; v != nil {
		projectEnv.ComputeType = aws.String(v.(string))
	}

	if v := envConfig["image"]; v != nil {
		projectEnv.Image = aws.String(v.(string))
	}

	if v := envConfig["type"]; v != nil {
		projectEnv.Type = aws.String(v.(string))
	}

	if v := envConfig["environment_variable"]; v != nil {
		envVariables := v.([]interface{})
		if len(envVariables) > 0 {
			projectEnvironmentVariables := make([]*codebuild.EnvironmentVariable, 0, len(envVariables))

			for _, envVariablesConfig := range envVariables {
				config := envVariablesConfig.(map[string]interface{})

				projectEnvironmentVar := &codebuild.EnvironmentVariable{}

				if v := config["name"].(string); v != "" {
					projectEnvironmentVar.Name = &v
				}

				if v := config["value"].(string); v != "" {
					projectEnvironmentVar.Value = &v
				}

				projectEnvironmentVariables = append(projectEnvironmentVariables, projectEnvironmentVar)
			}

			projectEnv.EnvironmentVariables = projectEnvironmentVariables
		}
	}

	return projectEnv
}

func expandProjectSource(d *schema.ResourceData) codebuild.ProjectSource {
	configs := d.Get("source").(*schema.Set).List()
	projectSource := codebuild.ProjectSource{}

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		sourceType := data["type"].(string)
		location := data["location"].(string)
		buildspec := data["buildspec"].(string)

		projectSource = codebuild.ProjectSource{
			Type:      &sourceType,
			Location:  &location,
			Buildspec: &buildspec,
		}

		if v, ok := data["auth"]; ok {
			if len(v.(*schema.Set).List()) > 0 {
				auth := v.(*schema.Set).List()[0].(map[string]interface{})

				projectSource.Auth = &codebuild.SourceAuth{
					Type:     aws.String(auth["type"].(string)),
					Resource: aws.String(auth["resource"].(string)),
				}
			}
		}
	}

	return projectSource
}

func resourceAwsCodeBuildProjectRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codebuildconn

	resp, err := conn.BatchGetProjects(&codebuild.BatchGetProjectsInput{
		Names: []*string{
			aws.String(d.Id()),
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

	if err := d.Set("artifacts", flattenAwsCodebuildProjectArtifacts(project.Artifacts)); err != nil {
		return err
	}

	if err := d.Set("environment", schema.NewSet(resourceAwsCodeBuildProjectEnvironmentHash, flattenAwsCodebuildProjectEnvironment(project.Environment))); err != nil {
		return err
	}

	if err := d.Set("source", flattenAwsCodebuildProjectSource(project.Source)); err != nil {
		return err
	}

	d.Set("description", project.Description)
	d.Set("encryption_key", project.EncryptionKey)
	d.Set("name", project.Name)
	d.Set("service_role", project.ServiceRole)
	d.Set("build_timeout", project.TimeoutInMinutes)

	if err := d.Set("tags", tagsToMapCodeBuild(project.Tags)); err != nil {
		return err
	}

	return nil
}

func resourceAwsCodeBuildProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codebuildconn

	params := &codebuild.UpdateProjectInput{
		Name: aws.String(d.Get("name").(string)),
	}

	if d.HasChange("environment") {
		projectEnv := expandProjectEnvironment(d)
		params.Environment = projectEnv
	}

	if d.HasChange("source") {
		projectSource := expandProjectSource(d)
		params.Source = &projectSource
	}

	if d.HasChange("artifacts") {
		projectArtifacts := expandProjectArtifacts(d)
		params.Artifacts = &projectArtifacts
	}

	if d.HasChange("description") {
		params.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("encryption_key") {
		params.EncryptionKey = aws.String(d.Get("encryption_key").(string))
	}

	if d.HasChange("service_role") {
		params.ServiceRole = aws.String(d.Get("service_role").(string))
	}

	if d.HasChange("build_timeout") {
		params.TimeoutInMinutes = aws.Int64(int64(d.Get("build_timeout").(int)))
	}

	// The documentation clearly says "The replacement set of tags for this build project."
	// But its a slice of pointers so if not set for every update, they get removed.
	params.Tags = tagsFromMapCodeBuild(d.Get("tags").(map[string]interface{}))

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

func flattenAwsCodebuildProjectArtifacts(artifacts *codebuild.ProjectArtifacts) *schema.Set {

	artifactSet := schema.Set{
		F: resourceAwsCodeBuildProjectArtifactsHash,
	}

	values := map[string]interface{}{}

	values["type"] = *artifacts.Type

	if artifacts.Location != nil {
		values["location"] = *artifacts.Location
	}

	if artifacts.Name != nil {
		values["name"] = *artifacts.Name
	}

	if artifacts.NamespaceType != nil {
		values["namespace_type"] = *artifacts.NamespaceType
	}

	if artifacts.Packaging != nil {
		values["packaging"] = *artifacts.Packaging
	}

	if artifacts.Path != nil {
		values["path"] = *artifacts.Path
	}

	artifactSet.Add(values)

	return &artifactSet
}

func flattenAwsCodebuildProjectEnvironment(environment *codebuild.ProjectEnvironment) []interface{} {
	envConfig := map[string]interface{}{}

	envConfig["type"] = *environment.Type
	envConfig["compute_type"] = *environment.ComputeType
	envConfig["image"] = *environment.Image
	envConfig["privileged_mode"] = *environment.PrivilegedMode

	if environment.EnvironmentVariables != nil {
		envConfig["environment_variable"] = environmentVariablesToMap(environment.EnvironmentVariables)
	}

	return []interface{}{envConfig}

}

func flattenAwsCodebuildProjectSource(source *codebuild.ProjectSource) *schema.Set {

	sourceSet := schema.Set{
		F: resourceAwsCodeBuildProjectSourceHash,
	}

	authSet := schema.Set{
		F: resourceAwsCodeBuildProjectSourceAuthHash,
	}

	sourceConfig := map[string]interface{}{}

	sourceConfig["type"] = *source.Type

	if source.Auth != nil {
		authSet.Add(sourceAuthToMap(source.Auth))
		sourceConfig["auth"] = &authSet
	}

	if source.Buildspec != nil {
		sourceConfig["buildspec"] = *source.Buildspec
	}

	if source.Location != nil {
		sourceConfig["location"] = *source.Location
	}

	sourceSet.Add(sourceConfig)

	return &sourceSet

}

func resourceAwsCodeBuildProjectArtifactsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	artifactType := m["type"].(string)

	buf.WriteString(fmt.Sprintf("%s-", artifactType))

	return hashcode.String(buf.String())
}

func resourceAwsCodeBuildProjectEnvironmentHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	environmentType := m["type"].(string)
	computeType := m["compute_type"].(string)
	image := m["image"].(string)
	privilegedMode := m["privileged_mode"].(bool)
	environmentVariables := m["environment_variable"].([]interface{})
	buf.WriteString(fmt.Sprintf("%s-", environmentType))
	buf.WriteString(fmt.Sprintf("%s-", computeType))
	buf.WriteString(fmt.Sprintf("%s-", image))
	buf.WriteString(fmt.Sprintf("%t-", privilegedMode))
	for _, e := range environmentVariables {
		if e != nil { // Old statefiles might have nil values in them
			ev := e.(map[string]interface{})
			buf.WriteString(fmt.Sprintf("%s:%s-", ev["name"].(string), ev["value"].(string)))
		}
	}

	return hashcode.String(buf.String())
}

func resourceAwsCodeBuildProjectSourceHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	sourceType := m["type"].(string)
	buildspec := m["buildspec"].(string)
	location := m["location"].(string)

	buf.WriteString(fmt.Sprintf("%s-", sourceType))
	buf.WriteString(fmt.Sprintf("%s-", buildspec))
	buf.WriteString(fmt.Sprintf("%s-", location))

	return hashcode.String(buf.String())
}

func resourceAwsCodeBuildProjectSourceAuthHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))

	if m["resource"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["resource"].(string)))
	}

	return hashcode.String(buf.String())
}

func environmentVariablesToMap(environmentVariables []*codebuild.EnvironmentVariable) []interface{} {

	envVariables := []interface{}{}
	if len(environmentVariables) > 0 {
		for _, env := range environmentVariables {
			item := map[string]interface{}{}
			item["name"] = *env.Name
			item["value"] = *env.Value
			envVariables = append(envVariables, item)
		}
	}

	return envVariables
}

func sourceAuthToMap(sourceAuth *codebuild.SourceAuth) map[string]interface{} {

	auth := map[string]interface{}{}
	auth["type"] = *sourceAuth.Type

	if sourceAuth.Resource != nil {
		auth["resource"] = *sourceAuth.Resource
	}

	return auth
}

func validateAwsCodeBuildArifactsType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	types := map[string]bool{
		"CODEPIPELINE": true,
		"NO_ARTIFACTS": true,
		"S3":           true,
	}

	if !types[value] {
		errors = append(errors, fmt.Errorf("CodeBuild: Arifacts Type can only be CODEPIPELINE / NO_ARTIFACTS / S3"))
	}
	return
}

func validateAwsCodeBuildArifactsNamespaceType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	types := map[string]bool{
		"NONE":     true,
		"BUILD_ID": true,
	}

	if !types[value] {
		errors = append(errors, fmt.Errorf("CodeBuild: Arifacts Namespace Type can only be NONE / BUILD_ID"))
	}
	return
}

func validateAwsCodeBuildProjectName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[A-Za-z0-9]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter or number", value))
	}

	if !regexp.MustCompile(`^[A-Za-z0-9\-_]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters, hyphens and underscores allowed in %q", value))
	}

	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than 255 characters", value))
	}

	return
}

func validateAwsCodeBuildProjectDescription(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf("%q cannot be greater than 255 characters", value))
	}
	return
}

func validateAwsCodeBuildEnvironmentComputeType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	types := map[string]bool{
		"BUILD_GENERAL1_SMALL":  true,
		"BUILD_GENERAL1_MEDIUM": true,
		"BUILD_GENERAL1_LARGE":  true,
	}

	if !types[value] {
		errors = append(errors, fmt.Errorf("CodeBuild: Environment Compute Type can only be BUILD_GENERAL1_SMALL / BUILD_GENERAL1_MEDIUM / BUILD_GENERAL1_LARGE"))
	}
	return
}

func validateAwsCodeBuildEnvironmentType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	types := map[string]bool{
		"LINUX_CONTAINER": true,
	}

	if !types[value] {
		errors = append(errors, fmt.Errorf("CodeBuild: Environment Type can only be LINUX_CONTAINER"))
	}
	return
}

func validateAwsCodeBuildSourceType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	types := map[string]bool{
		codebuild.SourceTypeBitbucket:    true,
		codebuild.SourceTypeCodecommit:   true,
		codebuild.SourceTypeCodepipeline: true,
		codebuild.SourceTypeGithub:       true,
		codebuild.SourceTypeS3:           true,
	}
	s := make([]string, 0, len(types))

	for key, _ := range types {
		s = append(s, key)
	}

	if !types[value] {
		strings.Join(s, ", ")
		errors = append(errors, fmt.Errorf("CodeBuild: Source Type can only be one of: %s", strings.Join(s, ", ")))
	}
	return
}

func validateAwsCodeBuildSourceAuthType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	types := map[string]bool{
		"OAUTH": true,
	}

	if !types[value] {
		errors = append(errors, fmt.Errorf("CodeBuild: Source Auth Type can only be OAUTH"))
	}
	return
}

func validateAwsCodeBuildTimeout(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	if value < 5 || value > 480 {
		errors = append(errors, fmt.Errorf("%q must be greater than 5 minutes and less than 480 minutes (8 hours)", value))
	}
	return
}
