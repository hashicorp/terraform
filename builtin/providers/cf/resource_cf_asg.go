package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAsg() *schema.Resource {

	return &schema.Resource{

		Create: resourceAsgCreate,
		Read:   resourceAsgRead,
		Update: resourceAsgUpdate,
		Delete: resourceAsgDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"rule": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"destination": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"ports": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"log": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
		},
	}
}

func resourceAsgCreate(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	am := session.ASGManager()
	id, err := am.CreateASG(d.Get("name").(string), readASGRulesFromConfig(d))
	if err != nil {
		return err
	}
	d.SetId(id)

	return nil
}

func resourceAsgRead(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	am := session.ASGManager()
	asg, err := am.GetASG(d.Id())
	if err != nil {
		return err
	}

	session.Log.DebugMessage("Read ASG from CC: %# v", asg)

	d.Set("name", asg.Name)

	tfRules := []interface{}{}
	for _, r := range asg.Rules {
		tfRule := make(map[string]interface{})
		tfRule["protocol"] = r.Protocol
		tfRule["destination"] = r.Destination
		if len(r.Ports) > 0 {
			tfRule["ports"] = r.Ports
		}
		tfRule["log"] = r.Log
		tfRules = append(tfRules, tfRule)
	}
	d.Set("rule", tfRules)

	return nil
}

func resourceAsgUpdate(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	am := session.ASGManager()
	err := am.UpdateASG(d.Id(), d.Get("name").(string), readASGRulesFromConfig(d))
	if err != nil {
		return err
	}
	return nil
}

func resourceAsgDelete(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}
	return session.ASGManager().Delete(d.Id())
}

func readASGRulesFromConfig(d *schema.ResourceData) (rules []cfapi.CCASGRule) {

	rules = []cfapi.CCASGRule{}
	for _, r := range d.Get("rule").([]interface{}) {

		tfRule := r.(map[string]interface{})
		asgRule := cfapi.CCASGRule{
			Protocol:    tfRule["protocol"].(string),
			Destination: tfRule["destination"].(string),
		}
		if v, ok := tfRule["ports"]; ok {
			asgRule.Ports = v.(string)
		}
		if v, ok := tfRule["log"]; ok {
			asgRule.Log = v.(bool)
		}
		rules = append(rules, asgRule)
	}
	return
}
