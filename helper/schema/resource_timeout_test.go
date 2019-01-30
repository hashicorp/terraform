package schema

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceTimeout_ConfigDecode_badkey(t *testing.T) {
	cases := []struct {
		Name string
		// what the resource has defined in source
		ResourceDefaultTimeout *ResourceTimeout
		// configuration provider by user in tf file
		Config map[string]interface{}
		// what we expect the parsed ResourceTimeout to be
		Expected *ResourceTimeout
		// Should we have an error (key not defined in source)
		ShouldErr bool
	}{
		{
			Name:                   "Source does not define 'delete' key",
			ResourceDefaultTimeout: timeoutForValues(10, 0, 5, 0, 0),
			Config:                 expectedConfigForValues(2, 0, 0, 1, 0),
			Expected:               timeoutForValues(10, 0, 5, 0, 0),
			ShouldErr:              true,
		},
		{
			Name:                   "Config overrides create",
			ResourceDefaultTimeout: timeoutForValues(10, 0, 5, 0, 0),
			Config:                 expectedConfigForValues(2, 0, 7, 0, 0),
			Expected:               timeoutForValues(2, 0, 7, 0, 0),
			ShouldErr:              false,
		},
		{
			Name:                   "Config overrides create, default provided. Should still have zero values",
			ResourceDefaultTimeout: timeoutForValues(10, 0, 5, 0, 3),
			Config:                 expectedConfigForValues(2, 0, 7, 0, 0),
			Expected:               timeoutForValues(2, 0, 7, 0, 3),
			ShouldErr:              false,
		},
		{
			Name:                   "Use something besides 'minutes'",
			ResourceDefaultTimeout: timeoutForValues(10, 0, 5, 0, 3),
			Config: map[string]interface{}{
				"create": "2h",
			},
			Expected:  timeoutForValues(120, 0, 5, 0, 3),
			ShouldErr: false,
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, c.Name), func(t *testing.T) {
			r := &Resource{
				Timeouts: c.ResourceDefaultTimeout,
			}

			raw, err := config.NewRawConfig(
				map[string]interface{}{
					"foo":             "bar",
					TimeoutsConfigKey: c.Config,
				})
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			conf := terraform.NewResourceConfig(raw)

			timeout := &ResourceTimeout{}
			decodeErr := timeout.ConfigDecode(r, conf)
			if c.ShouldErr {
				if decodeErr == nil {
					t.Fatalf("ConfigDecode case (%d): Expected bad timeout key: %s", i, decodeErr)
				}
				// should error, err was not nil, continue
				return
			} else {
				if decodeErr != nil {
					// should not error, error was not nil, fatal
					t.Fatalf("decodeError was not nil: %s", decodeErr)
				}
			}

			if !reflect.DeepEqual(c.Expected, timeout) {
				t.Fatalf("ConfigDecode match error case (%d).\nExpected:\n%#v\nGot:\n%#v", i, c.Expected, timeout)
			}
		})
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
			TimeoutsConfigKey: map[string]interface{}{
				"create": "2m",
				"update": "1m",
			},
		})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	c := terraform.NewResourceConfig(raw)

	timeout := &ResourceTimeout{}
	err = timeout.ConfigDecode(r, c)
	if err != nil {
		t.Fatalf("Expected good timeout returned:, %s", err)
	}

	expected := &ResourceTimeout{
		Create: DefaultTimeout(2 * time.Minute),
		Update: DefaultTimeout(1 * time.Minute),
	}

	if !reflect.DeepEqual(timeout, expected) {
		t.Fatalf("bad timeout decode.\nExpected:\n%#v\nGot:\n%#v\n", expected, timeout)
	}
}

func TestResourceTimeout_legacyConfigDecode(t *testing.T) {
	r := &Resource{
		Timeouts: &ResourceTimeout{
			Create: DefaultTimeout(10 * time.Minute),
			Update: DefaultTimeout(5 * time.Minute),
		},
	}

	raw, err := config.NewRawConfig(
		map[string]interface{}{
			"foo": "bar",
			TimeoutsConfigKey: []map[string]interface{}{
				{
					"create": "2m",
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
		t.Fatalf("Expected good timeout returned:, %s", err)
	}

	expected := &ResourceTimeout{
		Create: DefaultTimeout(2 * time.Minute),
		Update: DefaultTimeout(1 * time.Minute),
	}

	if !reflect.DeepEqual(timeout, expected) {
		t.Fatalf("bad timeout decode.\nExpected:\n%#v\nGot:\n%#v\n", expected, timeout)
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
		err := rt.DiffDecode(c.State)
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

// Generates a ResourceTimeout struct that should reflect the
// d.Timeout("key") results
func expectedTimeoutForValues(create, read, update, del, def int) *ResourceTimeout {
	rt := ResourceTimeout{}

	defaultValues := []*int{&create, &read, &update, &del, &def}
	for _, v := range defaultValues {
		if *v == 0 {
			*v = 20
		}
	}

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

		for _, k := range timeoutKeys() {
			if _, ok := ex[k]; !ok {
				ex[k] = defNano
			}
		}
	}

	return ex
}

func expectedConfigForValues(create, read, update, delete, def int) map[string]interface{} {
	ex := make(map[string]interface{}, 0)

	if create != 0 {
		ex["create"] = fmt.Sprintf("%dm", create)
	}
	if read != 0 {
		ex["read"] = fmt.Sprintf("%dm", read)
	}
	if update != 0 {
		ex["update"] = fmt.Sprintf("%dm", update)
	}
	if delete != 0 {
		ex["delete"] = fmt.Sprintf("%dm", delete)
	}

	if def != 0 {
		ex["default"] = fmt.Sprintf("%dm", def)
	}
	return ex
}
