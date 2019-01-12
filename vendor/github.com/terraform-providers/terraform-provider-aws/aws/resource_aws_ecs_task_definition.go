package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsEcsTaskDefinition() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcsTaskDefinitionCreate,
		Read:   resourceAwsEcsTaskDefinitionRead,
		Update: resourceAwsEcsTaskDefinitionUpdate,
		Delete: resourceAwsEcsTaskDefinitionDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("arn", d.Id())

				idErr := fmt.Errorf("Expected ID in format of arn:PARTITION:ecs:REGION:ACCOUNTID:task-definition/FAMILY:REVISION and provided: %s", d.Id())
				resARN, err := arn.Parse(d.Id())
				if err != nil {
					return nil, idErr
				}
				familyRevision := strings.TrimPrefix(resARN.Resource, "task-definition/")
				familyRevisionParts := strings.Split(familyRevision, ":")
				if len(familyRevisionParts) != 2 {
					return nil, idErr
				}
				d.SetId(familyRevisionParts[0])

				return []*schema.ResourceData{d}, nil
			},
		},

		SchemaVersion: 1,
		MigrateState:  resourceAwsEcsTaskDefinitionMigrateState,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cpu": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					networkMode, ok := d.GetOk("network_mode")
					isAWSVPC := ok && networkMode.(string) == ecs.NetworkModeAwsvpc
					equal, _ := EcsContainerDefinitionsAreEquivalent(old, new, isAWSVPC)
					return equal
				},
				ValidateFunc: validateAwsEcsTaskDefinitionContainerDefinitions,
			},

			"task_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"execution_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"memory": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"network_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					ecs.NetworkModeBridge,
					ecs.NetworkModeHost,
					ecs.NetworkModeAwsvpc,
					ecs.NetworkModeNone,
				}, false),
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

						"docker_volume_configuration": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"scope": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
										ValidateFunc: validation.StringInSlice([]string{
											ecs.ScopeShared,
											ecs.ScopeTask,
										}, false),
									},
									"autoprovision": {
										Type:     schema.TypeBool,
										Optional: true,
										ForceNew: true,
										Default:  false,
									},
									"driver": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
									"driver_opts": {
										Type:     schema.TypeMap,
										Elem:     &schema.Schema{Type: schema.TypeString},
										ForceNew: true,
										Optional: true,
									},
									"labels": {
										Type:     schema.TypeMap,
										Elem:     &schema.Schema{Type: schema.TypeString},
										ForceNew: true,
										Optional: true,
									},
								},
							},
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

			"requires_compatibilities": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"ipc_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					ecs.IpcModeHost,
					ecs.IpcModeNone,
					ecs.IpcModeTask,
				}, false),
			},

			"pid_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					ecs.PidModeHost,
					ecs.PidModeTask,
				}, false),
			},

			"tags": tagsSchema(),
		},
	}
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

	// ClientException: Tags can not be empty.
	if v, ok := d.GetOk("tags"); ok {
		input.Tags = tagsFromMapECS(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("task_role_arn"); ok {
		input.TaskRoleArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("execution_role_arn"); ok {
		input.ExecutionRoleArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cpu"); ok {
		input.Cpu = aws.String(v.(string))
	}

	if v, ok := d.GetOk("memory"); ok {
		input.Memory = aws.String(v.(string))
	}

	if v, ok := d.GetOk("network_mode"); ok {
		input.NetworkMode = aws.String(v.(string))
	}

	if v, ok := d.GetOk("ipc_mode"); ok {
		input.IpcMode = aws.String(v.(string))
	}

	if v, ok := d.GetOk("pid_mode"); ok {
		input.PidMode = aws.String(v.(string))
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

	if v, ok := d.GetOk("requires_compatibilities"); ok && v.(*schema.Set).Len() > 0 {
		input.RequiresCompatibilities = expandStringList(v.(*schema.Set).List())
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
		Include:        []*string{aws.String(ecs.TaskDefinitionFieldTags)},
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Received task definition %s, status:%s\n %s", aws.StringValue(out.TaskDefinition.Family),
		aws.StringValue(out.TaskDefinition.Status), out)

	taskDefinition := out.TaskDefinition

	if aws.StringValue(taskDefinition.Status) == "INACTIVE" {
		log.Printf("[DEBUG] Removing ECS task definition %s because it's INACTIVE", aws.StringValue(out.TaskDefinition.Family))
		d.SetId("")
		return nil
	}

	d.SetId(*taskDefinition.Family)
	d.Set("arn", taskDefinition.TaskDefinitionArn)
	d.Set("family", taskDefinition.Family)
	d.Set("revision", taskDefinition.Revision)

	defs, err := flattenEcsContainerDefinitions(taskDefinition.ContainerDefinitions)
	if err != nil {
		return err
	}
	err = d.Set("container_definitions", defs)
	if err != nil {
		return err
	}

	d.Set("task_role_arn", taskDefinition.TaskRoleArn)
	d.Set("execution_role_arn", taskDefinition.ExecutionRoleArn)
	d.Set("cpu", taskDefinition.Cpu)
	d.Set("memory", taskDefinition.Memory)
	d.Set("network_mode", taskDefinition.NetworkMode)

	if err := d.Set("tags", tagsToMapECS(out.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	if err := d.Set("volume", flattenEcsVolumes(taskDefinition.Volumes)); err != nil {
		return fmt.Errorf("error setting volume: %s", err)
	}

	if err := d.Set("placement_constraints", flattenPlacementConstraints(taskDefinition.PlacementConstraints)); err != nil {
		log.Printf("[ERR] Error setting placement_constraints for (%s): %s", d.Id(), err)
	}

	if err := d.Set("requires_compatibilities", flattenStringList(taskDefinition.RequiresCompatibilities)); err != nil {
		return err
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

func resourceAwsEcsTaskDefinitionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	if d.HasChange("tags") {
		oldTagsRaw, newTagsRaw := d.GetChange("tags")
		oldTagsMap := oldTagsRaw.(map[string]interface{})
		newTagsMap := newTagsRaw.(map[string]interface{})
		createTags, removeTags := diffTagsECS(tagsFromMapECS(oldTagsMap), tagsFromMapECS(newTagsMap))

		if len(removeTags) > 0 {
			removeTagKeys := make([]*string, len(removeTags))
			for i, removeTag := range removeTags {
				removeTagKeys[i] = removeTag.Key
			}

			input := &ecs.UntagResourceInput{
				ResourceArn: aws.String(d.Get("arn").(string)),
				TagKeys:     removeTagKeys,
			}

			log.Printf("[DEBUG] Untagging ECS Cluster: %s", input)
			if _, err := conn.UntagResource(input); err != nil {
				return fmt.Errorf("error untagging ECS Cluster (%s): %s", d.Get("arn").(string), err)
			}
		}

		if len(createTags) > 0 {
			input := &ecs.TagResourceInput{
				ResourceArn: aws.String(d.Get("arn").(string)),
				Tags:        createTags,
			}

			log.Printf("[DEBUG] Tagging ECS Cluster: %s", input)
			if _, err := conn.TagResource(input); err != nil {
				return fmt.Errorf("error tagging ECS Cluster (%s): %s", d.Get("arn").(string), err)
			}
		}
	}

	return nil
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
