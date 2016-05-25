package aws

import (
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

var resourceAwsIamPolicyDocumentVarReplacer = strings.NewReplacer("&{", "${")

func resourceAwsIamPolicyDocument() *schema.Resource {
	setOfString := &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}

	return &schema.Resource{
		Create: resourceAwsIamPolicyDocumentUpdate,
		Read:   resourceAwsIamPolicyDocumentNoop,
		Update: resourceAwsIamPolicyDocumentUpdate,
		Delete: resourceAwsIamPolicyDocumentDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"statement": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"effect": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Allow",
						},
						"actions":        setOfString,
						"not_actions":    setOfString,
						"resources":      setOfString,
						"not_resources":  setOfString,
						"principals":     resourceAwsIamPolicyPrincipalSchema(),
						"not_principals": resourceAwsIamPolicyPrincipalSchema(),
						"condition": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"test": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"variable": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"values": &schema.Schema{
										Type:     schema.TypeSet,
										Required: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
			"json": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsIamPolicyDocumentUpdate(d *schema.ResourceData, meta interface{}) error {

	doc := &IAMPolicyDoc{
		Version: "2012-10-17",
	}

	id := d.Get("id").(string)
	if id != "" {
		doc.Id = id
	}

	var cfgStmts = d.Get("statement").(*schema.Set).List()
	stmts := make([]*IAMPolicyStatement, len(cfgStmts))
	doc.Statements = stmts
	for i, stmtI := range cfgStmts {
		cfgStmt := stmtI.(map[string]interface{})
		stmt := &IAMPolicyStatement{
			Effect: cfgStmt["effect"].(string),
		}

		if actions := cfgStmt["actions"].(*schema.Set).List(); len(actions) > 0 {
			stmt.Actions = iamPolicyDecodeConfigStringList(actions)
		}
		if actions := cfgStmt["not_actions"].(*schema.Set).List(); len(actions) > 0 {
			stmt.NotActions = iamPolicyDecodeConfigStringList(actions)
		}

		if resources := cfgStmt["resources"].(*schema.Set).List(); len(resources) > 0 {
			stmt.Resources = resourceAwsIamPolicyDocumentReplaceVarsInList(
				iamPolicyDecodeConfigStringList(resources),
			)
		}
		if resources := cfgStmt["not_resources"].(*schema.Set).List(); len(resources) > 0 {
			stmt.NotResources = resourceAwsIamPolicyDocumentReplaceVarsInList(
				iamPolicyDecodeConfigStringList(resources),
			)
		}

		if principals := cfgStmt["principals"].(*schema.Set).List(); len(principals) > 0 {
			stmt.Principals = resourceAwsIamPolicyDocumentMakePrincipals(principals)
		}

		if principals := cfgStmt["not_principals"].(*schema.Set).List(); len(principals) > 0 {
			stmt.NotPrincipals = resourceAwsIamPolicyDocumentMakePrincipals(principals)
		}

		if conditions := cfgStmt["condition"].(*schema.Set).List(); len(conditions) > 0 {
			stmt.Conditions = resourceAwsIamPolicyDocumentMakeConditions(conditions)
		}

		stmts[i] = stmt
	}

	jsonDoc, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		// should never happen if the above code is correct
		return err
	}

	d.Set("json", string(jsonDoc))
	if id != "" {
		d.SetId(id)
	} else {
		d.SetId("anon")
	}

	return nil
}

func resourceAwsIamPolicyDocumentReplaceVarsInList(in []string) []string {
	out := make([]string, len(in))
	for i, item := range in {
		out[i] = resourceAwsIamPolicyDocumentVarReplacer.Replace(item)
	}
	return out
}

func resourceAwsIamPolicyDocumentMakeConditions(in []interface{}) IAMPolicyStatementConditionSet {
	out := make([]IAMPolicyStatementCondition, len(in))
	for i, itemI := range in {
		item := itemI.(map[string]interface{})
		out[i] = IAMPolicyStatementCondition{
			Test:     item["test"].(string),
			Variable: item["variable"].(string),
			Values: resourceAwsIamPolicyDocumentReplaceVarsInList(
				iamPolicyDecodeConfigStringList(
					item["values"].(*schema.Set).List(),
				),
			),
		}
	}
	return IAMPolicyStatementConditionSet(out)
}

func resourceAwsIamPolicyDocumentMakePrincipals(in []interface{}) IAMPolicyStatementPrincipalSet {
	out := make([]IAMPolicyStatementPrincipal, len(in))
	for i, itemI := range in {
		item := itemI.(map[string]interface{})
		out[i] = IAMPolicyStatementPrincipal{
			Type: item["type"].(string),
			Identifiers: resourceAwsIamPolicyDocumentReplaceVarsInList(
				iamPolicyDecodeConfigStringList(
					item["identifiers"].(*schema.Set).List(),
				),
			),
		}
	}
	return IAMPolicyStatementPrincipalSet(out)
}

func resourceAwsIamPolicyDocumentDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func resourceAwsIamPolicyDocumentNoop(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIamPolicyPrincipalSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"identifiers": &schema.Schema{
					Type:     schema.TypeSet,
					Required: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
			},
		},
	}
}
