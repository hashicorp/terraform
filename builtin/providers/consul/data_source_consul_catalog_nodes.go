package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	queryOptNodesAttr = "nodes"

	nodeID              = "id"
	nodeAddress         = "address"
	nodeMetaAttr        = "meta"
	nodeName            = "name"
	nodeTaggedAddresses = "tagged_addresses"

	queryOpts = "query_options"

	apiTaggedLAN    = "lan"
	apiTaggedWAN    = "wan"
	schemaTaggedLAN = "lan"
	schemaTaggedWAN = "wan"
)

func dataSourceConsulCatalogNodes() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceConsulCatalogNodesRead,
		Schema: map[string]*schema.Schema{
			queryOpts: schemaQueryOpts,
			queryOptNodesAttr: &schema.Schema{
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
		},
	}
}

func dataSourceConsulCatalogNodesRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)

	// Parse out data source filters to populate Consul's query options
	queryOpts, err := getQueryOpts(d, client)
	if err != nil {
		return errwrap.Wrapf("unable to get query options for fetching catalog nodes: {{err}}", err)
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
	d.SetId(fmt.Sprintf(idKeyFmt, queryOpts.Datacenter))

	d.Set("datacenter", queryOpts.Datacenter)
	if err := d.Set(queryOptNodesAttr, l); err != nil {
		return errwrap.Wrapf("Unable to store nodes: {{err}}", err)
	}

	return nil
}
