package test

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceCustomDiff() *schema.Resource {
	return &schema.Resource{
		Create:        testResourceCustomDiffCreate,
		Read:          testResourceCustomDiffRead,
		CustomizeDiff: testResourceCustomDiffCustomizeDiff,
		Update:        testResourceCustomDiffUpdate,
		Delete:        testResourceCustomDiffDelete,
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
			"list": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

type listDiffCases struct {
	Type  string
	Value string
}

func testListDiffCases(index int) []listDiffCases {
	switch index {
	case 0:
		return []listDiffCases{
			{
				Type:  "add",
				Value: "dc1",
			},
		}
	case 1:
		return []listDiffCases{
			{
				Type:  "remove",
				Value: "dc1",
			},
			{
				Type:  "add",
				Value: "dc2",
			},
			{
				Type:  "add",
				Value: "dc3",
			},
		}
	}
	return nil
}

func testListDiffCasesReadResult(index int) []interface{} {
	switch index {
	case 1:
		return []interface{}{"dc1"}
	default:
		return []interface{}{"dc2", "dc3"}
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
	if err := d.Set("list", testListDiffCasesReadResult(d.Get("index").(int))); err != nil {
		return err
	}
	return nil
}

func testResourceCustomDiffCustomizeDiff(d *schema.ResourceDiff, meta interface{}) error {
	if d.Get("veto").(bool) == true {
		return fmt.Errorf("veto is true, diff vetoed")
	}
	// Note that this gets put into state after the update, regardless of whether
	// or not anything is acted upon in the diff.
	d.SetNew("computed", d.Get("computed").(int)+1)

	// This tests a diffed list, based off of the value of index
	dcs := testListDiffCases(d.Get("index").(int))
	s := d.Get("list").([]interface{})
	for _, dc := range dcs {
		switch dc.Type {
		case "add":
			s = append(s, dc.Value)
		case "remove":
			for i := range s {
				if s[i].(string) == dc.Value {
					copy(s[i:], s[i+1:])
					s = s[:len(s)-1]
					break
				}
			}
		}
	}
	d.SetNew("list", s)

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
	return testResourceCustomDiffRead(d, meta)
}

func testResourceCustomDiffDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
