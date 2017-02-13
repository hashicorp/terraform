package consul

import (
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	queryOptAllowStale        = "allow_stale"
	queryOptNodeMeta          = "node_meta"
	queryOptRequireConsistent = "require_consistent"
	queryOptToken             = "token"
	queryOptWaitIndex         = "wait_index"
	queryOptWaitTime          = "wait_time"
)

var schemaQueryOpts = &schema.Schema{
	Optional: true,
	Type:     schema.TypeSet,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			queryOptAllowStale: &schema.Schema{
				Optional: true,
				Default:  true,
				Type:     schema.TypeBool,
			},
			queryOptRequireConsistent: &schema.Schema{
				Optional: true,
				Default:  false,
				Type:     schema.TypeBool,
			},
			queryOptToken: &schema.Schema{
				Optional: true,
				Default:  true,
				Type:     schema.TypeString,
			},
			queryOptWaitIndex: &schema.Schema{
				Optional: true,
				Default:  true,
				Type:     schema.TypeInt,
				ValidateFunc: makeValidationFunc(queryOptWaitIndex, []interface{}{
					validateIntMin(0),
				}),
			},
			queryOptWaitTime: &schema.Schema{
				Optional: true,
				Default:  true,
				Type:     schema.TypeString,
				ValidateFunc: makeValidationFunc(queryOptWaitTime, []interface{}{
					validateDurationMin("0ns"),
				}),
			},
		},
	},
}

func getQueryOpts(d *schema.ResourceData, client *consulapi.Client) (*consulapi.QueryOptions, error) {
	dc, err := getDC(d, client)
	if err != nil {
		return nil, err
	}

	queryOpts := &consulapi.QueryOptions{
		Datacenter: dc,
	}

	if v, ok := d.GetOk(queryOptAllowStale); ok {
		queryOpts.AllowStale = v.(bool)
	}

	if v, ok := d.GetOk(queryOptRequireConsistent); ok {
		queryOpts.RequireConsistent = v.(bool)
	}

	if v, ok := d.GetOk(queryOptNodeMeta); ok {
		m := v.(map[string]interface{})
		nodeMetaMap := make(map[string]string, len(queryOptNodeMeta))
		for s, t := range m {
			nodeMetaMap[s] = t.(string)
		}
		queryOpts.NodeMeta = nodeMetaMap
	}

	if v, ok := d.GetOk(queryOptToken); ok {
		queryOpts.Token = v.(string)
	}

	if v, ok := d.GetOk(queryOptWaitIndex); ok {
		queryOpts.WaitIndex = uint64(v.(int))
	}

	if v, ok := d.GetOk(queryOptWaitTime); ok {
		d, _ := time.ParseDuration(v.(string))
		queryOpts.WaitTime = d
	}

	return queryOpts, nil
}
