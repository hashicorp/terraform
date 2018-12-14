package aws

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

var dataSourceAwsIamPolicyDocumentVarReplacer = strings.NewReplacer("&{", "${")

func dataSourceAwsIamPolicyDocument() *schema.Resource {
	setOfString := &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}

	return &schema.Resource{
		Read: dataSourceAwsIamPolicyDocumentRead,

		Schema: map[string]*schema.Schema{
			"override_json": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"policy_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"source_json": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"statement": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sid": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"effect": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "Allow",
							ValidateFunc: validation.StringInSlice([]string{"Allow", "Deny"}, false),
						},
						"actions":        setOfString,
						"not_actions":    setOfString,
						"resources":      setOfString,
						"not_resources":  setOfString,
						"principals":     dataSourceAwsIamPolicyPrincipalSchema(),
						"not_principals": dataSourceAwsIamPolicyPrincipalSchema(),
						"condition": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"test": {
										Type:     schema.TypeString,
										Required: true,
									},
									"variable": {
										Type:     schema.TypeString,
										Required: true,
									},
									"values": {
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
			"version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "2012-10-17",
				ValidateFunc: validation.StringInSlice([]string{
					"2008-10-17",
					"2012-10-17",
				}, false),
			},
			"json": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsIamPolicyDocumentRead(d *schema.ResourceData, meta interface{}) error {
	mergedDoc := &IAMPolicyDoc{}

	// populate mergedDoc directly with any source_json
	if sourceJSON, hasSourceJSON := d.GetOk("source_json"); hasSourceJSON {
		if err := json.Unmarshal([]byte(sourceJSON.(string)), mergedDoc); err != nil {
			return err
		}
	}

	// process the current document
	doc := &IAMPolicyDoc{
		Version: d.Get("version").(string),
	}

	if policyID, hasPolicyID := d.GetOk("policy_id"); hasPolicyID {
		doc.Id = policyID.(string)
	}

	if cfgStmts, hasCfgStmts := d.GetOk("statement"); hasCfgStmts {
		var cfgStmtIntf = cfgStmts.([]interface{})
		stmts := make([]*IAMPolicyStatement, len(cfgStmtIntf))
		sidMap := make(map[string]struct{})

		for i, stmtI := range cfgStmtIntf {
			cfgStmt := stmtI.(map[string]interface{})
			stmt := &IAMPolicyStatement{
				Effect: cfgStmt["effect"].(string),
			}

			if sid, ok := cfgStmt["sid"]; ok {
				if _, ok := sidMap[sid.(string)]; ok {
					return fmt.Errorf("Found duplicate sid (%s). Either remove the sid or ensure the sid is unique across all statements.", sid.(string))
				}
				stmt.Sid = sid.(string)
				if len(stmt.Sid) > 0 {
					sidMap[stmt.Sid] = struct{}{}
				}
			}

			if actions := cfgStmt["actions"].(*schema.Set).List(); len(actions) > 0 {
				stmt.Actions = iamPolicyDecodeConfigStringList(actions)
			}
			if actions := cfgStmt["not_actions"].(*schema.Set).List(); len(actions) > 0 {
				stmt.NotActions = iamPolicyDecodeConfigStringList(actions)
			}

			if resources := cfgStmt["resources"].(*schema.Set).List(); len(resources) > 0 {
				var err error
				stmt.Resources, err = dataSourceAwsIamPolicyDocumentReplaceVarsInList(
					iamPolicyDecodeConfigStringList(resources), doc.Version,
				)
				if err != nil {
					return fmt.Errorf("error reading resources: %s", err)
				}
			}
			if notResources := cfgStmt["not_resources"].(*schema.Set).List(); len(notResources) > 0 {
				var err error
				stmt.NotResources, err = dataSourceAwsIamPolicyDocumentReplaceVarsInList(
					iamPolicyDecodeConfigStringList(notResources), doc.Version,
				)
				if err != nil {
					return fmt.Errorf("error reading not_resources: %s", err)
				}
			}

			if principals := cfgStmt["principals"].(*schema.Set).List(); len(principals) > 0 {
				var err error
				stmt.Principals, err = dataSourceAwsIamPolicyDocumentMakePrincipals(principals, doc.Version)
				if err != nil {
					return fmt.Errorf("error reading principals: %s", err)
				}
			}

			if notPrincipals := cfgStmt["not_principals"].(*schema.Set).List(); len(notPrincipals) > 0 {
				var err error
				stmt.NotPrincipals, err = dataSourceAwsIamPolicyDocumentMakePrincipals(notPrincipals, doc.Version)
				if err != nil {
					return fmt.Errorf("error reading not_principals: %s", err)
				}
			}

			if conditions := cfgStmt["condition"].(*schema.Set).List(); len(conditions) > 0 {
				var err error
				stmt.Conditions, err = dataSourceAwsIamPolicyDocumentMakeConditions(conditions, doc.Version)
				if err != nil {
					return fmt.Errorf("error reading condition: %s", err)
				}
			}

			stmts[i] = stmt
		}

		doc.Statements = stmts

	}

	// merge our current document into mergedDoc
	mergedDoc.Merge(doc)

	// merge in override_json
	if overrideJSON, hasOverrideJSON := d.GetOk("override_json"); hasOverrideJSON {
		overrideDoc := &IAMPolicyDoc{}
		if err := json.Unmarshal([]byte(overrideJSON.(string)), overrideDoc); err != nil {
			return err
		}

		mergedDoc.Merge(overrideDoc)
	}

	jsonDoc, err := json.MarshalIndent(mergedDoc, "", "  ")
	if err != nil {
		// should never happen if the above code is correct
		return err
	}
	jsonString := string(jsonDoc)

	d.Set("json", jsonString)
	d.SetId(strconv.Itoa(hashcode.String(jsonString)))

	return nil
}

func dataSourceAwsIamPolicyDocumentReplaceVarsInList(in interface{}, version string) (interface{}, error) {
	switch v := in.(type) {
	case string:
		if version == "2008-10-17" && strings.Contains(v, "&{") {
			return nil, fmt.Errorf("found &{ sequence in (%s), which is not supported in document version 2008-10-17", v)
		}
		return dataSourceAwsIamPolicyDocumentVarReplacer.Replace(v), nil
	case []string:
		out := make([]string, len(v))
		for i, item := range v {
			if version == "2008-10-17" && strings.Contains(item, "&{") {
				return nil, fmt.Errorf("found &{ sequence in (%s), which is not supported in document version 2008-10-17", item)
			}
			out[i] = dataSourceAwsIamPolicyDocumentVarReplacer.Replace(item)
		}
		return out, nil
	default:
		panic("dataSourceAwsIamPolicyDocumentReplaceVarsInList: input not string nor []string")
	}
}

func dataSourceAwsIamPolicyDocumentMakeConditions(in []interface{}, version string) (IAMPolicyStatementConditionSet, error) {
	out := make([]IAMPolicyStatementCondition, len(in))
	for i, itemI := range in {
		var err error
		item := itemI.(map[string]interface{})
		out[i] = IAMPolicyStatementCondition{
			Test:     item["test"].(string),
			Variable: item["variable"].(string),
		}
		out[i].Values, err = dataSourceAwsIamPolicyDocumentReplaceVarsInList(
			iamPolicyDecodeConfigStringList(
				item["values"].(*schema.Set).List(),
			), version,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading values: %s", err)
		}
	}
	return IAMPolicyStatementConditionSet(out), nil
}

func dataSourceAwsIamPolicyDocumentMakePrincipals(in []interface{}, version string) (IAMPolicyStatementPrincipalSet, error) {
	out := make([]IAMPolicyStatementPrincipal, len(in))
	for i, itemI := range in {
		var err error
		item := itemI.(map[string]interface{})
		out[i] = IAMPolicyStatementPrincipal{
			Type: item["type"].(string),
		}
		out[i].Identifiers, err = dataSourceAwsIamPolicyDocumentReplaceVarsInList(
			iamPolicyDecodeConfigStringList(
				item["identifiers"].(*schema.Set).List(),
			), version,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading identifiers: %s", err)
		}
	}
	return IAMPolicyStatementPrincipalSet(out), nil
}

func dataSourceAwsIamPolicyPrincipalSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Type:     schema.TypeString,
					Required: true,
				},
				"identifiers": {
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
