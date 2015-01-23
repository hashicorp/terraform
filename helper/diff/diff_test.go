package diff

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/lang/ast"
	"github.com/hashicorp/terraform/terraform"
)

func testConfig(
	t *testing.T,
	c map[string]interface{},
	vs map[string]string) *terraform.ResourceConfig {
	rc, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(vs) > 0 {
		vars := make(map[string]ast.Variable)
		for k, v := range vs {
			vars[k] = ast.Variable{Value: v, Type: ast.TypeString}
		}

		if err := rc.Interpolate(vars); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	return terraform.NewResourceConfig(rc)
}

func testResourceDiffStr(rd *terraform.InstanceDiff) string {
	var buf bytes.Buffer

	crud := "UPDATE"
	if rd.RequiresNew() {
		crud = "CREATE"
	}

	buf.WriteString(fmt.Sprintf(
		"%s\n",
		crud))

	keyLen := 0
	keys := make([]string, 0, len(rd.Attributes))
	for key, _ := range rd.Attributes {
		keys = append(keys, key)
		if len(key) > keyLen {
			keyLen = len(key)
		}
	}
	sort.Strings(keys)

	for _, attrK := range keys {
		attrDiff := rd.Attributes[attrK]

		v := attrDiff.New
		if attrDiff.NewComputed {
			v = "<computed>"
		}
		if attrDiff.NewRemoved {
			v = "<removed>"
		}

		newResource := ""
		if attrDiff.RequiresNew {
			newResource = " (forces new resource)"
		}

		inOut := "IN "
		if attrDiff.Type == terraform.DiffAttrOutput {
			inOut = "OUT"
		}

		buf.WriteString(fmt.Sprintf(
			"  %s %s:%s %#v => %#v%s\n",
			inOut,
			attrK,
			strings.Repeat(" ", keyLen-len(attrK)),
			attrDiff.Old,
			v,
			newResource))
	}

	return buf.String()
}
