package test

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/rand"

	"github.com/hashicorp/terraform/helper/hashcode"
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

			// set block with computed elements
			"set_block": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      setBlockHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"required": {
							Type:     schema.TypeString,
							Required: true,
						},
						"optional": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func stateFuncHash(v interface{}) string {
	hash := sha1.Sum([]byte(v.(string)))
	return hex.EncodeToString(hash[:])
}

func setBlockHash(v interface{}) int {
	m := v.(map[string]interface{})
	required, _ := m["required"].(string)
	optional, _ := m["optional"].(string)
	return hashcode.String(fmt.Sprintf("%s|%s", required, optional))
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

	// Check that we can lookup set elements by our computed hash.
	// This is not advised, but we can use this to make sure the final diff was
	// prepared with the correct values.
	setBlock, ok := d.GetOk("set_block")
	if ok {
		set := setBlock.(*schema.Set)
		for _, obj := range set.List() {
			idx := setBlockHash(obj)
			requiredAddr := fmt.Sprintf("%s.%d.%s", "set_block", idx, "required")
			_, ok := d.GetOkExists(requiredAddr)
			if !ok {
				return fmt.Errorf("failed to get attr %q from %#v", fmt.Sprintf(requiredAddr), d.State().Attributes)
			}
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
