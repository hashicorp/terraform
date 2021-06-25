package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/zclconf/go-cty/cty"
)

// The output is tested in greater detail in other tests; this suite focuses on
// details specific to the Resource function.
func TestAddResource(t *testing.T) {
	t.Run("config only", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		v := addHuman{view: NewView(streams), optional: true}
		err := v.Resource(
			mustResourceInstanceAddr("test_instance.foo"),
			addTestSchemaSensitive(configschema.NestingSingle),
			addrs.NewDefaultLocalProviderConfig("mytest"), cty.NilVal,
		)
		if err != nil {
			t.Fatal(err.Error())
		}

		expected := `resource "test_instance" "foo" {
  provider = mytest
  ami      = null      # OPTIONAL string
  disks = {            # OPTIONAL object
    mount_point = null # OPTIONAL string
    size        = null # OPTIONAL string
  }
  id = null            # OPTIONAL string
  root_block_device {  # OPTIONAL block
    volume_type = null # OPTIONAL string
  }
}
`
		output := done(t)
		if output.Stdout() != expected {
			t.Errorf("wrong result: %s", cmp.Diff(expected, output.Stdout()))
		}
	})

	t.Run("from state", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		v := addHuman{view: NewView(streams), optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"ami": cty.StringVal("ami-123456789"),
			"disks": cty.ObjectVal(map[string]cty.Value{
				"mount_point": cty.StringVal("/mnt/foo"),
				"size":        cty.StringVal("50GB"),
			}),
		})

		err := v.Resource(
			mustResourceInstanceAddr("test_instance.foo"),
			addTestSchemaSensitive(configschema.NestingSingle),
			addrs.NewDefaultLocalProviderConfig("mytest"), val,
		)
		if err != nil {
			t.Fatal(err.Error())
		}

		expected := `resource "test_instance" "foo" {
  provider = mytest
  ami      = "ami-123456789"
  disks    = {} # sensitive
  id       = null
}
`
		output := done(t)
		if output.Stdout() != expected {
			t.Errorf("wrong result: %s", cmp.Diff(expected, output.Stdout()))
		}
	})

}

func TestAdd_writeConfigAttributes(t *testing.T) {
	tests := map[string]struct {
		attrs    map[string]*configschema.Attribute
		expected string
	}{
		"empty returns nil": {
			map[string]*configschema.Attribute{},
			"",
		},
		"attributes": {
			map[string]*configschema.Attribute{
				"ami": {
					Type:     cty.Number,
					Required: true,
				},
				"boot_disk": {
					Type:     cty.String,
					Optional: true,
				},
				"password": {
					Type:      cty.String,
					Optional:  true,
					Sensitive: true, // sensitivity is ignored when printing blank templates
				},
			},
			`ami = null # REQUIRED number
boot_disk = null # OPTIONAL string
password = null # OPTIONAL string
`,
		},
		"attributes with nested types": {
			map[string]*configschema.Attribute{
				"ami": {
					Type:     cty.Number,
					Required: true,
				},
				"disks": {
					NestedType: &configschema.Object{
						Nesting: configschema.NestingSingle,
						Attributes: map[string]*configschema.Attribute{
							"size": {
								Type:     cty.Number,
								Optional: true,
							},
							"mount_point": {
								Type:     cty.String,
								Required: true,
							},
						},
					},
					Optional: true,
				},
			},
			`ami = null # REQUIRED number
disks = { # OPTIONAL object
  mount_point = null # REQUIRED string
  size = null # OPTIONAL number
}
`,
		},
	}

	v := addHuman{optional: true}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var buf strings.Builder
			if err := v.writeConfigAttributes(&buf, test.attrs, 0); err != nil {
				t.Errorf("unexpected error")
			}
			if buf.String() != test.expected {
				t.Errorf("wrong result: %s", cmp.Diff(test.expected, buf.String()))
			}
		})
	}
}

func TestAdd_writeConfigAttributesFromExisting(t *testing.T) {
	attrs := map[string]*configschema.Attribute{
		"ami": {
			Type:     cty.Number,
			Required: true,
		},
		"boot_disk": {
			Type:     cty.String,
			Optional: true,
		},
		"password": {
			Type:      cty.String,
			Optional:  true,
			Sensitive: true,
		},
		"disks": {
			NestedType: &configschema.Object{
				Nesting: configschema.NestingSingle,
				Attributes: map[string]*configschema.Attribute{
					"size": {
						Type:     cty.Number,
						Optional: true,
					},
					"mount_point": {
						Type:     cty.String,
						Required: true,
					},
				},
			},
			Optional: true,
		},
	}

	tests := map[string]struct {
		attrs    map[string]*configschema.Attribute
		val      cty.Value
		expected string
	}{
		"empty returns nil": {
			map[string]*configschema.Attribute{},
			cty.NilVal,
			"",
		},
		"mixed attributes": {
			attrs,
			cty.ObjectVal(map[string]cty.Value{
				"ami":       cty.NumberIntVal(123456),
				"boot_disk": cty.NullVal(cty.String),
				"password":  cty.StringVal("i am secret"),
				"disks": cty.ObjectVal(map[string]cty.Value{
					"size":        cty.NumberIntVal(50),
					"mount_point": cty.NullVal(cty.String),
				}),
			}),
			`ami = 123456
boot_disk = null
disks = {
  mount_point = null
  size = 50
}
password = null # sensitive
`,
		},
	}

	v := addHuman{optional: true}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var buf strings.Builder
			if err := v.writeConfigAttributesFromExisting(&buf, test.val, test.attrs, 0); err != nil {
				t.Errorf("unexpected error")
			}
			if buf.String() != test.expected {
				t.Errorf("wrong result: %s", cmp.Diff(test.expected, buf.String()))
			}
		})
	}
}

func TestAdd_writeConfigBlocks(t *testing.T) {
	t.Run("NestingSingle", func(t *testing.T) {
		v := addHuman{optional: true}
		schema := addTestSchema(configschema.NestingSingle)
		var buf strings.Builder
		v.writeConfigBlocks(&buf, schema.BlockTypes, 0)

		expected := `network_rules { # REQUIRED block
  ip_address = null # OPTIONAL string
}
root_block_device { # OPTIONAL block
  volume_type = null # OPTIONAL string
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Errorf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingList", func(t *testing.T) {
		v := addHuman{optional: true}
		schema := addTestSchema(configschema.NestingList)
		var buf strings.Builder
		v.writeConfigBlocks(&buf, schema.BlockTypes, 0)

		expected := `network_rules { # REQUIRED block
  ip_address = null # OPTIONAL string
}
root_block_device { # OPTIONAL block
  volume_type = null # OPTIONAL string
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingSet", func(t *testing.T) {
		v := addHuman{optional: true}
		schema := addTestSchema(configschema.NestingSet)
		var buf strings.Builder
		v.writeConfigBlocks(&buf, schema.BlockTypes, 0)

		expected := `network_rules { # REQUIRED block
  ip_address = null # OPTIONAL string
}
root_block_device { # OPTIONAL block
  volume_type = null # OPTIONAL string
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingMap", func(t *testing.T) {
		v := addHuman{optional: true}
		schema := addTestSchema(configschema.NestingMap)
		var buf strings.Builder
		v.writeConfigBlocks(&buf, schema.BlockTypes, 0)

		expected := `network_rules "key" { # REQUIRED block
  ip_address = null # OPTIONAL string
}
root_block_device "key" { # OPTIONAL block
  volume_type = null # OPTIONAL string
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})
}

func TestAdd_writeConfigBlocksFromExisting(t *testing.T) {

	t.Run("NestingSingle", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.ObjectVal(map[string]cty.Value{
				"volume_type": cty.StringVal("foo"),
			}),
		})
		schema := addTestSchema(configschema.NestingSingle)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device {
  volume_type = "foo"
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Errorf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingSingle_marked_attr", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.ObjectVal(map[string]cty.Value{
				"volume_type": cty.StringVal("foo").Mark(marks.Sensitive),
			}),
		})
		schema := addTestSchema(configschema.NestingSingle)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device {
  volume_type = null # sensitive
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Errorf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingSingle_entirely_marked", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.ObjectVal(map[string]cty.Value{
				"volume_type": cty.StringVal("foo"),
			}),
		}).Mark(marks.Sensitive)
		schema := addTestSchema(configschema.NestingSingle)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device {} # sensitive
`

		if !cmp.Equal(buf.String(), expected) {
			t.Errorf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingList", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("foo"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("bar"),
				}),
			}),
		})
		schema := addTestSchema(configschema.NestingList)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device {
  volume_type = "foo"
}
root_block_device {
  volume_type = "bar"
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingList_marked_attr", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("foo").Mark(marks.Sensitive),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("bar"),
				}),
			}),
		})
		schema := addTestSchema(configschema.NestingList)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device {
  volume_type = null # sensitive
}
root_block_device {
  volume_type = "bar"
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingList_entirely_marked", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("foo"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("bar"),
				}),
			}).Mark(marks.Sensitive),
		})
		schema := addTestSchema(configschema.NestingList)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device {} # sensitive
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingSet", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("foo"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("bar"),
				}),
			}),
		})
		schema := addTestSchema(configschema.NestingSet)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device {
  volume_type = "bar"
}
root_block_device {
  volume_type = "foo"
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingSet_marked", func(t *testing.T) {
		v := addHuman{optional: true}
		// In cty.Sets, the entire set ends up marked if any element is marked.
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("foo"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("bar"),
				}),
			}).Mark(marks.Sensitive),
		})
		schema := addTestSchema(configschema.NestingSet)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		// When the entire set of blocks is sensitive, we only print one block.
		expected := `root_block_device {} # sensitive
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingMap", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.MapVal(map[string]cty.Value{
				"1": cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("foo"),
				}),
				"2": cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("bar"),
				}),
			}),
		})
		schema := addTestSchema(configschema.NestingMap)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device "1" {
  volume_type = "foo"
}
root_block_device "2" {
  volume_type = "bar"
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingMap_marked", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.MapVal(map[string]cty.Value{
				"1": cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("foo").Mark(marks.Sensitive),
				}),
				"2": cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("bar"),
				}),
			}),
		})
		schema := addTestSchema(configschema.NestingMap)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device "1" {
  volume_type = null # sensitive
}
root_block_device "2" {
  volume_type = "bar"
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingMap_entirely_marked", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.MapVal(map[string]cty.Value{
				"1": cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("foo"),
				}),
				"2": cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("bar"),
				}),
			}).Mark(marks.Sensitive),
		})
		schema := addTestSchema(configschema.NestingMap)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device {} # sensitive
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingMap_marked_elem", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"root_block_device": cty.MapVal(map[string]cty.Value{
				"1": cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("foo"),
				}),
				"2": cty.ObjectVal(map[string]cty.Value{
					"volume_type": cty.StringVal("bar"),
				}).Mark(marks.Sensitive),
			}),
		})
		schema := addTestSchema(configschema.NestingMap)
		var buf strings.Builder
		v.writeConfigBlocksFromExisting(&buf, val, schema.BlockTypes, 0)

		expected := `root_block_device "1" {
  volume_type = "foo"
}
root_block_device "2" {} # sensitive
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})
}

func TestAdd_writeConfigNestedTypeAttribute(t *testing.T) {
	t.Run("NestingSingle", func(t *testing.T) {
		v := addHuman{optional: true}
		schema := addTestSchema(configschema.NestingSingle)
		var buf strings.Builder
		v.writeConfigNestedTypeAttribute(&buf, "disks", schema.Attributes["disks"], 0)

		expected := `disks = { # OPTIONAL object
  mount_point = null # OPTIONAL string
  size = null # OPTIONAL string
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingList", func(t *testing.T) {
		v := addHuman{optional: true}
		schema := addTestSchema(configschema.NestingList)
		var buf strings.Builder
		v.writeConfigNestedTypeAttribute(&buf, "disks", schema.Attributes["disks"], 0)

		expected := `disks = [{ # OPTIONAL list of object
  mount_point = null # OPTIONAL string
  size = null # OPTIONAL string
}]
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingSet", func(t *testing.T) {
		v := addHuman{optional: true}
		schema := addTestSchema(configschema.NestingSet)
		var buf strings.Builder
		v.writeConfigNestedTypeAttribute(&buf, "disks", schema.Attributes["disks"], 0)

		expected := `disks = [{ # OPTIONAL set of object
  mount_point = null # OPTIONAL string
  size = null # OPTIONAL string
}]
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingMap", func(t *testing.T) {
		v := addHuman{optional: true}
		schema := addTestSchema(configschema.NestingMap)
		var buf strings.Builder
		v.writeConfigNestedTypeAttribute(&buf, "disks", schema.Attributes["disks"], 0)

		expected := `disks = { # OPTIONAL map of object
  key = {
    mount_point = null # OPTIONAL string
    size = null # OPTIONAL string
  }
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})
}

func TestAdd_WriteConfigNestedTypeAttributeFromExisting(t *testing.T) {
	t.Run("NestingSingle", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"disks": cty.ObjectVal(map[string]cty.Value{
				"mount_point": cty.StringVal("/mnt/foo"),
				"size":        cty.StringVal("50GB"),
			}),
		})
		schema := addTestSchema(configschema.NestingSingle)
		var buf strings.Builder
		v.writeConfigNestedTypeAttributeFromExisting(&buf, "disks", schema.Attributes["disks"], val, 0)

		expected := `disks = {
  mount_point = "/mnt/foo"
  size = "50GB"
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingSingle_sensitive", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"disks": cty.ObjectVal(map[string]cty.Value{
				"mount_point": cty.StringVal("/mnt/foo"),
				"size":        cty.StringVal("50GB"),
			}),
		})
		schema := addTestSchemaSensitive(configschema.NestingSingle)
		var buf strings.Builder
		v.writeConfigNestedTypeAttributeFromExisting(&buf, "disks", schema.Attributes["disks"], val, 0)

		expected := `disks = {} # sensitive
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingList", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"disks": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/foo"),
					"size":        cty.StringVal("50GB"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/bar"),
					"size":        cty.StringVal("250GB"),
				}),
			}),
		})

		schema := addTestSchema(configschema.NestingList)
		var buf strings.Builder
		v.writeConfigNestedTypeAttributeFromExisting(&buf, "disks", schema.Attributes["disks"], val, 0)

		expected := `disks = [
  {
    mount_point = "/mnt/foo"
    size = "50GB"
  },
  {
    mount_point = "/mnt/bar"
    size = "250GB"
  },
]
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingList - marked", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"disks": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/foo"),
					"size":        cty.StringVal("50GB").Mark(marks.Sensitive),
				}),
				// This is an odd example, where the entire element is marked.
				cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/bar"),
					"size":        cty.StringVal("250GB"),
				}).Mark(marks.Sensitive),
			}),
		})

		schema := addTestSchema(configschema.NestingList)
		var buf strings.Builder
		v.writeConfigNestedTypeAttributeFromExisting(&buf, "disks", schema.Attributes["disks"], val, 0)

		expected := `disks = [
  {
    mount_point = "/mnt/foo"
    size = null # sensitive
  },
  {}, # sensitive
]
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingList - entirely marked", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"disks": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/foo"),
					"size":        cty.StringVal("50GB"),
				}),
				// This is an odd example, where the entire element is marked.
				cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/bar"),
					"size":        cty.StringVal("250GB"),
				}),
			}),
		}).Mark(marks.Sensitive)

		schema := addTestSchema(configschema.NestingList)
		var buf strings.Builder
		v.writeConfigNestedTypeAttributeFromExisting(&buf, "disks", schema.Attributes["disks"], val, 0)

		expected := `disks = [] # sensitive
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingMap", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"disks": cty.MapVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/foo"),
					"size":        cty.StringVal("50GB"),
				}),
				"bar": cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/bar"),
					"size":        cty.StringVal("250GB"),
				}),
			}),
		})
		schema := addTestSchema(configschema.NestingMap)
		var buf strings.Builder
		v.writeConfigNestedTypeAttributeFromExisting(&buf, "disks", schema.Attributes["disks"], val, 0)

		expected := `disks = {
  bar = {
    mount_point = "/mnt/bar"
    size = "250GB"
  }
  foo = {
    mount_point = "/mnt/foo"
    size = "50GB"
  }
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})

	t.Run("NestingMap - marked", func(t *testing.T) {
		v := addHuman{optional: true}
		val := cty.ObjectVal(map[string]cty.Value{
			"disks": cty.MapVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/foo"),
					"size":        cty.StringVal("50GB").Mark(marks.Sensitive),
				}),
				"bar": cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/mnt/bar"),
					"size":        cty.StringVal("250GB"),
				}).Mark(marks.Sensitive),
			}),
		})
		schema := addTestSchema(configschema.NestingMap)
		var buf strings.Builder
		v.writeConfigNestedTypeAttributeFromExisting(&buf, "disks", schema.Attributes["disks"], val, 0)

		expected := `disks = {
  bar = {} # sensitive
  foo = {
    mount_point = "/mnt/foo"
    size = null # sensitive
  }
}
`

		if !cmp.Equal(buf.String(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, buf.String()))
		}
	})
}

func addTestSchema(nesting configschema.NestingMode) *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {Type: cty.String, Optional: true, Computed: true},
			// Attributes which are neither optional nor required should not print.
			"uuid": {Type: cty.String, Computed: true},
			"ami":  {Type: cty.String, Optional: true},
			"disks": {
				NestedType: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"mount_point": {Type: cty.String, Optional: true},
						"size":        {Type: cty.String, Optional: true},
					},
					Nesting: nesting,
				},
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"root_block_device": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"volume_type": {
							Type:     cty.String,
							Optional: true,
							Computed: true,
						},
					},
				},
				Nesting: nesting,
			},
			"network_rules": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ip_address": {
							Type:     cty.String,
							Optional: true,
							Computed: true,
						},
					},
				},
				Nesting:  nesting,
				MinItems: 1,
			},
		},
	}
}

// addTestSchemaSensitive returns a schema with a sensitive NestedType and a
// NestedBlock with sensitive attributes.
func addTestSchemaSensitive(nesting configschema.NestingMode) *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {Type: cty.String, Optional: true, Computed: true},
			// Attributes which are neither optional nor required should not print.
			"uuid": {Type: cty.String, Computed: true},
			"ami":  {Type: cty.String, Optional: true},
			"disks": {
				NestedType: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"mount_point": {Type: cty.String, Optional: true},
						"size":        {Type: cty.String, Optional: true},
					},
					Nesting: nesting,
				},
				Sensitive: true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"root_block_device": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"volume_type": {
							Type:      cty.String,
							Optional:  true,
							Computed:  true,
							Sensitive: true,
						},
					},
				},
				Nesting: nesting,
			},
		},
	}
}

func mustResourceInstanceAddr(s string) addrs.AbsResourceInstance {
	addr, diags := addrs.ParseAbsResourceInstanceStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}
