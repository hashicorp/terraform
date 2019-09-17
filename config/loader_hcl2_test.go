package config

import (
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"

	gohcl2 "github.com/hashicorp/hcl2/gohcl"
	hcl2 "github.com/hashicorp/hcl2/hcl"
)

func TestHCL2ConfigurableConfigurable(t *testing.T) {
	var _ configurable = new(hcl2Configurable)
}

func TestHCL2Basic(t *testing.T) {
	loader := globalHCL2Loader
	cbl, _, err := loader.loadFile("testdata/basic-hcl2.tf")
	if err != nil {
		if diags, isDiags := err.(hcl2.Diagnostics); isDiags {
			for _, diag := range diags {
				t.Logf("- %s", diag.Error())
			}
			t.Fatalf("unexpected diagnostics in load")
		} else {
			t.Fatalf("unexpected error in load: %s", err)
		}
	}

	cfg, err := cbl.Config()
	if err != nil {
		if diags, isDiags := err.(hcl2.Diagnostics); isDiags {
			for _, diag := range diags {
				t.Logf("- %s", diag.Error())
			}
			t.Fatalf("unexpected diagnostics in decode")
		} else {
			t.Fatalf("unexpected error in decode: %s", err)
		}
	}

	// Unfortunately the config structure isn't DeepEqual-friendly because
	// of all the nested RawConfig, etc structures, so we'll need to
	// hand-assert each item.

	// The "terraform" block
	if cfg.Terraform == nil {
		t.Fatalf("Terraform field is nil")
	}
	if got, want := cfg.Terraform.RequiredVersion, "foo"; got != want {
		t.Errorf("wrong Terraform.RequiredVersion %q; want %q", got, want)
	}
	if cfg.Terraform.Backend == nil {
		t.Fatalf("Terraform.Backend is nil")
	}
	if got, want := cfg.Terraform.Backend.Type, "baz"; got != want {
		t.Errorf("wrong Terraform.Backend.Type %q; want %q", got, want)
	}
	if got, want := cfg.Terraform.Backend.RawConfig.Raw, map[string]interface{}{"something": "nothing"}; !reflect.DeepEqual(got, want) {
		t.Errorf("wrong Terraform.Backend.RawConfig.Raw %#v; want %#v", got, want)
	}

	// The "atlas" block
	if cfg.Atlas == nil {
		t.Fatalf("Atlas field is nil")
	}
	if got, want := cfg.Atlas.Name, "example/foo"; got != want {
		t.Errorf("wrong Atlas.Name %q; want %q", got, want)
	}

	// "module" blocks
	if got, want := len(cfg.Modules), 1; got != want {
		t.Errorf("Modules slice has wrong length %#v; want %#v", got, want)
	} else {
		m := cfg.Modules[0]
		if got, want := m.Name, "child"; got != want {
			t.Errorf("wrong Modules[0].Name %#v; want %#v", got, want)
		}
		if got, want := m.Source, "./baz"; got != want {
			t.Errorf("wrong Modules[0].Source %#v; want %#v", got, want)
		}
		want := map[string]string{"toasty": "true"}
		var got map[string]string
		gohcl2.DecodeBody(m.RawConfig.Body, nil, &got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("wrong Modules[0].RawConfig.Body %#v; want %#v", got, want)
		}
	}

	// "resource" blocks
	if got, want := len(cfg.Resources), 5; got != want {
		t.Errorf("Resources slice has wrong length %#v; want %#v", got, want)
	} else {
		{
			r := cfg.Resources[0]

			if got, want := r.Id(), "aws_security_group.firewall"; got != want {
				t.Errorf("wrong Resources[0].Id() %#v; want %#v", got, want)
			}

			wantConfig := map[string]string{}
			var gotConfig map[string]string
			gohcl2.DecodeBody(r.RawConfig.Body, nil, &gotConfig)
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Errorf("wrong Resources[0].RawConfig.Body %#v; want %#v", gotConfig, wantConfig)
			}

			wantCount := map[string]string{"count": "5"}
			var gotCount map[string]string
			gohcl2.DecodeBody(r.RawCount.Body, nil, &gotCount)
			if !reflect.DeepEqual(gotCount, wantCount) {
				t.Errorf("wrong Resources[0].RawCount.Body %#v; want %#v", gotCount, wantCount)
			}
			if got, want := r.RawCount.Key, "count"; got != want {
				t.Errorf("wrong Resources[0].RawCount.Key %#v; want %#v", got, want)
			}

			if got, want := len(r.Provisioners), 0; got != want {
				t.Errorf("wrong Resources[0].Provisioners length %#v; want %#v", got, want)
			}
			if got, want := len(r.DependsOn), 0; got != want {
				t.Errorf("wrong Resources[0].DependsOn length %#v; want %#v", got, want)
			}
			if got, want := r.Provider, "another"; got != want {
				t.Errorf("wrong Resources[0].Provider %#v; want %#v", got, want)
			}
			if got, want := r.Lifecycle, (ResourceLifecycle{}); !reflect.DeepEqual(got, want) {
				t.Errorf("wrong Resources[0].Lifecycle %#v; want %#v", got, want)
			}
		}
		{
			r := cfg.Resources[1]

			if got, want := r.Id(), "aws_instance.web"; got != want {
				t.Errorf("wrong Resources[1].Id() %#v; want %#v", got, want)
			}
			if got, want := r.Provider, ""; got != want {
				t.Errorf("wrong Resources[1].Provider %#v; want %#v", got, want)
			}

			if got, want := len(r.Provisioners), 1; got != want {
				t.Errorf("wrong Resources[1].Provisioners length %#v; want %#v", got, want)
			} else {
				p := r.Provisioners[0]

				if got, want := p.Type, "file"; got != want {
					t.Errorf("wrong Resources[1].Provisioners[0].Type %#v; want %#v", got, want)
				}

				wantConfig := map[string]string{
					"source":      "foo",
					"destination": "bar",
				}
				var gotConfig map[string]string
				gohcl2.DecodeBody(p.RawConfig.Body, nil, &gotConfig)
				if !reflect.DeepEqual(gotConfig, wantConfig) {
					t.Errorf("wrong Resources[1].Provisioners[0].RawConfig.Body %#v; want %#v", gotConfig, wantConfig)
				}

				wantConn := map[string]string{
					"default": "true",
				}
				var gotConn map[string]string
				gohcl2.DecodeBody(p.ConnInfo.Body, nil, &gotConn)
				if !reflect.DeepEqual(gotConn, wantConn) {
					t.Errorf("wrong Resources[1].Provisioners[0].ConnInfo.Body %#v; want %#v", gotConn, wantConn)
				}
			}

			// We'll use these throwaway structs to more easily decode and
			// compare the main config body.
			type instanceNetworkInterface struct {
				DeviceIndex int    `hcl:"device_index"`
				Description string `hcl:"description"`
			}
			type instanceConfig struct {
				AMI              string                   `hcl:"ami"`
				SecurityGroups   []string                 `hcl:"security_groups"`
				NetworkInterface instanceNetworkInterface `hcl:"network_interface,block"`
			}
			var gotConfig instanceConfig
			wantConfig := instanceConfig{
				AMI:            "ami-abc123",
				SecurityGroups: []string{"foo", "sg-firewall"},
				NetworkInterface: instanceNetworkInterface{
					DeviceIndex: 0,
					Description: "Main network interface",
				},
			}
			ctx := &hcl2.EvalContext{
				Variables: map[string]cty.Value{
					"var": cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("ami-abc123"),
					}),
					"aws_security_group": cty.ObjectVal(map[string]cty.Value{
						"firewall": cty.ObjectVal(map[string]cty.Value{
							"foo": cty.StringVal("sg-firewall"),
						}),
					}),
				},
			}
			diags := gohcl2.DecodeBody(r.RawConfig.Body, ctx, &gotConfig)
			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics decoding Resources[1].RawConfig.Body")
				for _, diag := range diags {
					t.Logf("- %s", diag.Error())
				}
			}
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Errorf("wrong Resources[1].RawConfig.Body %#v; want %#v", gotConfig, wantConfig)
			}

		}
		{
			r := cfg.Resources[2]

			if got, want := r.Id(), "aws_instance.db"; got != want {
				t.Errorf("wrong Resources[2].Id() %#v; want %#v", got, want)
			}
			if got, want := r.DependsOn, []string{"aws_instance.web"}; !reflect.DeepEqual(got, want) {
				t.Errorf("wrong Resources[2].DependsOn %#v; want %#v", got, want)
			}

			if got, want := len(r.Provisioners), 1; got != want {
				t.Errorf("wrong Resources[2].Provisioners length %#v; want %#v", got, want)
			} else {
				p := r.Provisioners[0]

				if got, want := p.Type, "file"; got != want {
					t.Errorf("wrong Resources[2].Provisioners[0].Type %#v; want %#v", got, want)
				}

				wantConfig := map[string]string{
					"source":      "here",
					"destination": "there",
				}
				var gotConfig map[string]string
				gohcl2.DecodeBody(p.RawConfig.Body, nil, &gotConfig)
				if !reflect.DeepEqual(gotConfig, wantConfig) {
					t.Errorf("wrong Resources[2].Provisioners[0].RawConfig.Body %#v; want %#v", gotConfig, wantConfig)
				}

				wantConn := map[string]string{
					"default": "false",
				}
				var gotConn map[string]string
				gohcl2.DecodeBody(p.ConnInfo.Body, nil, &gotConn)
				if !reflect.DeepEqual(gotConn, wantConn) {
					t.Errorf("wrong Resources[2].Provisioners[0].ConnInfo.Body %#v; want %#v", gotConn, wantConn)
				}
			}
		}
		{
			r := cfg.Resources[3]

			if got, want := r.Id(), "data.do.simple"; got != want {
				t.Errorf("wrong Resources[3].Id() %#v; want %#v", got, want)
			}
			if got, want := r.DependsOn, []string(nil); !reflect.DeepEqual(got, want) {
				t.Errorf("wrong Resources[3].DependsOn %#v; want %#v", got, want)
			}
			if got, want := r.Provider, "do.foo"; got != want {
				t.Errorf("wrong Resources[3].Provider %#v; want %#v", got, want)
			}

			wantConfig := map[string]string{
				"foo": "baz",
			}
			var gotConfig map[string]string
			gohcl2.DecodeBody(r.RawConfig.Body, nil, &gotConfig)
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Errorf("wrong Resources[3].RawConfig.Body %#v; want %#v", gotConfig, wantConfig)
			}
		}
		{
			r := cfg.Resources[4]

			if got, want := r.Id(), "data.do.depends"; got != want {
				t.Errorf("wrong Resources[4].Id() %#v; want %#v", got, want)
			}
			if got, want := r.DependsOn, []string{"data.do.simple"}; !reflect.DeepEqual(got, want) {
				t.Errorf("wrong Resources[4].DependsOn %#v; want %#v", got, want)
			}
			if got, want := r.Provider, ""; got != want {
				t.Errorf("wrong Resources[4].Provider %#v; want %#v", got, want)
			}

			wantConfig := map[string]string{}
			var gotConfig map[string]string
			gohcl2.DecodeBody(r.RawConfig.Body, nil, &gotConfig)
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Errorf("wrong Resources[4].RawConfig.Body %#v; want %#v", gotConfig, wantConfig)
			}
		}
	}

	// "variable" blocks
	if got, want := len(cfg.Variables), 3; got != want {
		t.Errorf("Variables slice has wrong length %#v; want %#v", got, want)
	} else {
		{
			v := cfg.Variables[0]

			if got, want := v.Name, "foo"; got != want {
				t.Errorf("wrong Variables[0].Name %#v; want %#v", got, want)
			}
			if got, want := v.Default, "bar"; got != want {
				t.Errorf("wrong Variables[0].Default %#v; want %#v", got, want)
			}
			if got, want := v.Description, "barbar"; got != want {
				t.Errorf("wrong Variables[0].Description %#v; want %#v", got, want)
			}
			if got, want := v.DeclaredType, ""; got != want {
				t.Errorf("wrong Variables[0].DeclaredType %#v; want %#v", got, want)
			}
		}
		{
			v := cfg.Variables[1]

			if got, want := v.Name, "bar"; got != want {
				t.Errorf("wrong Variables[1].Name %#v; want %#v", got, want)
			}
			if got, want := v.Default, interface{}(nil); got != want {
				t.Errorf("wrong Variables[1].Default %#v; want %#v", got, want)
			}
			if got, want := v.Description, ""; got != want {
				t.Errorf("wrong Variables[1].Description %#v; want %#v", got, want)
			}
			if got, want := v.DeclaredType, "string"; got != want {
				t.Errorf("wrong Variables[1].DeclaredType %#v; want %#v", got, want)
			}
		}
		{
			v := cfg.Variables[2]

			if got, want := v.Name, "baz"; got != want {
				t.Errorf("wrong Variables[2].Name %#v; want %#v", got, want)
			}
			if got, want := v.Default, map[string]interface{}{"key": "value"}; !reflect.DeepEqual(got, want) {
				t.Errorf("wrong Variables[2].Default %#v; want %#v", got, want)
			}
			if got, want := v.Description, ""; got != want {
				t.Errorf("wrong Variables[2].Description %#v; want %#v", got, want)
			}
			if got, want := v.DeclaredType, "map"; got != want {
				t.Errorf("wrong Variables[2].DeclaredType %#v; want %#v", got, want)
			}
		}
	}

	// "output" blocks
	if got, want := len(cfg.Outputs), 2; got != want {
		t.Errorf("Outputs slice has wrong length %#v; want %#v", got, want)
	} else {
		{
			o := cfg.Outputs[0]

			if got, want := o.Name, "web_ip"; got != want {
				t.Errorf("wrong Outputs[0].Name %#v; want %#v", got, want)
			}
			if got, want := o.DependsOn, []string(nil); !reflect.DeepEqual(got, want) {
				t.Errorf("wrong Outputs[0].DependsOn %#v; want %#v", got, want)
			}
			if got, want := o.Description, ""; got != want {
				t.Errorf("wrong Outputs[0].Description %#v; want %#v", got, want)
			}
			if got, want := o.Sensitive, true; got != want {
				t.Errorf("wrong Outputs[0].Sensitive %#v; want %#v", got, want)
			}

			wantConfig := map[string]string{
				"value": "312.213.645.123",
			}
			var gotConfig map[string]string
			ctx := &hcl2.EvalContext{
				Variables: map[string]cty.Value{
					"aws_instance": cty.ObjectVal(map[string]cty.Value{
						"web": cty.ObjectVal(map[string]cty.Value{
							"private_ip": cty.StringVal("312.213.645.123"),
						}),
					}),
				},
			}
			gohcl2.DecodeBody(o.RawConfig.Body, ctx, &gotConfig)
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Errorf("wrong Outputs[0].RawConfig.Body %#v; want %#v", gotConfig, wantConfig)
			}
		}
		{
			o := cfg.Outputs[1]

			if got, want := o.Name, "web_id"; got != want {
				t.Errorf("wrong Outputs[1].Name %#v; want %#v", got, want)
			}
			if got, want := o.DependsOn, []string{"aws_instance.db"}; !reflect.DeepEqual(got, want) {
				t.Errorf("wrong Outputs[1].DependsOn %#v; want %#v", got, want)
			}
			if got, want := o.Description, "The ID"; got != want {
				t.Errorf("wrong Outputs[1].Description %#v; want %#v", got, want)
			}
			if got, want := o.Sensitive, false; got != want {
				t.Errorf("wrong Outputs[1].Sensitive %#v; want %#v", got, want)
			}
		}
	}

	// "provider" blocks
	if got, want := len(cfg.ProviderConfigs), 2; got != want {
		t.Errorf("ProviderConfigs slice has wrong length %#v; want %#v", got, want)
	} else {
		{
			p := cfg.ProviderConfigs[0]

			if got, want := p.Name, "aws"; got != want {
				t.Errorf("wrong ProviderConfigs[0].Name %#v; want %#v", got, want)
			}
			if got, want := p.Alias, ""; got != want {
				t.Errorf("wrong ProviderConfigs[0].Alias %#v; want %#v", got, want)
			}
			if got, want := p.Version, "1.0.0"; got != want {
				t.Errorf("wrong ProviderConfigs[0].Version %#v; want %#v", got, want)
			}

			wantConfig := map[string]string{
				"access_key": "foo",
				"secret_key": "bar",
			}
			var gotConfig map[string]string
			gohcl2.DecodeBody(p.RawConfig.Body, nil, &gotConfig)
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Errorf("wrong ProviderConfigs[0].RawConfig.Body %#v; want %#v", gotConfig, wantConfig)
			}

		}
		{
			p := cfg.ProviderConfigs[1]

			if got, want := p.Name, "do"; got != want {
				t.Errorf("wrong ProviderConfigs[1].Name %#v; want %#v", got, want)
			}
			if got, want := p.Alias, "fum"; got != want {
				t.Errorf("wrong ProviderConfigs[1].Alias %#v; want %#v", got, want)
			}
			if got, want := p.Version, ""; got != want {
				t.Errorf("wrong ProviderConfigs[1].Version %#v; want %#v", got, want)
			}

		}
	}

	// "locals" definitions
	if got, want := len(cfg.Locals), 5; got != want {
		t.Errorf("Locals slice has wrong length %#v; want %#v", got, want)
	} else {
		{
			l := cfg.Locals[0]

			if got, want := l.Name, "security_group_ids"; got != want {
				t.Errorf("wrong Locals[0].Name %#v; want %#v", got, want)
			}

			wantConfig := map[string][]string{
				"value": []string{"sg-abc123"},
			}
			var gotConfig map[string][]string
			ctx := &hcl2.EvalContext{
				Variables: map[string]cty.Value{
					"aws_security_group": cty.ObjectVal(map[string]cty.Value{
						"firewall": cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("sg-abc123"),
						}),
					}),
				},
			}
			gohcl2.DecodeBody(l.RawConfig.Body, ctx, &gotConfig)
			if !reflect.DeepEqual(gotConfig, wantConfig) {
				t.Errorf("wrong Locals[0].RawConfig.Body %#v; want %#v", gotConfig, wantConfig)
			}
		}
		{
			l := cfg.Locals[1]

			if got, want := l.Name, "web_ip"; got != want {
				t.Errorf("wrong Locals[1].Name %#v; want %#v", got, want)
			}
		}
		{
			l := cfg.Locals[2]

			if got, want := l.Name, "literal"; got != want {
				t.Errorf("wrong Locals[2].Name %#v; want %#v", got, want)
			}
		}
		{
			l := cfg.Locals[3]

			if got, want := l.Name, "literal_list"; got != want {
				t.Errorf("wrong Locals[3].Name %#v; want %#v", got, want)
			}
		}
		{
			l := cfg.Locals[4]

			if got, want := l.Name, "literal_map"; got != want {
				t.Errorf("wrong Locals[4].Name %#v; want %#v", got, want)
			}
		}
	}
}
