package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	catalogNodesElem       = "nodes"
	catalogNodesDatacenter = "datacenter"
	catalogNodesQueryOpts  = "query_options"

	catalogNodesNodeID              = "id"
	catalogNodesNodeAddress         = "address"
	catalogNodesNodeMeta            = "meta"
	catalogNodesNodeName            = "name"
	catalogNodesNodeTaggedAddresses = "tagged_addresses"

	catalogNodesNodeIDs   = "node_ids"
	catalogNodesNodeNames = "node_names"

	catalogNodesAPITaggedLAN    = "lan"
	catalogNodesAPITaggedWAN    = "wan"
	catalogNodesSchemaTaggedLAN = "lan"
	catalogNodesSchemaTaggedWAN = "wan"
)

func dataSourceConsulCatalogNodes() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceConsulCatalogNodesRead,
		Schema: map[string]*schema.Schema{
			// Filters
			catalogNodesQueryOpts: schemaQueryOpts,

			// Out parameters
			catalogNodesDatacenter: &schema.Schema{
				Computed: true,
				Type:     schema.TypeString,
			},
			catalogNodesNodeIDs: &schema.Schema{
				Computed: true,
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			catalogNodesNodeNames: &schema.Schema{
				Computed: true,
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			catalogNodesElem: &schema.Schema{
				Computed: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						catalogNodesNodeID: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						catalogNodesNodeName: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						catalogNodesNodeAddress: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						catalogNodesNodeMeta: &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
						},
						catalogNodesNodeTaggedAddresses: &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									catalogNodesSchemaTaggedLAN: &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									catalogNodesSchemaTaggedWAN: &schema.Schema{
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

	nodeNames := make([]interface{}, 0, len(nodes))
	nodeIDs := make([]interface{}, 0, len(nodes))

	for _, node := range nodes {
		const defaultNodeAttrs = 4
		m := make(map[string]interface{}, defaultNodeAttrs)
		id := node.ID
		if id == "" {
			id = node.Node
		}

		nodeIDs = append(nodeIDs, id)
		nodeNames = append(nodeNames, node.Node)

		m[catalogNodesNodeAddress] = node.Address
		m[catalogNodesNodeID] = id
		m[catalogNodesNodeName] = node.Node
		m[catalogNodesNodeMeta] = node.Meta
		m[catalogNodesNodeTaggedAddresses] = node.TaggedAddresses

		l = append(l, m)
	}

	const idKeyFmt = "catalog-nodes-%s"
	d.SetId(fmt.Sprintf(idKeyFmt, queryOpts.Datacenter))

	d.Set(catalogNodesDatacenter, queryOpts.Datacenter)
	if err := d.Set(catalogNodesNodeIDs, nodeIDs); err != nil {
		return errwrap.Wrapf("Unable to store node IDs: {{err}}", err)
	}

	if err := d.Set(catalogNodesNodeNames, nodeNames); err != nil {
		return errwrap.Wrapf("Unable to store node names: {{err}}", err)
	}

	if err := d.Set(catalogNodesElem, l); err != nil {
		return errwrap.Wrapf("Unable to store nodes: {{err}}", err)
	}

	return nil
}
