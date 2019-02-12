package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSagemakerModel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSagemakerModelCreate,
		Read:   resourceAwsSagemakerModelRead,
		Update: resourceAwsSagemakerModelUpdate,
		Delete: resourceAwsSagemakerModelDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateSagemakerName,
			},

			"primary_container": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"container_hostname": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateSagemakerName,
						},

						"image": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateSagemakerImage,
						},

						"model_data_url": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateSagemakerModelDataUrl,
						},

						"environment": {
							Type:         schema.TypeMap,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateSagemakerEnvironment,
						},
					},
				},
			},

			"vpc_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnets": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"security_group_ids": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"execution_role_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"enable_network_isolation": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"container": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"container_hostname": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateSagemakerName,
						},

						"image": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateSagemakerImage,
						},

						"model_data_url": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateSagemakerModelDataUrl,
						},

						"environment": {
							Type:         schema.TypeMap,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateSagemakerEnvironment,
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsSagemakerModelCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sagemakerconn

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		name = resource.UniqueId()
	}

	createOpts := &sagemaker.CreateModelInput{
		ModelName: aws.String(name),
	}

	if v, ok := d.GetOk("primary_container"); ok {
		m := v.([]interface{})[0].(map[string]interface{})
		createOpts.PrimaryContainer = expandContainer(m)
	}

	if v, ok := d.GetOk("container"); ok {
		containers := expandContainers(v.([]interface{}))
		createOpts.SetContainers(containers)
	}

	if v, ok := d.GetOk("execution_role_arn"); ok {
		createOpts.SetExecutionRoleArn(v.(string))
	}

	if v, ok := d.GetOk("tags"); ok {
		createOpts.SetTags(tagsFromMapSagemaker(v.(map[string]interface{})))
	}

	if v, ok := d.GetOk("vpc_config"); ok {
		vpcConfig := expandSageMakerVpcConfigRequest(v.([]interface{}))
		createOpts.SetVpcConfig(vpcConfig)
	}

	if v, ok := d.GetOk("enable_network_isolation"); ok {
		createOpts.SetEnableNetworkIsolation(v.(bool))
	}

	log.Printf("[DEBUG] Sagemaker model create config: %#v", *createOpts)
	_, err := retryOnAwsCode("ValidationException", func() (interface{}, error) {
		return conn.CreateModel(createOpts)
	})

	if err != nil {
		return fmt.Errorf("error creating Sagemaker model: %s", err)
	}
	d.SetId(name)

	return resourceAwsSagemakerModelRead(d, meta)
}

func expandSageMakerVpcConfigRequest(l []interface{}) *sagemaker.VpcConfig {
	if len(l) == 0 {
		return nil
	}

	m := l[0].(map[string]interface{})

	return &sagemaker.VpcConfig{
		SecurityGroupIds: expandStringSet(m["security_group_ids"].(*schema.Set)),
		Subnets:          expandStringSet(m["subnets"].(*schema.Set)),
	}
}

func resourceAwsSagemakerModelRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sagemakerconn

	request := &sagemaker.DescribeModelInput{
		ModelName: aws.String(d.Id()),
	}

	model, err := conn.DescribeModel(request)
	if err != nil {
		if sagemakerErr, ok := err.(awserr.Error); ok && sagemakerErr.Code() == "ValidationException" {
			log.Printf("[INFO] unable to find the sagemaker model resource and therefore it is removed from the state: %s", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Sagemaker model %s: %s", d.Id(), err)
	}

	if err := d.Set("arn", model.ModelArn); err != nil {
		return fmt.Errorf("unable to set arn for sagemaker model %q: %+v", d.Id(), err)
	}
	if err := d.Set("name", model.ModelName); err != nil {
		return err
	}
	if err := d.Set("execution_role_arn", model.ExecutionRoleArn); err != nil {
		return err
	}
	if err := d.Set("enable_network_isolation", model.EnableNetworkIsolation); err != nil {
		return err
	}
	if err := d.Set("primary_container", flattenContainer(model.PrimaryContainer)); err != nil {
		return err
	}
	if err := d.Set("container", flattenContainers(model.Containers)); err != nil {
		return err
	}
	if err := d.Set("vpc_config", flattenSageMakerVpcConfigResponse(model.VpcConfig)); err != nil {
		return fmt.Errorf("error setting vpc_config: %s", err)
	}

	tagsOutput, err := conn.ListTags(&sagemaker.ListTagsInput{
		ResourceArn: model.ModelArn,
	})
	if err != nil {
		return fmt.Errorf("error listing tags of Sagemaker model %s: %s", d.Id(), err)
	}

	if err := d.Set("tags", tagsToMapSagemaker(tagsOutput.Tags)); err != nil {
		return err
	}
	return nil
}

func flattenSageMakerVpcConfigResponse(vpcConfig *sagemaker.VpcConfig) []map[string]interface{} {
	if vpcConfig == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"security_group_ids": schema.NewSet(schema.HashString, flattenStringList(vpcConfig.SecurityGroupIds)),
		"subnets":            schema.NewSet(schema.HashString, flattenStringList(vpcConfig.Subnets)),
	}

	return []map[string]interface{}{m}
}

func resourceAwsSagemakerModelUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sagemakerconn

	d.Partial(true)

	if err := setSagemakerTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsSagemakerModelRead(d, meta)
}

func resourceAwsSagemakerModelDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sagemakerconn

	deleteOpts := &sagemaker.DeleteModelInput{
		ModelName: aws.String(d.Id()),
	}
	log.Printf("[INFO] Deleting Sagemaker model: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteModel(deleteOpts)
		if err == nil {
			return nil
		}

		sagemakerErr, ok := err.(awserr.Error)
		if !ok {
			return resource.NonRetryableError(err)
		}

		if sagemakerErr.Code() == "ResourceNotFound" {
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(fmt.Errorf("error deleting Sagemaker model: %s", err))
	})
}

func expandContainer(m map[string]interface{}) *sagemaker.ContainerDefinition {
	container := sagemaker.ContainerDefinition{
		Image: aws.String(m["image"].(string)),
	}

	if v, ok := m["container_hostname"]; ok && v.(string) != "" {
		container.ContainerHostname = aws.String(v.(string))
	}
	if v, ok := m["model_data_url"]; ok && v.(string) != "" {
		container.ModelDataUrl = aws.String(v.(string))
	}
	if v, ok := m["environment"]; ok {
		container.Environment = stringMapToPointers(v.(map[string]interface{}))
	}

	return &container
}

func expandContainers(a []interface{}) []*sagemaker.ContainerDefinition {
	containers := make([]*sagemaker.ContainerDefinition, 0, len(a))

	for _, m := range a {
		containers = append(containers, expandContainer(m.(map[string]interface{})))
	}

	return containers
}

func flattenContainer(container *sagemaker.ContainerDefinition) []interface{} {
	if container == nil {
		return []interface{}{}
	}

	cfg := make(map[string]interface{})

	cfg["image"] = *container.Image

	if container.ContainerHostname != nil {
		cfg["container_hostname"] = *container.ContainerHostname
	}
	if container.ModelDataUrl != nil {
		cfg["model_data_url"] = *container.ModelDataUrl
	}
	if container.Environment != nil {
		cfg["environment"] = flattenEnvironment(container.Environment)
	}

	return []interface{}{cfg}
}

func flattenContainers(containers []*sagemaker.ContainerDefinition) []interface{} {
	fContainers := make([]interface{}, 0, len(containers))
	for _, container := range containers {
		fContainers = append(fContainers, flattenContainer(container)[0].(map[string]interface{}))
	}
	return fContainers
}

func flattenEnvironment(env map[string]*string) map[string]string {
	m := map[string]string{}
	for k, v := range env {
		m[k] = *v
	}
	return m
}
