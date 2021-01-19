package stressgen

import (
	"bytes"
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestConfigFile(t *testing.T) {
	tests := map[string]struct {
		Input []ConfigObject
		Want  string
	}{
		// ConfigBoilerplate examples
		"boilerplate empty": {
			[]ConfigObject{
				&ConfigBoilerplate{},
			},
			`
				terraform {
					provider_requirements {
					}
				}
			`,
		},
		"boilerplate with providers": {
			[]ConfigObject{
				&ConfigBoilerplate{
					Providers: map[string]addrs.Provider{
						"a": addrs.MustParseProviderSourceString("terraform.io/stresstest/a"),
						"b": addrs.MustParseProviderSourceString("terraform.io/stresstest/b"),
					},
				},
			},
			`
				terraform {
					provider_requirements {
						a = {
							source = "terraform.io/stresstest/a"
						}
						b = {
							source = "terraform.io/stresstest/b"
						}
					}
				}
			`,
		},

		// ConfigVariable examples
		"variable empty": {
			[]ConfigObject{
				&ConfigVariable{
					Addr: addrs.InputVariable{Name: "blorp"},
				},
			},
			`
				variable "blorp" {
				}
			`,
		},
		"variable with type constraint string": {
			[]ConfigObject{
				&ConfigVariable{
					Addr:           addrs.InputVariable{Name: "blorp"},
					TypeConstraint: cty.String,
				},
			},
			`
				variable "blorp" {
					type = string
				}
			`,
		},
		"variable with type constraint number": {
			[]ConfigObject{
				&ConfigVariable{
					Addr:           addrs.InputVariable{Name: "blorp"},
					TypeConstraint: cty.Number,
				},
			},
			`
				variable "blorp" {
					type = number
				}
			`,
		},
		"variable with type constraint bool": {
			[]ConfigObject{
				&ConfigVariable{
					Addr:           addrs.InputVariable{Name: "blorp"},
					TypeConstraint: cty.Bool,
				},
			},
			`
				variable "blorp" {
					type = bool
				}
			`,
		},
		"variable with type constraint list(any)": {
			[]ConfigObject{
				&ConfigVariable{
					Addr:           addrs.InputVariable{Name: "blorp"},
					TypeConstraint: cty.List(cty.DynamicPseudoType),
				},
			},
			`
				variable "blorp" {
					type = list(any)
				}
			`,
		},
		"variable with type constraint map(string)": {
			[]ConfigObject{
				&ConfigVariable{
					Addr:           addrs.InputVariable{Name: "blorp"},
					TypeConstraint: cty.Map(cty.String),
				},
			},
			`
				variable "blorp" {
					type = map(string)
				}
			`,
		},
		"variable with type constraint set(object({}))": {
			[]ConfigObject{
				&ConfigVariable{
					Addr:           addrs.InputVariable{Name: "blorp"},
					TypeConstraint: cty.Set(cty.EmptyObject),
				},
			},
			`
				variable "blorp" {
					type = set(object({}))
				}
			`,
		},
		"variable with type constraint object({foo = string})": {
			[]ConfigObject{
				&ConfigVariable{
					Addr: addrs.InputVariable{Name: "blorp"},
					TypeConstraint: cty.Object(map[string]cty.Type{
						"foo": cty.String,
					}),
				},
			},
			`
				variable "blorp" {
					type = object({ "foo" = string })
				}
			`,
		},
		"variable with type constraint tuple([string, bool])": {
			[]ConfigObject{
				&ConfigVariable{
					Addr: addrs.InputVariable{Name: "blorp"},
					TypeConstraint: cty.Tuple([]cty.Type{
						cty.String,
						cty.Bool,
					}),
				},
			},
			`
				variable "blorp" {
					type = tuple([string, bool])
				}
			`,
		},
		"variable with default value": {
			[]ConfigObject{
				&ConfigVariable{
					Addr:         addrs.InputVariable{Name: "blorp"},
					DefaultValue: cty.StringVal("barge"),
				},
			},
			`
				variable "blorp" {
					default = "barge"
				}
			`,
		},
		"variable with type constraint and default value": {
			[]ConfigObject{
				&ConfigVariable{
					Addr:           addrs.InputVariable{Name: "blorp"},
					TypeConstraint: cty.String,
					DefaultValue:   cty.StringVal("barge"),
				},
			},
			`
				variable "blorp" {
					type    = string
					default = "barge"
				}
			`,
		},

		// ConfigOutput examples
		"output value any optional arguments": {
			[]ConfigObject{
				&ConfigOutput{
					Addr:  addrs.OutputValue{Name: "blorp"},
					Value: &ConfigExprConst{cty.NumberIntVal(15)},
				},
			},
			`
				output "blorp" {
					value = 15
				}
			`,
		},
		"sensitive output value": {
			[]ConfigObject{
				&ConfigOutput{
					Addr:      addrs.OutputValue{Name: "blorp"},
					Value:     &ConfigExprConst{cty.EmptyTupleVal},
					Sensitive: true,
				},
			},
			`
				output "blorp" {
					value     = []
					sensitive = true
				}
			`,
		},

		// Combinations and other misc
		"empty": {
			nil,
			``,
		},
		"variable and output value together": {
			[]ConfigObject{
				&ConfigVariable{
					Addr: addrs.InputVariable{Name: "blorp"},
				},
				&ConfigOutput{
					Addr:  addrs.OutputValue{Name: "blorp"},
					Value: &ConfigExprConst{cty.True},
				},
			},
			`
				variable "blorp" {
				}
				output "blorp" {
					value = true
				}
			`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := bytes.TrimSpace(ConfigFile(test.Input))
			want := bytes.TrimSpace(hclwrite.Format([]byte(test.Want)))

			if !bytes.Equal(got, want) {
				t.Errorf("wrong configuration\ngot:\n%s\n\nwant:\n%s", got, want)
			}
		})
	}
}
