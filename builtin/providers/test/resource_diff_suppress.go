package test

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceDiffSuppress() *schema.Resource {
	diffSuppress := func(k, old, new string, d *schema.ResourceData) bool {
		if old == "" || strings.Contains(new, "replace") {
			return false
		}
		return true
	}

	return &schema.Resource{
		Create: testResourceDiffSuppressCreate,
		Read:   testResourceDiffSuppressRead,
		Delete: testResourceDiffSuppressDelete,
		Update: testResourceDiffSuppressUpdate,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"optional": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"val_to_upper": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(val interface{}) string {
					return strings.ToUpper(val.(string))
				},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.ToUpper(old) == strings.ToUpper(new)
				},
			},
			"network": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "default",
				ForceNew:         true,
				DiffSuppressFunc: diffSuppress,
			},
			"subnetwork": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				DiffSuppressFunc: diffSuppress,
			},

			"node_pool": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func testResourceDiffSuppressCreate(d *schema.ResourceData, meta interface{}) error {
	d.Set("network", "modified")
	d.Set("subnetwork", "modified")

	id := fmt.Sprintf("%x", rand.Int63())
	d.SetId(id)
	return nil
}

func testResourceDiffSuppressRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceDiffSuppressUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceDiffSuppressDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
