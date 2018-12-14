package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/hashicorp/terraform/helper/customdiff"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsCodeBuildProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeBuildProjectCreate,
		Read:   resourceAwsCodeBuildProjectRead,
		Update: resourceAwsCodeBuildProjectUpdate,
		Delete: resourceAwsCodeBuildProjectDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
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
						"encryption_disabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"location": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"namespace_type": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								codebuild.ArtifactNamespaceNone,
								codebuild.ArtifactNamespaceBuildId,
							}, false),
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
							ValidateFunc: validation.StringInSlice([]string{
								codebuild.ArtifactsTypeCodepipeline,
								codebuild.ArtifactsTypeS3,
								codebuild.ArtifactsTypeNoArtifacts,
							}, false),
						},
					},
				},
				Set: resourceAwsCodeBuildProjectArtifactsHash,
			},
			"cache": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  codebuild.CacheTypeNoCache,
							ValidateFunc: validation.StringInSlice([]string{
								codebuild.CacheTypeNoCache,
								codebuild.CacheTypeS3,
							}, false),
						},
						"location": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringLenBetween(0, 255),
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
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								codebuild.ComputeTypeBuildGeneral1Small,
								codebuild.ComputeTypeBuildGeneral1Medium,
								codebuild.ComputeTypeBuildGeneral1Large,
							}, false),
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
									"type": {
										Type:     schema.TypeString,
										Optional: true,
										ValidateFunc: validation.StringInSlice([]string{
											codebuild.EnvironmentVariableTypePlaintext,
											codebuild.EnvironmentVariableTypeParameterStore,
										}, false),
										Default: codebuild.EnvironmentVariableTypePlaintext,
									},
								},
							},
						},
						"image": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								codebuild.EnvironmentTypeLinuxContainer,
								codebuild.EnvironmentTypeWindowsContainer,
							}, false),
						},
						"privileged_mode": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"certificate": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringMatch(regexp.MustCompile(`\.(pem|zip)$`), "must end in .pem or .zip"),
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
			"secondary_artifacts": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      resourceAwsCodeBuildProjectArtifactsHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"encryption_disabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"location": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"namespace_type": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								codebuild.ArtifactNamespaceNone,
								codebuild.ArtifactNamespaceBuildId,
							}, false),
						},
						"packaging": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"artifact_identifier": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								codebuild.ArtifactsTypeCodepipeline,
								codebuild.ArtifactsTypeS3,
								codebuild.ArtifactsTypeNoArtifacts,
							}, false),
						},
					},
				},
			},
			"secondary_sources": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auth": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resource": {
										Type:      schema.TypeString,
										Sensitive: true,
										Optional:  true,
									},
									"type": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											codebuild.SourceAuthTypeOauth,
										}, false),
									},
								},
							},
							Optional: true,
							Set:      resourceAwsCodeBuildProjectSourceAuthHash,
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
							ValidateFunc: validation.StringInSlice([]string{
								codebuild.SourceTypeCodecommit,
								codebuild.SourceTypeCodepipeline,
								codebuild.SourceTypeGithub,
								codebuild.SourceTypeS3,
								codebuild.SourceTypeBitbucket,
								codebuild.SourceTypeGithubEnterprise,
							}, false),
						},
						"git_clone_depth": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntAtLeast(0),
						},
						"insecure_ssl": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"report_build_status": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"source_identifier": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"service_role": {
				Type:     schema.TypeString,
				Required: true,
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
										Type:      schema.TypeString,
										Sensitive: true,
										Optional:  true,
									},
									"type": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											codebuild.SourceAuthTypeOauth,
										}, false),
									},
								},
							},
							Optional: true,
							Set:      resourceAwsCodeBuildProjectSourceAuthHash,
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
							ValidateFunc: validation.StringInSlice([]string{
								codebuild.SourceTypeCodecommit,
								codebuild.SourceTypeCodepipeline,
								codebuild.SourceTypeGithub,
								codebuild.SourceTypeS3,
								codebuild.SourceTypeBitbucket,
								codebuild.SourceTypeGithubEnterprise,
								codebuild.SourceTypeNoSource,
							}, false),
						},
						"git_clone_depth": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntAtLeast(0),
						},
						"insecure_ssl": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"report_build_status": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
				Required: true,
				MaxItems: 1,
				Set:      resourceAwsCodeBuildProjectSourceHash,
			},
			"timeout": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(5, 480),
				Removed:      "This field has been removed. Please use build_timeout instead",
			},
			"build_timeout": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      "60",
				ValidateFunc: validation.IntBetween(5, 480),
			},
			"badge_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"badge_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
			"vpc_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vpc_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"subnets": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							// Set:      schema.HashString,
							MaxItems: 16,
						},
						"security_group_ids": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							// Set:      schema.HashString,
							MaxItems: 5,
						},
					},
				},
			},
		},

		CustomizeDiff: customdiff.Sequence(
			func(diff *schema.ResourceDiff, v interface{}) error {
				// Plan time validation for cache location
				cacheType, cacheTypeOk := diff.GetOk("cache.0.type")
				if !cacheTypeOk || cacheType.(string) == codebuild.CacheTypeNoCache {
					return nil
				}
				if v, ok := diff.GetOk("cache.0.location"); ok && v.(string) != "" {
					return nil
				}
				return fmt.Errorf(`cache location is required when cache type is %q`, cacheType.(string))
			},
		),
	}
}

func resourceAwsCodeBuildProjectCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codebuildconn

	projectEnv := expandProjectEnvironment(d)
	projectSource := expandProjectSource(d)
	projectArtifacts := expandProjectArtifacts(d)
	projectSecondaryArtifacts := expandProjectSecondaryArtifacts(d)
	projectSecondarySources := expandProjectSecondarySources(d)

	if aws.StringValue(projectSource.Type) == codebuild.SourceTypeNoSource {
		if aws.StringValue(projectSource.Buildspec) == "" {
			return fmt.Errorf("`build_spec` must be set when source's `type` is `NO_SOURCE`")
		}

		if aws.StringValue(projectSource.Location) != "" {
			return fmt.Errorf("`location` must be empty when source's `type` is `NO_SOURCE`")
		}
	}

	params := &codebuild.CreateProjectInput{
		Environment:        projectEnv,
		Name:               aws.String(d.Get("name").(string)),
		Source:             &projectSource,
		Artifacts:          &projectArtifacts,
		SecondaryArtifacts: projectSecondaryArtifacts,
		SecondarySources:   projectSecondarySources,
	}

	if v, ok := d.GetOk("cache"); ok {
		params.Cache = expandProjectCache(v.([]interface{}))
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

	if v, ok := d.GetOk("vpc_config"); ok {
		params.VpcConfig = expandCodeBuildVpcConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("badge_enabled"); ok {
		params.BadgeEnabled = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("tags"); ok {
		params.Tags = tagsFromMapCodeBuild(v.(map[string]interface{}))
	}

	var resp *codebuild.CreateProjectOutput
	// Handle IAM eventual consistency
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		var err error

		resp, err = conn.CreateProject(params)
		if err != nil {
			// InvalidInputException: CodeBuild is not authorized to perform
			// InvalidInputException: Not authorized to perform DescribeSecurityGroups
			if isAWSErr(err, "InvalidInputException", "ot authorized to perform") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil

	})

	if err != nil {
		return fmt.Errorf("Error creating CodeBuild project: %s", err)
	}

	d.SetId(*resp.Project.Arn)

	return resourceAwsCodeBuildProjectRead(d, meta)
}

func expandProjectSecondaryArtifacts(d *schema.ResourceData) []*codebuild.ProjectArtifacts {
	artifacts := make([]*codebuild.ProjectArtifacts, 0)

	configsList := d.Get("secondary_artifacts").(*schema.Set).List()

	if len(configsList) == 0 {
		return nil
	}

	for _, config := range configsList {
		art := expandProjectArtifactData(config.(map[string]interface{}))
		artifacts = append(artifacts, &art)
	}

	return artifacts
}

func expandProjectArtifacts(d *schema.ResourceData) codebuild.ProjectArtifacts {
	configs := d.Get("artifacts").(*schema.Set).List()
	data := configs[0].(map[string]interface{})

	return expandProjectArtifactData(data)
}

func expandProjectArtifactData(data map[string]interface{}) codebuild.ProjectArtifacts {
	artifactType := data["type"].(string)

	projectArtifacts := codebuild.ProjectArtifacts{
		Type: aws.String(artifactType),
	}

	// Only valid for S3 and CODEPIPELINE artifacts types
	// InvalidInputException: Invalid artifacts: artifact type NO_ARTIFACTS should have null encryptionDisabled
	if artifactType == codebuild.ArtifactsTypeS3 || artifactType == codebuild.ArtifactsTypeCodepipeline {
		projectArtifacts.EncryptionDisabled = aws.Bool(data["encryption_disabled"].(bool))
	}

	if data["artifact_identifier"] != nil && data["artifact_identifier"].(string) != "" {
		projectArtifacts.ArtifactIdentifier = aws.String(data["artifact_identifier"].(string))
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

func expandProjectCache(s []interface{}) *codebuild.ProjectCache {
	var projectCache *codebuild.ProjectCache

	data := s[0].(map[string]interface{})

	projectCache = &codebuild.ProjectCache{
		Type: aws.String(data["type"].(string)),
	}

	if v, ok := data["location"]; ok {
		projectCache.Location = aws.String(v.(string))
	}

	return projectCache
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

	if v, ok := envConfig["certificate"]; ok && v.(string) != "" {
		projectEnv.Certificate = aws.String(v.(string))
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

				if v := config["type"].(string); v != "" {
					projectEnvironmentVar.Type = &v
				}

				projectEnvironmentVariables = append(projectEnvironmentVariables, projectEnvironmentVar)
			}

			projectEnv.EnvironmentVariables = projectEnvironmentVariables
		}
	}

	return projectEnv
}

func expandCodeBuildVpcConfig(rawVpcConfig []interface{}) *codebuild.VpcConfig {
	vpcConfig := codebuild.VpcConfig{}
	if len(rawVpcConfig) == 0 || rawVpcConfig[0] == nil {
		return &vpcConfig
	}

	data := rawVpcConfig[0].(map[string]interface{})
	vpcConfig.VpcId = aws.String(data["vpc_id"].(string))
	vpcConfig.Subnets = expandStringList(data["subnets"].(*schema.Set).List())
	vpcConfig.SecurityGroupIds = expandStringList(data["security_group_ids"].(*schema.Set).List())

	return &vpcConfig
}

func expandProjectSecondarySources(d *schema.ResourceData) []*codebuild.ProjectSource {
	configs := d.Get("secondary_sources").(*schema.Set).List()

	if len(configs) == 0 {
		return nil
	}

	sources := make([]*codebuild.ProjectSource, 0)

	for _, config := range configs {
		source := expandProjectSourceData(config.(map[string]interface{}))
		sources = append(sources, &source)
	}

	return sources
}

func expandProjectSource(d *schema.ResourceData) codebuild.ProjectSource {
	configs := d.Get("source").(*schema.Set).List()

	data := configs[0].(map[string]interface{})
	return expandProjectSourceData(data)
}

func expandProjectSourceData(data map[string]interface{}) codebuild.ProjectSource {
	sourceType := data["type"].(string)

	projectSource := codebuild.ProjectSource{
		Buildspec:     aws.String(data["buildspec"].(string)),
		GitCloneDepth: aws.Int64(int64(data["git_clone_depth"].(int))),
		InsecureSsl:   aws.Bool(data["insecure_ssl"].(bool)),
		Type:          aws.String(sourceType),
	}

	if data["source_identifier"] != nil {
		projectSource.SourceIdentifier = aws.String(data["source_identifier"].(string))
	}

	if data["location"].(string) != "" {
		projectSource.Location = aws.String(data["location"].(string))
	}

	// Only valid for BITBUCKET and GITHUB source type, e.g.
	// InvalidInputException: Source type GITHUB_ENTERPRISE does not support ReportBuildStatus
	if sourceType == codebuild.SourceTypeBitbucket || sourceType == codebuild.SourceTypeGithub {
		projectSource.ReportBuildStatus = aws.Bool(data["report_build_status"].(bool))
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
		return fmt.Errorf("Error retreiving Projects: %q", err)
	}

	// if nothing was found, then return no state
	if len(resp.Projects) == 0 {
		log.Printf("[INFO]: No projects were found, removing from state")
		d.SetId("")
		return nil
	}

	project := resp.Projects[0]

	if err := d.Set("artifacts", flattenAwsCodeBuildProjectArtifacts(project.Artifacts)); err != nil {
		return err
	}

	if err := d.Set("environment", schema.NewSet(resourceAwsCodeBuildProjectEnvironmentHash, flattenAwsCodeBuildProjectEnvironment(project.Environment))); err != nil {
		return err
	}

	if err := d.Set("cache", flattenAwsCodebuildProjectCache(project.Cache)); err != nil {
		return err
	}

	if err := d.Set("secondary_artifacts", flattenAwsCodeBuildProjectSecondaryArtifacts(project.SecondaryArtifacts)); err != nil {
		return err
	}

	if err := d.Set("secondary_sources", flattenAwsCodeBuildProjectSecondarySources(project.SecondarySources)); err != nil {
		return err
	}

	if err := d.Set("source", flattenAwsCodeBuildProjectSource(project.Source)); err != nil {
		return err
	}

	if err := d.Set("vpc_config", flattenAwsCodeBuildVpcConfig(project.VpcConfig)); err != nil {
		return err
	}

	d.Set("arn", project.Arn)
	d.Set("description", project.Description)
	d.Set("encryption_key", project.EncryptionKey)
	d.Set("name", project.Name)
	d.Set("service_role", project.ServiceRole)
	d.Set("build_timeout", project.TimeoutInMinutes)
	if project.Badge != nil {
		d.Set("badge_enabled", project.Badge.BadgeEnabled)
		d.Set("badge_url", project.Badge.BadgeRequestUrl)
	} else {
		d.Set("badge_enabled", false)
		d.Set("badge_url", "")
	}

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

	if d.HasChange("secondary_sources") {
		projectSecondarySources := expandProjectSecondarySources(d)
		params.SecondarySources = projectSecondarySources
	}

	if d.HasChange("secondary_artifacts") {
		projectSecondaryArtifacts := expandProjectSecondaryArtifacts(d)
		params.SecondaryArtifacts = projectSecondaryArtifacts
	}

	if d.HasChange("vpc_config") {
		params.VpcConfig = expandCodeBuildVpcConfig(d.Get("vpc_config").([]interface{}))
	}

	if d.HasChange("cache") {
		if v, ok := d.GetOk("cache"); ok {
			params.Cache = expandProjectCache(v.([]interface{}))
		} else {
			params.Cache = &codebuild.ProjectCache{
				Type: aws.String("NO_CACHE"),
			}
		}
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

	if d.HasChange("badge_enabled") {
		params.BadgeEnabled = aws.Bool(d.Get("badge_enabled").(bool))
	}

	// The documentation clearly says "The replacement set of tags for this build project."
	// But its a slice of pointers so if not set for every update, they get removed.
	params.Tags = tagsFromMapCodeBuild(d.Get("tags").(map[string]interface{}))

	// Handle IAM eventual consistency
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error

		_, err = conn.UpdateProject(params)
		if err != nil {
			// InvalidInputException: CodeBuild is not authorized to perform
			// InvalidInputException: Not authorized to perform DescribeSecurityGroups
			if isAWSErr(err, "InvalidInputException", "ot authorized to perform") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil

	})

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

	return nil
}

func flattenAwsCodeBuildProjectSecondaryArtifacts(artifactsList []*codebuild.ProjectArtifacts) *schema.Set {
	artifactSet := schema.Set{
		F: resourceAwsCodeBuildProjectArtifactsHash,
	}

	for _, artifacts := range artifactsList {
		artifactSet.Add(flattenAwsCodeBuildProjectArtifactsData(*artifacts))
	}
	return &artifactSet
}

func flattenAwsCodeBuildProjectArtifacts(artifacts *codebuild.ProjectArtifacts) *schema.Set {

	artifactSet := schema.Set{
		F: resourceAwsCodeBuildProjectArtifactsHash,
	}

	values := flattenAwsCodeBuildProjectArtifactsData(*artifacts)

	artifactSet.Add(values)

	return &artifactSet
}

func flattenAwsCodeBuildProjectArtifactsData(artifacts codebuild.ProjectArtifacts) map[string]interface{} {
	values := map[string]interface{}{}

	values["type"] = *artifacts.Type

	if artifacts.ArtifactIdentifier != nil {
		values["artifact_identifier"] = *artifacts.ArtifactIdentifier
	}

	if artifacts.EncryptionDisabled != nil {
		values["encryption_disabled"] = *artifacts.EncryptionDisabled
	}
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
	return values
}

func flattenAwsCodebuildProjectCache(cache *codebuild.ProjectCache) []interface{} {
	if cache == nil {
		return []interface{}{}
	}

	values := map[string]interface{}{
		"location": aws.StringValue(cache.Location),
		"type":     aws.StringValue(cache.Type),
	}

	return []interface{}{values}
}

func flattenAwsCodeBuildProjectEnvironment(environment *codebuild.ProjectEnvironment) []interface{} {
	envConfig := map[string]interface{}{}

	envConfig["type"] = *environment.Type
	envConfig["compute_type"] = *environment.ComputeType
	envConfig["image"] = *environment.Image
	envConfig["certificate"] = aws.StringValue(environment.Certificate)
	envConfig["privileged_mode"] = *environment.PrivilegedMode

	if environment.EnvironmentVariables != nil {
		envConfig["environment_variable"] = environmentVariablesToMap(environment.EnvironmentVariables)
	}

	return []interface{}{envConfig}

}

func flattenAwsCodeBuildProjectSecondarySources(sourceList []*codebuild.ProjectSource) []interface{} {
	l := make([]interface{}, 0)

	for _, source := range sourceList {
		l = append(l, flattenAwsCodeBuildProjectSourceData(source))
	}

	return l
}

func flattenAwsCodeBuildProjectSource(source *codebuild.ProjectSource) []interface{} {
	l := make([]interface{}, 1)

	l[0] = flattenAwsCodeBuildProjectSourceData(source)

	return l
}

func flattenAwsCodeBuildProjectSourceData(source *codebuild.ProjectSource) interface{} {
	m := map[string]interface{}{
		"buildspec":           aws.StringValue(source.Buildspec),
		"location":            aws.StringValue(source.Location),
		"git_clone_depth":     int(aws.Int64Value(source.GitCloneDepth)),
		"insecure_ssl":        aws.BoolValue(source.InsecureSsl),
		"report_build_status": aws.BoolValue(source.ReportBuildStatus),
		"type":                aws.StringValue(source.Type),
	}

	if source.Auth != nil {
		m["auth"] = schema.NewSet(resourceAwsCodeBuildProjectSourceAuthHash, []interface{}{sourceAuthToMap(source.Auth)})
	}

	if source.SourceIdentifier != nil {
		m["source_identifier"] = aws.StringValue(source.SourceIdentifier)
	}

	return m
}

func flattenAwsCodeBuildVpcConfig(vpcConfig *codebuild.VpcConfig) []interface{} {
	if vpcConfig != nil {
		values := map[string]interface{}{}

		values["vpc_id"] = *vpcConfig.VpcId
		values["subnets"] = schema.NewSet(schema.HashString, flattenStringList(vpcConfig.Subnets))
		values["security_group_ids"] = schema.NewSet(schema.HashString, flattenStringList(vpcConfig.SecurityGroupIds))

		return []interface{}{values}
	}
	return nil
}

func resourceAwsCodeBuildProjectArtifactsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))

	if v, ok := m["artifact_identifier"]; ok {
		buf.WriteString(fmt.Sprintf("%s:", v.(string)))
	}
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
	if v, ok := m["certificate"]; ok && v.(string) != "" {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	for _, e := range environmentVariables {
		if e != nil { // Old statefiles might have nil values in them
			ev := e.(map[string]interface{})
			buf.WriteString(fmt.Sprintf("%s:", ev["name"].(string)))
			// type is sometimes not returned by the API
			if v, ok := ev["type"]; ok {
				buf.WriteString(fmt.Sprintf("%s:", v.(string)))
			}
			buf.WriteString(fmt.Sprintf("%s-", ev["value"].(string)))
		}
	}

	return hashcode.String(buf.String())
}

func resourceAwsCodeBuildProjectSourceHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))
	if v, ok := m["source_identifier"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", strconv.Itoa(v.(int))))
	}
	if v, ok := m["buildspec"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["location"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["git_clone_depth"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", strconv.Itoa(v.(int))))
	}
	if v, ok := m["insecure_ssl"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", strconv.FormatBool(v.(bool))))
	}
	if v, ok := m["report_build_status"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", strconv.FormatBool(v.(bool))))
	}

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
			if env.Type != nil {
				item["type"] = *env.Type
			}
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
