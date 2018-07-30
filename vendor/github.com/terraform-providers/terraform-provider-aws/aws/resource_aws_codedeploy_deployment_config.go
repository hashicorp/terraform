package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsCodeDeployDeploymentConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeDeployDeploymentConfigCreate,
		Read:   resourceAwsCodeDeployDeploymentConfigRead,
		Delete: resourceAwsCodeDeployDeploymentConfigDelete,

		Schema: map[string]*schema.Schema{
			"deployment_config_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"minimum_healthy_hosts": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								codedeploy.MinimumHealthyHostsTypeHostCount,
								codedeploy.MinimumHealthyHostsTypeFleetPercent,
							}, false),
						},
						"value": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},

			"deployment_config_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsCodeDeployDeploymentConfigCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	input := &codedeploy.CreateDeploymentConfigInput{
		DeploymentConfigName: aws.String(d.Get("deployment_config_name").(string)),
		MinimumHealthyHosts:  expandAwsCodeDeployConfigMinimumHealthHosts(d),
	}

	_, err := conn.CreateDeploymentConfig(input)
	if err != nil {
		return err
	}

	d.SetId(d.Get("deployment_config_name").(string))

	return resourceAwsCodeDeployDeploymentConfigRead(d, meta)
}

func resourceAwsCodeDeployDeploymentConfigRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	input := &codedeploy.GetDeploymentConfigInput{
		DeploymentConfigName: aws.String(d.Id()),
	}

	resp, err := conn.GetDeploymentConfig(input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if "DeploymentConfigDoesNotExistException" == awsErr.Code() {
				log.Printf("[DEBUG] CodeDeploy Deployment Config (%s) not found", d.Id())
				d.SetId("")
				return nil
			}
		}
		return err
	}

	if resp.DeploymentConfigInfo == nil {
		return fmt.Errorf("[ERROR] Cannot find DeploymentConfig %q", d.Id())
	}

	if err := d.Set("minimum_healthy_hosts", flattenAwsCodeDeployConfigMinimumHealthHosts(resp.DeploymentConfigInfo.MinimumHealthyHosts)); err != nil {
		return err
	}
	d.Set("deployment_config_id", resp.DeploymentConfigInfo.DeploymentConfigId)
	d.Set("deployment_config_name", resp.DeploymentConfigInfo.DeploymentConfigName)

	return nil
}

func resourceAwsCodeDeployDeploymentConfigDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	input := &codedeploy.DeleteDeploymentConfigInput{
		DeploymentConfigName: aws.String(d.Id()),
	}

	_, err := conn.DeleteDeploymentConfig(input)
	if err != nil {
		return err
	}

	return nil
}

func expandAwsCodeDeployConfigMinimumHealthHosts(d *schema.ResourceData) *codedeploy.MinimumHealthyHosts {
	hosts := d.Get("minimum_healthy_hosts").([]interface{})
	host := hosts[0].(map[string]interface{})

	minimumHealthyHost := codedeploy.MinimumHealthyHosts{
		Type:  aws.String(host["type"].(string)),
		Value: aws.Int64(int64(host["value"].(int))),
	}

	return &minimumHealthyHost
}

func flattenAwsCodeDeployConfigMinimumHealthHosts(hosts *codedeploy.MinimumHealthyHosts) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)

	item := make(map[string]interface{})

	item["type"] = *hosts.Type
	item["value"] = *hosts.Value

	result = append(result, item)

	return result
}
