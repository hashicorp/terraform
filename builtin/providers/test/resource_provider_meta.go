package test

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceProviderMeta() *schema.Resource {
	return &schema.Resource{
		Create: testResourceProviderMetaCreate,
		Read:   testResourceProviderMetaRead,
		Update: testResourceProviderMetaUpdate,
		Delete: testResourceProviderMetaDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"optional": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

type providerMeta struct {
	Foo string `cty:"foo"`
}

func testResourceProviderMetaCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")
	var m providerMeta

	err := d.GetProviderMeta(&m)
	if err != nil {
		return err
	}

	if m.Foo != "bar" {
		return fmt.Errorf("expected provider_meta.foo to be %q, was %q",
			"bar", m.Foo)
	}

	return testResourceProviderMetaRead(d, meta)
}

func testResourceProviderMetaRead(d *schema.ResourceData, meta interface{}) error {
	var m providerMeta

	err := d.GetProviderMeta(&m)
	if err != nil {
		return err
	}

	if m.Foo != "bar" {
		return fmt.Errorf("expected provider_meta.foo to be %q, was %q",
			"bar", m.Foo)
	}

	return nil
}

func testResourceProviderMetaUpdate(d *schema.ResourceData, meta interface{}) error {
	var m providerMeta

	err := d.GetProviderMeta(&m)
	if err != nil {
		return err
	}

	if m.Foo != "bar" {
		return fmt.Errorf("expected provider_meta.foo to be %q, was %q",
			"bar", m.Foo)
	}
	return testResourceProviderMetaRead(d, meta)
}

func testResourceProviderMetaDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	var m providerMeta

	err := d.GetProviderMeta(&m)
	if err != nil {
		return err
	}

	if m.Foo != "bar" {
		return fmt.Errorf("expected provider_meta.foo to be %q, was %q",
			"bar", m.Foo)
	}
	return nil
}
