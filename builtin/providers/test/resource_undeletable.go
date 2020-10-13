package test

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceUndeleteable() *schema.Resource {
	return &schema.Resource{
		Create: testResourceUndeleteableCreate,
		Read:   testResourceUndeleteableRead,
		Delete: testResourceUndeleteableDelete,

		Schema: map[string]*schema.Schema{},
	}
}

func testResourceUndeleteableCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("placeholder")
	return testResourceUndeleteableRead(d, meta)
}

func testResourceUndeleteableRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceUndeleteableDelete(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("test_undeleteable always fails deletion (use terraform state rm if you really want to delete it)")
}
