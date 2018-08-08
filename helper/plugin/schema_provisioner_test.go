package plugin

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/proto"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"

	mockproto "github.com/hashicorp/terraform/plugin/mock_proto"
)

// TestProvisioner functions in this file have been adapted from the
// helper/schema tests.

func noopApply(ctx context.Context) error {
	return nil
}

func TestProvisionerValidate(t *testing.T) {
	cases := []struct {
		Name   string
		P      *schema.Provisioner
		Config map[string]interface{}
		Err    bool
		Warns  []string
	}{
		{
			Name:   "No ApplyFunc",
			P:      &schema.Provisioner{},
			Config: map[string]interface{}{},
			Err:    true,
		},
		{
			"Basic required field set",
			&schema.Provisioner{
				Schema: map[string]*schema.Schema{
					"foo": &schema.Schema{
						Required: true,
						Type:     schema.TypeString,
					},
				},
				ApplyFunc: noopApply,
			},
			map[string]interface{}{
				"foo": "bar",
			},
			false,
			nil,
		},
		{
			Name: "Warning from property validation",
			P: &schema.Provisioner{
				Schema: map[string]*schema.Schema{
					"foo": {
						Type:     schema.TypeString,
						Optional: true,
						ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
							ws = append(ws, "Simple warning from property validation")
							return
						},
					},
				},
				ApplyFunc: noopApply,
			},
			Config: map[string]interface{}{
				"foo": "",
			},
			Err:   false,
			Warns: []string{"Simple warning from property validation"},
		},
		{
			Name: "No schema",
			P: &schema.Provisioner{
				Schema:    nil,
				ApplyFunc: noopApply,
			},
			Config: map[string]interface{}{},
			Err:    false,
		},
		{
			Name: "Warning from provisioner ValidateFunc",
			P: &schema.Provisioner{
				Schema:    nil,
				ApplyFunc: noopApply,
				ValidateFunc: func(*terraform.ResourceConfig) (ws []string, errors []error) {
					ws = append(ws, "Simple warning from provisioner ValidateFunc")
					return
				},
			},
			Config: map[string]interface{}{},
			Err:    false,
			Warns:  []string{"Simple warning from provisioner ValidateFunc"},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			p := &GRPCProvisionerServer{
				provisioner: tc.P,
			}

			cfgSchema := schema.InternalMap(tc.P.Schema).CoreConfigSchema()
			val := hcl2shim.HCL2ValueFromConfigValue(tc.Config)

			val, err := cfgSchema.CoerceValue(val)
			if err != nil {
				t.Fatal(err)
			}

			mp, err := msgpack.Marshal(val, cfgSchema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			req := &proto.ValidateProvisionerConfig_Request{
				Config: &proto.DynamicValue{Msgpack: mp},
			}

			resp, err := p.ValidateProvisionerConfig(nil, req)
			if err != nil {
				t.Fatal(err)
			}

			diags := plugin.ProtoToDiagnostics(resp.Diagnostics)

			if diags.HasErrors() != tc.Err {
				t.Fatal(diags.Err())
			}

			var ws []string
			for _, d := range diags {
				if d.Severity() == tfdiags.Warning {
					ws = append(ws, d.Description().Summary)
				}
			}

			if (tc.Warns != nil || len(ws) != 0) && !reflect.DeepEqual(ws, tc.Warns) {
				t.Fatalf("%d: warnings mismatch, actual: %#v", i, ws)
			}
		})
	}
}

func TestProvisionerApply(t *testing.T) {
	cases := []struct {
		Name   string
		P      *schema.Provisioner
		Conn   map[string]interface{}
		Config map[string]interface{}
		Err    bool
	}{
		{
			Name: "Basic config",
			P: &schema.Provisioner{
				ConnSchema: map[string]*schema.Schema{
					"foo": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
				},

				Schema: map[string]*schema.Schema{
					"foo": &schema.Schema{
						Type:     schema.TypeInt,
						Optional: true,
					},
				},

				ApplyFunc: func(ctx context.Context) error {
					cd := ctx.Value(schema.ProvConnDataKey).(*schema.ResourceData)
					d := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)
					if d.Get("foo").(int) != 42 {
						return fmt.Errorf("bad config data")
					}
					if cd.Get("foo").(string) != "bar" {
						return fmt.Errorf("bad conn data")
					}

					return nil
				},
			},
			Conn: map[string]interface{}{
				"foo": "bar",
			},
			Config: map[string]interface{}{
				"foo": 42,
			},
			Err: false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			p := &GRPCProvisionerServer{
				provisioner: tc.P,
			}

			cfgSchema := schema.InternalMap(tc.P.Schema).CoreConfigSchema()
			val := hcl2shim.HCL2ValueFromConfigValue(tc.Config)

			val, err := cfgSchema.CoerceValue(val)
			if err != nil {
				t.Fatal(err)
			}

			cfgMP, err := msgpack.Marshal(val, cfgSchema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			connVal := hcl2shim.HCL2ValueFromConfigValue(tc.Conn)

			connMP, err := msgpack.Marshal(connVal, cty.Map(cty.String))
			if err != nil {
				t.Fatal(err)
			}

			req := &proto.ProvisionResource_Request{
				Config:     &proto.DynamicValue{Msgpack: cfgMP},
				Connection: &proto.DynamicValue{Msgpack: connMP},
			}

			ctrl := gomock.NewController(t)
			srv := mockproto.NewMockProvisioner_ProvisionResourceServer(ctrl)
			srv.EXPECT().Send(gomock.Any()).Return(nil)

			err = p.ProvisionResource(req, srv)
			if err != nil && !tc.Err {
				t.Fatal(err)
			}
		})
	}
}

func TestProvisionerStop(t *testing.T) {
	p := &GRPCProvisionerServer{
		provisioner: &schema.Provisioner{},
	}

	// Verify stopch blocks
	ch := p.provisioner.StopContext().Done()
	select {
	case <-ch:
		t.Fatal("should not be stopped")
	case <-time.After(10 * time.Millisecond):
	}

	// Stop it
	resp, err := p.Stop(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Error != "" {
		t.Fatal(resp.Error)
	}

	select {
	case <-ch:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("should be stopped")
	}
}

func TestProvisionerStop_apply(t *testing.T) {
	p := &schema.Provisioner{
		ConnSchema: map[string]*schema.Schema{
			"foo": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},

		Schema: map[string]*schema.Schema{
			"foo": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
		},

		ApplyFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	}

	s := &GRPCProvisionerServer{
		provisioner: p,
	}
	srv := mockproto.NewMockProvisioner_ProvisionResourceServer(gomock.NewController(t))
	srv.EXPECT().Send(gomock.Any()).Return(nil)

	// Run the apply in a goroutine
	doneCh := make(chan struct{})
	go func() {
		req := &proto.ProvisionResource_Request{
			Config:     &proto.DynamicValue{Msgpack: []byte("\201\243foo*")},
			Connection: &proto.DynamicValue{Msgpack: []byte("\201\243foo\243bar")},
		}
		err := s.ProvisionResource(req, srv)
		if err != nil {
			t.Fatal(err)
		}
		close(doneCh)
	}()

	// Should block
	select {
	case <-doneCh:
		t.Fatal("should not be done")
	case <-time.After(10 * time.Millisecond):
	}

	resp, err := s.Stop(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error != "" {
		t.Fatal(resp.Error)
	}

	select {
	case <-doneCh:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("should be done")
	}
}
