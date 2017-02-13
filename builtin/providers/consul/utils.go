package consul

import (
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/schema"
)

func getQueryOpts(d *schema.ResourceData, client *consulapi.Client) (*consulapi.QueryOptions, error) {
	dc, err := getDC(d, client)
	if err != nil {
		return nil, err
	}

	queryOpts := &consulapi.QueryOptions{
		Datacenter: dc,
	}

	if v, ok := d.GetOk(allowStale); ok {
		queryOpts.AllowStale = v.(bool)
	}

	if v, ok := d.GetOk(requireConsistent); ok {
		queryOpts.RequireConsistent = v.(bool)
	}

	if v, ok := d.GetOk(nodeMeta); ok {
		m := v.(map[string]interface{})
		nodeMetaMap := make(map[string]string, len(nodeMeta))
		for s, t := range m {
			nodeMetaMap[s] = t.(string)
		}
		queryOpts.NodeMeta = nodeMetaMap
	}

	if v, ok := d.GetOk(token); ok {
		queryOpts.Token = v.(string)
	}

	if v, ok := d.GetOk(waitIndex); ok {
		queryOpts.WaitIndex = uint64(v.(int))
	}

	if v, ok := d.GetOk(waitTime); ok {
		d, _ := time.ParseDuration(v.(string))
		queryOpts.WaitTime = d
	}

	return queryOpts, nil
}
