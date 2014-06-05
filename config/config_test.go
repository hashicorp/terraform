package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func TestConfigResourceGraph(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "resource_graph.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	graph := c.ResourceGraph()
	if err := graph.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(graph.String())
	expected := resourceGraphValue

	if actual != strings.TrimSpace(expected) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestNewResourceVariable(t *testing.T) {
	v, err := NewResourceVariable("foo.bar.baz")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if v.Type != "foo" {
		t.Fatalf("bad: %#v", v)
	}
	if v.Name != "bar" {
		t.Fatalf("bad: %#v", v)
	}
	if v.Field != "baz" {
		t.Fatalf("bad: %#v", v)
	}

	if v.FullKey() != "foo.bar.baz" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestNewUserVariable(t *testing.T) {
	v, err := NewUserVariable("var.bar")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if v.Name != "bar" {
		t.Fatalf("bad: %#v", v.Name)
	}
	if v.FullKey() != "var.bar" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestResourceReplaceVariables(t *testing.T) {
	r := &Resource{
		Name: "foo",
		Type: "bar",
		Config: map[string]interface{}{
			"foo": "${var.bar}",
		},
	}

	values := map[string]string{
		"var.bar": "value",
	}

	r2 := r.ReplaceVariables(values)

	expected := &Resource{
		Name: "foo",
		Type: "bar",
		Config: map[string]interface{}{
			"foo": "value",
		},
	}
	if !reflect.DeepEqual(r2, expected) {
		t.Fatalf("bad: %#v", r2)
	}

	/*
	TODO(mitchellh): Eventually, preserve original config...

	expectedOriginal := &Resource{
		Name: "foo",
		Type: "bar",
		Config: map[string]interface{}{
			"foo": "${var.bar}",
		},
	}

	if !reflect.DeepEqual(r, expectedOriginal) {
		t.Fatalf("bad: %#v", r)
	}
	*/
}

const resourceGraphValue = `
root: root
  root -> aws_security_group.firewall
  root -> aws_instance.web
  root -> aws_load_balancer.weblb
aws_security_group.firewall
aws_instance.web
  aws_instance.web -> aws_security_group.firewall
aws_load_balancer.weblb
  aws_load_balancer.weblb -> aws_instance.web
root
  root -> aws_security_group.firewall
  root -> aws_instance.web
  root -> aws_load_balancer.weblb
`
