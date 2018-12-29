package test

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/rand"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceStateFunc() *schema.Resource {
	return &schema.Resource{
		Create: testResourceStateFuncCreate,
		Read:   testResourceStateFuncRead,
		Update: testResourceStateFuncUpdate,
		Delete: testResourceStateFuncDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"optional": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"state_func": {
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				StateFunc: stateFuncHash,
			},
			"state_func_value": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func stateFuncHash(v interface{}) string {
	hash := sha1.Sum([]byte(v.(string)))
	return hex.EncodeToString(hash[:])
}

func testResourceStateFuncCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%x", rand.Int63()))

	// if we have a reference for the actual data in the state_func field,
	// compare it
	if data, ok := d.GetOk("state_func_value"); ok {
		expected := data.(string)
		got := d.Get("state_func").(string)
		if expected != got {
			return fmt.Errorf("expected state_func value:%q, got%q", expected, got)
		}
	}

	return testResourceStateFuncRead(d, meta)
}

func testResourceStateFuncRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceStateFuncUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceStateFuncDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
