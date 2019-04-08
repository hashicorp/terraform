package test

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestDiffApply_set(t *testing.T) {
	priorAttrs := map[string]string{
		"id":                                   "testID",
		"egress.#":                             "1",
		"egress.2129912301.cidr_blocks.#":      "1",
		"egress.2129912301.cidr_blocks.0":      "10.0.0.0/8",
		"egress.2129912301.description":        "Egress description",
		"egress.2129912301.from_port":          "80",
		"egress.2129912301.ipv6_cidr_blocks.#": "0",
		"egress.2129912301.prefix_list_ids.#":  "0",
		"egress.2129912301.protocol":           "tcp",
		"egress.2129912301.security_groups.#":  "0",
		"egress.2129912301.self":               "false",
		"egress.2129912301.to_port":            "8000",
	}

	diff := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"egress.2129912301.cidr_blocks.#":      {Old: "1", New: "0", NewComputed: false, NewRemoved: false},
			"egress.2129912301.cidr_blocks.0":      {Old: "10.0.0.0/8", New: "", NewComputed: false, NewRemoved: true},
			"egress.2129912301.description":        {Old: "Egress description", New: "", NewComputed: false, NewRemoved: true},
			"egress.2129912301.from_port":          {Old: "80", New: "0", NewComputed: false, NewRemoved: true},
			"egress.2129912301.ipv6_cidr_blocks.#": {Old: "0", New: "0", NewComputed: false, NewRemoved: false},
			"egress.2129912301.prefix_list_ids.#":  {Old: "0", New: "0", NewComputed: false, NewRemoved: false},
			"egress.2129912301.protocol":           {Old: "tcp", New: "", NewComputed: false, NewRemoved: true},
			"egress.2129912301.security_groups.#":  {Old: "0", New: "0", NewComputed: false, NewRemoved: false},
			"egress.2129912301.self":               {Old: "false", New: "false", NewComputed: false, NewRemoved: true},
			"egress.2129912301.to_port":            {Old: "8000", New: "0", NewComputed: false, NewRemoved: true},
			"egress.746197026.cidr_blocks.#":       {Old: "", New: "1", NewComputed: false, NewRemoved: false},
			"egress.746197026.cidr_blocks.0":       {Old: "", New: "10.0.0.0/8", NewComputed: false, NewRemoved: false},
			"egress.746197026.description":         {Old: "", New: "New egress description", NewComputed: false, NewRemoved: false},
			"egress.746197026.from_port":           {Old: "", New: "80", NewComputed: false, NewRemoved: false},
			"egress.746197026.ipv6_cidr_blocks.#":  {Old: "", New: "0", NewComputed: false, NewRemoved: false},
			"egress.746197026.prefix_list_ids.#":   {Old: "", New: "0", NewComputed: false, NewRemoved: false},
			"egress.746197026.protocol":            {Old: "", New: "tcp", NewComputed: false, NewRemoved: false, NewExtra: "tcp"},
			"egress.746197026.security_groups.#":   {Old: "", New: "0", NewComputed: false, NewRemoved: false},
			"egress.746197026.self":                {Old: "", New: "false", NewComputed: false, NewRemoved: false},
			"egress.746197026.to_port":             {Old: "", New: "8000", NewComputed: false, NewRemoved: false},
		},
	}

	resSchema := map[string]*schema.Schema{
		"egress": {
			Type:       schema.TypeSet,
			Optional:   true,
			Computed:   true,
			ConfigMode: schema.SchemaConfigModeAttr,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"from_port": {
						Type:     schema.TypeInt,
						Required: true,
					},

					"to_port": {
						Type:     schema.TypeInt,
						Required: true,
					},

					"protocol": {
						Type:     schema.TypeString,
						Required: true,
					},

					"cidr_blocks": {
						Type:     schema.TypeList,
						Optional: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},

					"ipv6_cidr_blocks": {
						Type:     schema.TypeList,
						Optional: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},

					"prefix_list_ids": {
						Type:     schema.TypeList,
						Optional: true,
						Elem:     &schema.Schema{Type: schema.TypeString},
					},

					"security_groups": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem:     &schema.Schema{Type: schema.TypeString},
						Set:      schema.HashString,
					},

					"self": {
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},

					"description": {
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
		},
	}

	expected := map[string]string{
		"egress.#":                       "1",
		"egress.746197026.cidr_blocks.#": "1",
		"egress.746197026.cidr_blocks.0": "10.0.0.0/8",
		"egress.746197026.description":   "New egress description",
		"egress.746197026.from_port":     "80", "egress.746197026.ipv6_cidr_blocks.#": "0",
		"egress.746197026.prefix_list_ids.#": "0",
		"egress.746197026.protocol":          "tcp",
		"egress.746197026.security_groups.#": "0",
		"egress.746197026.self":              "false",
		"egress.746197026.to_port":           "8000",
		"id":                                 "testID",
	}

	attrs, err := diff.Apply(priorAttrs, schema.LegacyResourceSchema(&schema.Resource{Schema: resSchema}).CoreConfigSchema())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(attrs, expected) {
		t.Fatalf("\nexpected: %#v\ngot: %#v\n", expected, attrs)
	}
}
