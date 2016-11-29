package rabbitmq

import (
	"fmt"
	"log"
	"strings"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePolicy() *schema.Resource {
	return &schema.Resource{
		Create: CreatePolicy,
		Update: UpdatePolicy,
		Read:   ReadPolicy,
		Delete: DeletePolicy,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vhost": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"pattern": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"priority": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"apply_to": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"definition": &schema.Schema{
							Type:     schema.TypeMap,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func CreatePolicy(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	name := d.Get("name").(string)
	vhost := d.Get("vhost").(string)
	policyList := d.Get("policy").([]interface{})

	policyMap, ok := policyList[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to parse policy")
	}

	if err := putPolicy(rmqc, vhost, name, policyMap); err != nil {
		return err
	}

	id := fmt.Sprintf("%s@%s", name, vhost)
	d.SetId(id)

	return ReadPolicy(d, meta)
}

func ReadPolicy(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	policyId := strings.Split(d.Id(), "@")
	if len(policyId) < 2 {
		return fmt.Errorf("Unable to determine policy ID")
	}

	user := policyId[0]
	vhost := policyId[1]

	policy, err := rmqc.GetPolicy(vhost, user)
	if err != nil {
		return checkDeleted(d, err)
	}

	log.Printf("[DEBUG] RabbitMQ: Policy retrieved for %s: %#v", d.Id(), policy)

	d.Set("name", policy.Name)
	d.Set("vhost", policy.Vhost)

	setPolicy := make([]map[string]interface{}, 1)
	p := make(map[string]interface{})
	p["pattern"] = policy.Pattern
	p["priority"] = policy.Priority
	p["apply_to"] = policy.ApplyTo

	policyDefinition := make(map[string]interface{})
	for key, value := range policy.Definition {
		if v, ok := value.([]interface{}); ok {
			var nodes []string
			for _, node := range v {
				if n, ok := node.(string); ok {
					nodes = append(nodes, n)
				}
			}
			value = strings.Join(nodes, ",")
		}
		policyDefinition[key] = value
	}
	p["definition"] = policyDefinition
	setPolicy[0] = p

	d.Set("policy", setPolicy)

	return nil
}

func UpdatePolicy(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	policyId := strings.Split(d.Id(), "@")
	if len(policyId) < 2 {
		return fmt.Errorf("Unable to determine policy ID")
	}

	user := policyId[0]
	vhost := policyId[1]

	if d.HasChange("policy") {
		_, newPolicy := d.GetChange("policy")

		policyList := newPolicy.([]interface{})
		policyMap, ok := policyList[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("Unable to parse policy")
		}

		if err := putPolicy(rmqc, user, vhost, policyMap); err != nil {
			return err
		}
	}

	return ReadPolicy(d, meta)
}

func DeletePolicy(d *schema.ResourceData, meta interface{}) error {
	rmqc := meta.(*rabbithole.Client)

	policyId := strings.Split(d.Id(), "@")
	if len(policyId) < 2 {
		return fmt.Errorf("Unable to determine policy ID")
	}

	user := policyId[0]
	vhost := policyId[1]

	log.Printf("[DEBUG] RabbitMQ: Attempting to delete policy for %s", d.Id())

	resp, err := rmqc.DeletePolicy(vhost, user)
	log.Printf("[DEBUG] RabbitMQ: Policy delete response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		// the policy was automatically deleted
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error deleting RabbitMQ policy: %s", resp.Status)
	}

	return nil
}

func putPolicy(rmqc *rabbithole.Client, vhost string, name string, policyMap map[string]interface{}) error {
	policy := rabbithole.Policy{}
	policy.Vhost = vhost
	policy.Name = name

	if v, ok := policyMap["pattern"].(string); ok {
		policy.Pattern = v
	}

	if v, ok := policyMap["priority"].(int); ok {
		policy.Priority = v
	}

	if v, ok := policyMap["apply_to"].(string); ok {
		policy.ApplyTo = v
	}

	if v, ok := policyMap["definition"].(map[string]interface{}); ok {
		// special case for ha-mode = nodes
		if x, ok := v["ha-mode"]; ok && x == "nodes" {
			var nodes rabbithole.NodeNames
			nodes = strings.Split(v["ha-params"].(string), ",")
			v["ha-params"] = nodes
		}
		policyDefinition := rabbithole.PolicyDefinition{}
		policyDefinition = v
		policy.Definition = policyDefinition
	}

	log.Printf("[DEBUG] RabbitMQ: Attempting to declare policy for %s@%s: %#v", name, vhost, policy)

	resp, err := rmqc.PutPolicy(vhost, name, policy)
	log.Printf("[DEBUG] RabbitMQ: Policy declare response: %#v", resp)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error declaring RabbitMQ policy: %s", resp.Status)
	}

	return nil
}
