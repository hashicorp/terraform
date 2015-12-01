package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/devicefarm"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDeviceFarmDevicePool() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDeviceFarmDevicePoolCreate,
		Read:   resourceAwsDeviceFarmDevicePoolRead,
		Update: resourceAwsDeviceFarmDevicePoolUpdate,
		Delete: resourceAwsDeviceFarmDevicePoolDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"project_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"rules": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attribute": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"operator": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceAwsDeviceFarmDevicePoolRulesHash,
			},
		},
	}
}

func resourceAwsDeviceFarmDevicePoolCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).devicefarmconn
	region := meta.(*AWSClient).region

	//	We need to ensure that DeviceFarm is only being run against us-west-2
	//	As this is the only place that AWS currently supports it
	if region != "us-west-2" {
		return fmt.Errorf("DeviceFarm can only be used with us-west-2. You are trying to use it on %s", region)
	}

	rules, err := expandRules(d.Get("rules").(*schema.Set).List())
	if err != nil {
		return err
	}

	input := &devicefarm.CreateDevicePoolInput{
		Name:       aws.String(d.Get("name").(string)),
		ProjectArn: aws.String(d.Get("project_arn").(string)),
		Rules:      rules,
	}
	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating DeviceFarm DevicePool: %s", d.Get("name").(string))
	log.Printf("[DEBUG] Create DeviceFarm DevicePool with options: %#v", input)
	out, err := conn.CreateDevicePool(input)
	if err != nil {
		return fmt.Errorf("Error creating DeviceFarm DevicePool: %s", err)
	}

	log.Printf("[DEBUG] Successsfully Created DeviceFarm DevicePool: %s", *out.DevicePool.Arn)
	d.SetId(*out.DevicePool.Arn)

	return resourceAwsDeviceFarmDevicePoolRead(d, meta)
}

func resourceAwsDeviceFarmDevicePoolRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).devicefarmconn

	input := &devicefarm.GetDevicePoolInput{
		Arn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading DeviceFarm DevicePool: %s", d.Id())
	out, err := conn.GetDevicePool(input)
	if err != nil {
		return fmt.Errorf("Error reading DeviceFarm DevicePool: %s", err)
	}

	d.Set("name", out.DevicePool.Name)
	d.Set("description", out.DevicePool.Description)
	d.Set("rules", flattenRules(out.DevicePool.Rules))

	return nil
}

func resourceAwsDeviceFarmDevicePoolUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).devicefarmconn

	input := &devicefarm.UpdateDevicePoolInput{
		Arn: aws.String(d.Id()),
	}

	if d.HasChange("name") {
		input.Name = aws.String(d.Get("name").(string))
	}

	if d.HasChange("description") {
		input.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("rules") {
		rules, err := expandRules(d.Get("rules").(*schema.Set).List())
		if err != nil {
			return err
		}
		input.Rules = rules
	}

	log.Printf("[DEBUG] Updating DeviceFarm DevicePool: %s", d.Id())
	_, err := conn.UpdateDevicePool(input)
	if err != nil {
		return fmt.Errorf("Error Updating DeviceFarm DevicePool: %s", err)
	}

	return resourceAwsDeviceFarmDevicePoolRead(d, meta)
}

func resourceAwsDeviceFarmDevicePoolDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).devicefarmconn

	input := &devicefarm.DeleteDevicePoolInput{
		Arn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting DeviceFarm DevicePool: %s", d.Id())
	_, err := conn.DeleteDevicePool(input)
	if err != nil {
		return fmt.Errorf("Error deleting DeviceFarm DevicePool: %s", err)
	}

	return nil
}

func expandRules(configured []interface{}) ([]*devicefarm.Rule, error) {
	rules := make([]*devicefarm.Rule, 0, len(configured))

	for _, lRaw := range configured {
		data := lRaw.(map[string]interface{})

		rule := &devicefarm.Rule{
			Attribute: aws.String(data["attribute"].(string)),
			Operator:  aws.String(data["operator"].(string)),
			Value:     aws.String(data["value"].(string)),
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func flattenRules(list []*devicefarm.Rule) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, i := range list {
		l := map[string]interface{}{
			"attribute": *i.Attribute,
			"operator":  *i.Operator,
			"value":     *i.Value,
		}
		result = append(result, l)
	}
	return result
}

func resourceAwsDeviceFarmDevicePoolRulesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["attribute"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["operator"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["value"].(string))))

	return hashcode.String(buf.String())
}
