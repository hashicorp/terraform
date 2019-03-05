package test

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceComputedSet() *schema.Resource {
	return &schema.Resource{
		Create: testResourceComputedSetCreate,
		Read:   testResourceComputedSetRead,
		Delete: testResourceComputedSetDelete,
		Update: testResourceComputedSetUpdate,

		CustomizeDiff: func(d *schema.ResourceDiff, _ interface{}) error {
			o, n := d.GetChange("set_count")
			if o != n {
				d.SetNewComputed("string_set")
			}
			return nil
		},

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"set_count": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"string_set": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},

			"rule": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"ip_protocol": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"cidr": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
							StateFunc: func(v interface{}) string {
								return strings.ToLower(v.(string))
							},
						},
					},
				},
			},
		},
	}
}

func computeSecGroupV2RuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["ip_protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", strings.ToLower(m["cidr"].(string))))

	return hashcode.String(buf.String())
}

func testResourceComputedSetCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%x", rand.Int63()))
	return testResourceComputedSetRead(d, meta)
}

func testResourceComputedSetRead(d *schema.ResourceData, meta interface{}) error {
	count := 3
	v, ok := d.GetOk("set_count")
	if ok {
		count = v.(int)
	}

	var set []interface{}
	for i := 0; i < count; i++ {
		set = append(set, fmt.Sprintf("%d", i))
	}

	d.Set("string_set", schema.NewSet(schema.HashString, set))
	return nil
}

func testResourceComputedSetUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceComputedSetRead(d, meta)
}

func testResourceComputedSetDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
