package schema

import (
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceTimeout_ConfigDecode_badkey(t *testing.T) {
	r := &Resource{
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(10 * time.Minute),
			Update: DefaultTimeout(5 * time.Minute),
		},
	}

	//@TODO convert to test table
	raw, err := config.NewRawConfig(
		map[string]interface{}{
			"foo": "bar",
			"timeout": []map[string]interface{}{
				map[string]interface{}{
					"create": "2m",
				},
				map[string]interface{}{
					"delete": "1m",
				},
			},
		})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	c := terraform.NewResourceConfig(raw)

	timeout := &ResourceTimeout{}
	err = timeout.ConfigDecode(r, c)
	if err == nil {
		log.Println("Expected bad timeout key")
		t.Fatalf("err: %s", err)
	}
}

func TestResourceTimeout_ConfigDecode(t *testing.T) {
	r := &Resource{
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(10 * time.Minute),
			Update: DefaultTimeout(5 * time.Minute),
		},
	}

	raw, err := config.NewRawConfig(
		map[string]interface{}{
			"foo": "bar",
			"timeout": []map[string]interface{}{
				map[string]interface{}{
					"create": "2m",
				},
				map[string]interface{}{
					"update": "1m",
				},
			},
		})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	c := terraform.NewResourceConfig(raw)

	timeout := &ResourceTimeout{}
	err = timeout.ConfigDecode(r, c)
	if err != nil {
		log.Println("Expected good timeout returned")
		t.Fatalf("err: %s", err)
	}

	expected := &ResourceTimeout{
		Create: DefaultTimeout(2 * time.Minute),
		Update: DefaultTimeout(1 * time.Minute),
	}

	if !reflect.DeepEqual(timeout, expected) {
		t.Fatalf("bad timeout decode, expected (%#v), got (%#v)", expected, timeout)
	}
}

func TestResourceTimeout_DiffEncode_basic(t *testing.T) {
	cases := []struct {
		Timeout  *ResourceTimeout
		Expected map[string]interface{}
		// Not immediately clear when an error would hit
		ShouldErr bool
	}{
		// Two fields
		{
			Timeout:   timeoutForValues(10, 0, 5, 0, 0),
			Expected:  map[string]interface{}{TimeoutKey: expectedForValues(10, 0, 5, 0, 0)},
			ShouldErr: false,
		},
		// Two fields, one is Default
		{
			Timeout:   timeoutForValues(10, 0, 0, 0, 7),
			Expected:  map[string]interface{}{TimeoutKey: expectedForValues(10, 0, 0, 0, 7)},
			ShouldErr: false,
		},
		// All fields
		{
			Timeout:   timeoutForValues(10, 3, 4, 1, 7),
			Expected:  map[string]interface{}{TimeoutKey: expectedForValues(10, 3, 4, 1, 7)},
			ShouldErr: false,
		},
		// No fields
		{
			Timeout:   &ResourceTimeout{},
			Expected:  nil,
			ShouldErr: false,
		},
	}

	for _, c := range cases {
		state := &terraform.InstanceDiff{}
		err := c.Timeout.DiffEncode(state)
		if err != nil && !c.ShouldErr {
			t.Fatalf("Error, expected:\n%#v\n got:\n%#v\n", c.Expected, state.Meta)
		}

		// should maybe just compare [TimeoutKey] but for now we're assuming only
		// that in Meta
		if !reflect.DeepEqual(state.Meta, c.Expected) {
			t.Fatalf("Encode not equal, expected:\n%#v\n\ngot:\n%#v\n", c.Expected, state.Meta)
		}
	}
	// same test cases but for InstanceState
	for _, c := range cases {
		state := &terraform.InstanceState{}
		err := c.Timeout.StateEncode(state)
		if err != nil && !c.ShouldErr {
			t.Fatalf("Error, expected:\n%#v\n got:\n%#v\n", c.Expected, state.Meta)
		}

		// should maybe just compare [TimeoutKey] but for now we're assuming only
		// that in Meta
		if !reflect.DeepEqual(state.Meta, c.Expected) {
			t.Fatalf("Encode not equal, expected:\n%#v\n\ngot:\n%#v\n", c.Expected, state.Meta)
		}
	}
}

func TestResourceTimeout_MetaDecode_basic(t *testing.T) {
	cases := []struct {
		State    *terraform.InstanceDiff
		Expected *ResourceTimeout
		// Not immediately clear when an error would hit
		ShouldErr bool
	}{
		// Two fields
		{
			State:     &terraform.InstanceDiff{Meta: map[string]interface{}{TimeoutKey: expectedForValues(10, 0, 5, 0, 0)}},
			Expected:  timeoutForValues(10, 0, 5, 0, 0),
			ShouldErr: false,
		},
		// Two fields, one is Default
		{
			State:     &terraform.InstanceDiff{Meta: map[string]interface{}{TimeoutKey: expectedForValues(10, 0, 0, 0, 7)}},
			Expected:  timeoutForValues(10, 7, 7, 7, 7),
			ShouldErr: false,
		},
		// All fields
		{
			State:     &terraform.InstanceDiff{Meta: map[string]interface{}{TimeoutKey: expectedForValues(10, 3, 4, 1, 7)}},
			Expected:  timeoutForValues(10, 3, 4, 1, 7),
			ShouldErr: false,
		},
		// No fields
		{
			State:     &terraform.InstanceDiff{},
			Expected:  &ResourceTimeout{},
			ShouldErr: false,
		},
	}

	for _, c := range cases {
		rt := &ResourceTimeout{}
		err := rt.MetaDecode(c.State)
		if err != nil && !c.ShouldErr {
			t.Fatalf("Error, expected:\n%#v\n got:\n%#v\n", c.Expected, rt)
		}

		// should maybe just compare [TimeoutKey] but for now we're assuming only
		// that in Meta
		if !reflect.DeepEqual(rt, c.Expected) {
			t.Fatalf("Encode not equal, expected:\n%#v\n\ngot:\n%#v\n", c.Expected, rt)
		}
	}
}

func timeoutForValues(create, read, update, del, def int) *ResourceTimeout {
	rt := ResourceTimeout{}

	if create != 0 {
		rt.Create = DefaultTimeout(time.Duration(create) * time.Minute)
	}
	if read != 0 {
		rt.Read = DefaultTimeout(time.Duration(read) * time.Minute)
	}
	if update != 0 {
		rt.Update = DefaultTimeout(time.Duration(update) * time.Minute)
	}
	if del != 0 {
		rt.Delete = DefaultTimeout(time.Duration(del) * time.Minute)
	}

	if def != 0 {
		rt.Default = DefaultTimeout(time.Duration(def) * time.Minute)
	}

	return &rt
}

func expectedForValues(create, read, update, del, def int) map[string]interface{} {
	ex := make(map[string]interface{})

	if create != 0 {
		ex["create"] = DefaultTimeout(time.Duration(create) * time.Minute).Nanoseconds()
	}
	if read != 0 {
		ex["read"] = DefaultTimeout(time.Duration(read) * time.Minute).Nanoseconds()
	}
	if update != 0 {
		ex["update"] = DefaultTimeout(time.Duration(update) * time.Minute).Nanoseconds()
	}
	if del != 0 {
		ex["delete"] = DefaultTimeout(time.Duration(del) * time.Minute).Nanoseconds()
	}

	if def != 0 {
		defNano := DefaultTimeout(time.Duration(def) * time.Minute).Nanoseconds()
		ex["default"] = defNano

		for _, k := range timeKeys() {
			if _, ok := ex[k]; !ok {
				ex[k] = defNano
			}
		}
	}

	return ex
}
