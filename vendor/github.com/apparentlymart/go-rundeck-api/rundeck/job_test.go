package rundeck

import (
	"fmt"
	"testing"
)

func TestUnmarshalJobDetail(t *testing.T) {
	testUnmarshalXML(t, []unmarshalTest{
		unmarshalTest{
			"with-config",
			`<job><uuid>baz</uuid><dispatch><rankOrder>ascending</rankOrder></dispatch></job>`,
			&JobDetail{},
			func (rv interface {}) error {
				v := rv.(*JobDetail)
				if v.ID != "baz" {
					return fmt.Errorf("got ID %s, but expecting baz", v.ID)
				}
				if v.Dispatch.RankOrder != "ascending" {
					return fmt.Errorf("Dispatch.RankOrder = \"%v\", but expecting \"ascending\"", v.Dispatch.RankOrder)
				}
				return nil
			},
		},
		unmarshalTest{
			"with-empty-config",
			`<JobPlugin type="foo-plugin"><configuration/></JobPlugin>`,
			&JobPlugin{},
			func (rv interface {}) error {
				v := rv.(*JobPlugin)
				if v.Type != "foo-plugin" {
					return fmt.Errorf("got Type %s, but expecting foo-plugin", v.Type)
				}
				if len(v.Config) != 0 {
					return fmt.Errorf("got %i Config values, but expecting 0", len(v.Config))
				}
				return nil
			},
		},
	})
}

func TestMarshalJobPlugin(t *testing.T) {
	testMarshalXML(t, []marshalTest{
		marshalTest{
			"with-config",
			JobPlugin{
				Type: "foo-plugin",
				Config: map[string]string{
					"woo": "foo",
					"bar": "baz",
				},
			},
			`<JobPlugin type="foo-plugin"><configuration><entry key="bar" value="baz"></entry><entry key="woo" value="foo"></entry></configuration></JobPlugin>`,
		},
		marshalTest{
			"with-empty-config",
			JobPlugin{
				Type: "foo-plugin",
				Config: map[string]string{},
			},
			`<JobPlugin type="foo-plugin"></JobPlugin>`,
		},
		marshalTest{
			"with-zero-value-config",
			JobPlugin{
				Type: "foo-plugin",
			},
			`<JobPlugin type="foo-plugin"></JobPlugin>`,
		},
	})
}

func TestUnmarshalJobPlugin(t *testing.T) {
	testUnmarshalXML(t, []unmarshalTest{
		unmarshalTest{
			"with-config",
			`<JobPlugin type="foo-plugin"><configuration><entry key="woo" value="foo"/><entry key="bar" value="baz"/></configuration></JobPlugin>`,
			&JobPlugin{},
			func (rv interface {}) error {
				v := rv.(*JobPlugin)
				if v.Type != "foo-plugin" {
					return fmt.Errorf("got Type %s, but expecting foo-plugin", v.Type)
				}
				if len(v.Config) != 2 {
					return fmt.Errorf("got %v Config values, but expecting 2", len(v.Config))
				}
				if v.Config["woo"] != "foo" {
					return fmt.Errorf("Config[\"woo\"] = \"%s\", but expecting \"foo\"", v.Config["woo"])
				}
				if v.Config["bar"] != "baz" {
					return fmt.Errorf("Config[\"bar\"] = \"%s\", but expecting \"baz\"", v.Config["bar"])
				}
				return nil
			},
		},
		unmarshalTest{
			"with-empty-config",
			`<JobPlugin type="foo-plugin"><configuration/></JobPlugin>`,
			&JobPlugin{},
			func (rv interface {}) error {
				v := rv.(*JobPlugin)
				if v.Type != "foo-plugin" {
					return fmt.Errorf("got Type %s, but expecting foo-plugin", v.Type)
				}
				if len(v.Config) != 0 {
					return fmt.Errorf("got %i Config values, but expecting 0", len(v.Config))
				}
				return nil
			},
		},
	})
}
