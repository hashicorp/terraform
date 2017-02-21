package random

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceShuffle() *schema.Resource {
	return &schema.Resource{
		Create: CreateShuffle,
		Read:   schema.Noop,
		Delete: schema.RemoveFromState,

		Schema: map[string]*schema.Schema{
			"keepers": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"seed": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"input": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"result": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"result_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func CreateShuffle(d *schema.ResourceData, _ interface{}) error {
	input := d.Get("input").([]interface{})
	seed := d.Get("seed").(string)

	resultCount := d.Get("result_count").(int)
	if resultCount == 0 {
		resultCount = len(input)
	}
	result := make([]interface{}, 0, resultCount)

	rand := NewRand(seed)

	// Keep producing permutations until we fill our result
Batches:
	for {
		perm := rand.Perm(len(input))

		for _, i := range perm {
			result = append(result, input[i])

			if len(result) >= resultCount {
				break Batches
			}
		}
	}

	d.SetId("-")
	d.Set("result", result)

	return nil
}
