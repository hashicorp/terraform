package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEcsTaskDefinition() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcsTaskDefinitionCreate,
		Read:   resourceAwsEcsTaskDefinitionRead,
		Delete: resourceAwsEcsTaskDefinitionDelete,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"family": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"revision": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"container_definitions": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					hash := sha1.Sum([]byte(v.(string)))
					return hex.EncodeToString(hash[:])
				},
				ValidateFunc: validateAwsEcsTaskDefinitionContainerDefinitions,
			},

			"task_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"network_mode": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsEcsTaskDefinitionNetworkMode,
			},

			"volume": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"host_path": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceAwsEcsTaskDefinitionVolumeHash,
			},

			"placement_constraints": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 10,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"expression": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func validateAwsEcsTaskDefinitionNetworkMode(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	validTypes := map[string]struct{}{
		"bridge": {},
		"host":   {},
		"none":   {},
	}

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf("ECS Task Definition network_mode %q is invalid, must be `bridge`, `host` or `none`", value))
	}
	return
}

func validateAwsEcsTaskDefinitionContainerDefinitions(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, err := expandEcsContainerDefinitions(value)
	if err != nil {
		errors = append(errors, fmt.Errorf("ECS Task Definition container_definitions is invalid: %s", err))
	}
	return
}

func resourceAwsEcsTaskDefinitionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	rawDefinitions := d.Get("container_definitions").(string)
	definitions, err := expandEcsContainerDefinitions(rawDefinitions)
	if err != nil {
		return err
	}

	input := ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: definitions,
		Family:               aws.String(d.Get("family").(string)),
	}

	if v, ok := d.GetOk("task_role_arn"); ok {
		input.TaskRoleArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("network_mode"); ok {
		input.NetworkMode = aws.String(v.(string))
	}

	if v, ok := d.GetOk("volume"); ok {
		volumes, err := expandEcsVolumes(v.(*schema.Set).List())
		if err != nil {
			return err
		}
		input.Volumes = volumes
	}

	constraints := d.Get("placement_constraints").(*schema.Set).List()
	if len(constraints) > 0 {
		var pc []*ecs.TaskDefinitionPlacementConstraint
		for _, raw := range constraints {
			p := raw.(map[string]interface{})
			t := p["type"].(string)
			e := p["expression"].(string)
			if err := validateAwsEcsPlacementConstraint(t, e); err != nil {
				return err
			}
			pc = append(pc, &ecs.TaskDefinitionPlacementConstraint{
				Type:       aws.String(t),
				Expression: aws.String(e),
			})
		}
		input.PlacementConstraints = pc
	}

	log.Printf("[DEBUG] Registering ECS task definition: %s", input)
	out, err := conn.RegisterTaskDefinition(&input)
	if err != nil {
		return err
	}

	taskDefinition := *out.TaskDefinition

	log.Printf("[DEBUG] ECS task definition registered: %q (rev. %d)",
		*taskDefinition.TaskDefinitionArn, *taskDefinition.Revision)

	d.SetId(*taskDefinition.Family)
	d.Set("arn", taskDefinition.TaskDefinitionArn)

	return resourceAwsEcsTaskDefinitionRead(d, meta)
}

func resourceAwsEcsTaskDefinitionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	log.Printf("[DEBUG] Reading task definition %s", d.Id())
	out, err := conn.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(d.Get("arn").(string)),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Received task definition %s", out)

	taskDefinition := out.TaskDefinition

	d.SetId(*taskDefinition.Family)
	d.Set("arn", taskDefinition.TaskDefinitionArn)
	d.Set("family", taskDefinition.Family)
	d.Set("revision", taskDefinition.Revision)
	d.Set("container_definitions", taskDefinition.ContainerDefinitions)
	d.Set("task_role_arn", taskDefinition.TaskRoleArn)
	d.Set("network_mode", taskDefinition.NetworkMode)
	d.Set("volumes", flattenEcsVolumes(taskDefinition.Volumes))
	if err := d.Set("placement_constraints", flattenPlacementConstraints(taskDefinition.PlacementConstraints)); err != nil {
		log.Printf("[ERR] Error setting placement_constraints for (%s): %s", d.Id(), err)
	}

	return nil
}

func flattenPlacementConstraints(pcs []*ecs.TaskDefinitionPlacementConstraint) []map[string]interface{} {
	if len(pcs) == 0 {
		return nil
	}
	results := make([]map[string]interface{}, 0)
	for _, pc := range pcs {
		c := make(map[string]interface{})
		c["type"] = *pc.Type
		c["expression"] = *pc.Expression
		results = append(results, c)
	}
	return results
}

func resourceAwsEcsTaskDefinitionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	_, err := conn.DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: aws.String(d.Get("arn").(string)),
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Task definition %q deregistered.", d.Get("arn").(string))

	return nil
}

func resourceAwsEcsTaskDefinitionVolumeHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["host_path"].(string)))

	return hashcode.String(buf.String())
}
