package google

import (
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/cloudresourcemanager/v1"
)

func dataSourceGoogleIamPolicy() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGoogleIamPolicyRead,

		Schema: map[string]*schema.Schema{
			"binding": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
						"members": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},
			"policy": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceGoogleIamPolicyMembers(d *schema.Set) []string {
	var members []string
	members = make([]string, d.Len())

	for i, v := range d.List() {
		members[i] = v.(string)
	}
	return members
}

func dataSourceGoogleIamPolicyRead(d *schema.ResourceData, meta interface{}) error {
	doc := &cloudresourcemanager.Policy{}

	var bindings []*cloudresourcemanager.Binding

	bindingStatements := d.Get("binding").(*schema.Set)
	bindings = make([]*cloudresourcemanager.Binding, bindingStatements.Len())
	doc.Bindings = bindings

	for i, bindingRaw := range bindingStatements.List() {
		bindingStatement := bindingRaw.(map[string]interface{})
		doc.Bindings[i] = &cloudresourcemanager.Binding{
			Role:    bindingStatement["role"].(string),
			Members: dataSourceGoogleIamPolicyMembers(bindingStatement["members"].(*schema.Set)),
		}
	}

	jsonDoc, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		// should never happen if the above code is correct
		return err
	}
	jsonString := string(jsonDoc)

	d.Set("policy", jsonString)
	d.SetId(strconv.Itoa(hashcode.String(jsonString)))

	return nil
}
