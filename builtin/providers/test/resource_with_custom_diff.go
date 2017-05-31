package test

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceCustomDiff() *schema.Resource {
	return &schema.Resource{
		Create: testResourceCustomDiffCreate,
		Read:   testResourceCustomDiffRead,
		Review: testResourceCustomDiffReview,
		Update: testResourceCustomDiffUpdate,
		Delete: testResourceCustomDiffDelete,
		Schema: map[string]*schema.Schema{
			"required": {
				Type:     schema.TypeString,
				Required: true,
			},
			"computed": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"index": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"veto": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func testResourceCustomDiffCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")

	// Required must make it through to Create
	if _, ok := d.GetOk("required"); !ok {
		return fmt.Errorf("missing attribute 'required', but it's required")
	}

	_, new := d.GetChange("computed")
	expected := new.(int) - 1
	actual := d.Get("index").(int)
	if expected != actual {
		return fmt.Errorf("expected computed to be 1 ahead of index, got computed: %d, index: %d", expected, actual)
	}
	d.Set("index", new)

	return testResourceCustomDiffRead(d, meta)
}

func testResourceCustomDiffRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceCustomDiffReview(d *schema.ResourceDiff, meta interface{}) error {
	if d.Get("veto").(bool) == true {
		return fmt.Errorf("veto is true, diff vetoed")
	}
	// Note that this gets put into state after the update, regardless of whether
	// or not anything is acted upon in the diff.
	d.SetNew("computed", d.Get("computed").(int)+1)
	return nil
}

func testResourceCustomDiffUpdate(d *schema.ResourceData, meta interface{}) error {
	_, new := d.GetChange("computed")
	expected := new.(int) - 1
	actual := d.Get("index").(int)
	if expected != actual {
		return fmt.Errorf("expected computed to be 1 ahead of index, got computed: %d, index: %d", expected, actual)
	}
	d.Set("index", new)
	return nil
}

func testResourceCustomDiffDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
