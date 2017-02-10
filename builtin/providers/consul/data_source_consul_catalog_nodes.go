package consul

import (
	"fmt"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	allowStale        = "allow_stale"
	nodeMeta          = "node_meta"
	nodesAttr         = "nodes"
	requireConsistent = "require_consistent"
	token             = "token"
	waitIndex         = "wait_index"
	waitTime          = "wait_time"

	nodeID              = "id"
	nodeAddress         = "address"
	nodeMetaAttr        = "meta"
	nodeName            = "name"
	nodeTaggedAddresses = "tagged_addresses"

	apiTaggedLAN    = "lan"
	apiTaggedWAN    = "wan"
	schemaTaggedLAN = "lan"
	schemaTaggedWAN = "wan"
)

func dataSourceConsulCatalogNodes() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceConsulCatalogNodesRead,
		Schema: map[string]*schema.Schema{
			allowStale: &schema.Schema{
				Optional: true,
				Default:  true,
				Type:     schema.TypeBool,
			},
			nodesAttr: &schema.Schema{
				Computed: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						nodeID: &schema.Schema{
							Type:         schema.TypeString,
							Computed:     true,
							ValidateFunc: makeValidationFunc(nodeID, []interface{}{validateRegexp(`^[\S]+$`)}),
						},
						nodeName: &schema.Schema{
							Type:         schema.TypeString,
							Computed:     true,
							ValidateFunc: makeValidationFunc(nodeName, []interface{}{validateRegexp(`^[\S]+$`)}),
						},
						nodeAddress: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						nodeMetaAttr: &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
						},
						nodeTaggedAddresses: &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									schemaTaggedLAN: &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									schemaTaggedWAN: &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			requireConsistent: &schema.Schema{
				Optional: true,
				Default:  false,
				Type:     schema.TypeBool,
			},
			token: &schema.Schema{
				Optional: true,
				Default:  true,
				Type:     schema.TypeString,
			},
			waitIndex: &schema.Schema{
				Optional: true,
				Default:  true,
				Type:     schema.TypeInt,
				ValidateFunc: makeValidationFunc(waitIndex, []interface{}{
					validateIntMin(0),
				}),
			},
			waitTime: &schema.Schema{
				Optional: true,
				Default:  true,
				Type:     schema.TypeString,
				ValidateFunc: makeValidationFunc(waitTime, []interface{}{
					validateDurationMin("0ns"),
				}),
			},
		},
	}
}

func dataSourceConsulCatalogNodesRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)

	// Parse out data source filters to populate Consul's query options

	dc, err := getDC(d, client)
	if err != nil {
		return err
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

	nodes, meta, err := client.Catalog().Nodes(queryOpts)
	if err != nil {
		return err
	}

	l := make([]interface{}, 0, len(nodes))

	for _, node := range nodes {
		const defaultNodeAttrs = 4
		m := make(map[string]interface{}, defaultNodeAttrs)
		id := node.ID
		if id == "" {
			id = node.Node
		}

		m[nodeID] = id
		m[nodeName] = node.Node
		m[nodeAddress] = node.Address

		{
			const initNumTaggedAddrs = 2
			taggedAddrs := make(map[string]interface{}, initNumTaggedAddrs)
			if addr, found := node.TaggedAddresses[apiTaggedLAN]; found {
				taggedAddrs[schemaTaggedLAN] = addr
			}
			if addr, found := node.TaggedAddresses[apiTaggedWAN]; found {
				taggedAddrs[schemaTaggedWAN] = addr
			}
			m[nodeTaggedAddresses] = taggedAddrs
		}

		{
			const initNumMetaAddrs = 4
			metaVals := make(map[string]interface{}, initNumMetaAddrs)
			for s, t := range node.Meta {
				metaVals[s] = t
			}
			m[nodeMetaAttr] = metaVals
		}

		l = append(l, m)
	}

	const idKeyFmt = "catalog-nodes-%s"
	d.SetId(fmt.Sprintf(idKeyFmt, dc))

	d.Set("datacenter", dc)
	if err := d.Set(nodesAttr, l); err != nil {
		return errwrap.Wrapf("Unable to store nodes: {{err}}", err)
	}

	return nil
}
