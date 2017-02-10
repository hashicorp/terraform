package consul

import (
	"fmt"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

// Top-level consul_catalog_nodes attributes
const (
	catalogNodes typeKey = iota
	catalogNodesAllowStale
	catalogNodesDatacenter
	catalogNodesNear
	catalogNodesRequireConsistent
	catalogNodesToken
	catalogNodesWaitIndex
	catalogNodesWaitTime
)

// node.* attributes
const (
	catalogNodeID typeKey = iota
	catalogNodeName
	catalogNodeAddress
	catalogNodeTaggedAddresses
	catalogNodeMeta
)

// node.tagged_addresses.* attributes
const (
	catalogNodeTaggedAddressesLAN typeKey = iota
	catalogNodeTaggedAddressesWAN
)

var catalogNodeAttrs = map[typeKey]*typeEntry{
	catalogNodeID: {
		APIName:    "ID",
		SchemaName: "id",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
		ValidateFuncs: []interface{}{
			validateRegexp(`^[\S]+$`),
		},
		APITest: func(e *typeEntry, v interface{}) (interface{}, bool) {
			node := v.(*consulapi.Node)

			if id := node.ID; id != "" {
				return id, true
			}

			// Use the node name - confusingly stored in the Node attribute - if no ID
			// is available.
			if name := node.Node; name != "" {
				return name, true
			}

			return "", false
		},
	},
	catalogNodeName: {
		APIName:    "Name",
		SchemaName: "name",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
		ValidateFuncs: []interface{}{
			validateRegexp(`^[\S]+$`),
		},
		APITest: func(e *typeEntry, v interface{}) (interface{}, bool) {
			node := v.(*consulapi.Node)

			if name := node.Node; name != "" {
				return name, true
			}

			return "", false
		},
	},
	catalogNodeAddress: {
		APIName:    "Address",
		SchemaName: "address",
		Source:     sourceAPIResult,
		Type:       schema.TypeString,
		APITest: func(e *typeEntry, v interface{}) (interface{}, bool) {
			node := v.(*consulapi.Node)

			if addr := node.Address; addr != "" {
				return addr, true
			}

			return "", false
		},
	},
	catalogNodeTaggedAddresses: {
		APIName:    "TaggedAddresses",
		SchemaName: "tagged_addresses",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[typeKey]*typeEntry{
			catalogNodeTaggedAddressesLAN: {
				APIName:    "LAN",
				SchemaName: "lan",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
				APITest: func(e *typeEntry, v interface{}) (interface{}, bool) {
					m := v.(map[string]string)

					if addr, found := m[string(e.SchemaName)]; found {
						return addr, true
					}

					return nil, false
				},
			},
			catalogNodeTaggedAddressesWAN: {
				APIName:    "WAN",
				SchemaName: "wan",
				Source:     sourceAPIResult,
				Type:       schema.TypeString,
				APITest: func(e *typeEntry, v interface{}) (interface{}, bool) {
					m := v.(map[string]string)

					if addr, found := m[string(e.SchemaName)]; found {
						return addr, true
					}

					return nil, false
				},
			},
		},
		APITest: func(e *typeEntry, v interface{}) (interface{}, bool) {
			node := v.(*consulapi.Node)

			if addrs := node.TaggedAddresses; len(addrs) > 0 {
				return mapStringToMapInterface(addrs), true
			}

			return nil, false
		},
	},
	catalogNodeMeta: {
		APIName:    "Meta",
		SchemaName: "meta",
		Source:     sourceAPIResult,
		Type:       schema.TypeMap,
		APITest: func(e *typeEntry, v interface{}) (interface{}, bool) {
			node := v.(*consulapi.Node)

			if meta := node.Meta; len(meta) > 0 {
				return mapStringToMapInterface(meta), true
			}

			return nil, false
		},
	},
}

var catalogNodesAttrs = map[typeKey]*typeEntry{
	catalogNodesAllowStale: {
		SchemaName: "allow_stale",
		Source:     sourceLocalFilter,
		Type:       schema.TypeBool,
		Default:    true,
		ConfigRead: func(e *typeEntry, r attrReader) (interface{}, bool) {
			b, ok := r.GetBoolOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return b, true
		},
		ConfigUse: func(e *typeEntry, v interface{}, target interface{}) error {
			b := v.(bool)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.AllowStale = b
			return nil
		},
	},
	catalogNodesDatacenter: {
		SchemaName: "datacenter",
		Source:     sourceLocalFilter,
		Type:       schema.TypeString,
		ConfigRead: func(e *typeEntry, r attrReader) (interface{}, bool) {
			s, ok := r.GetStringOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return s, true
		},
		ConfigUse: func(e *typeEntry, v interface{}, target interface{}) error {
			s := v.(string)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.Datacenter = s
			return nil
		},
	},
	catalogNodesNear: {
		SchemaName: "near",
		Source:     sourceLocalFilter,
		Type:       schema.TypeString,
		ConfigRead: func(e *typeEntry, r attrReader) (interface{}, bool) {
			s, ok := r.GetStringOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return s, true
		},
		ConfigUse: func(e *typeEntry, v interface{}, target interface{}) error {
			s := v.(string)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.Near = s
			return nil
		},
	},
	catalogNodes: {
		SchemaName: "nodes",
		Source:     sourceAPIResult,
		Type:       schema.TypeList,
		ListSchema: catalogNodeAttrs,
	},
	catalogNodesRequireConsistent: {
		SchemaName: "require_consistent",
		Source:     sourceLocalFilter,
		Type:       schema.TypeBool,
		Default:    false,
		ConfigRead: func(e *typeEntry, r attrReader) (interface{}, bool) {
			b, ok := r.GetBoolOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return b, true
		},
		ConfigUse: func(e *typeEntry, v interface{}, target interface{}) error {
			b := v.(bool)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.RequireConsistent = b
			return nil
		},
	},
	catalogNodesToken: {
		SchemaName: "token",
		Source:     sourceLocalFilter,
		Type:       schema.TypeString,
		ConfigRead: func(e *typeEntry, r attrReader) (interface{}, bool) {
			s, ok := r.GetStringOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return s, true
		},
		ConfigUse: func(e *typeEntry, v interface{}, target interface{}) error {
			s := v.(string)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.Token = s
			return nil
		},
	},
	catalogNodesWaitIndex: {
		SchemaName: "wait_index",
		Source:     sourceLocalFilter,
		Type:       schema.TypeInt,
		ValidateFuncs: []interface{}{
			validateIntMin(0),
		},
		ConfigRead: func(e *typeEntry, r attrReader) (interface{}, bool) {
			i, ok := r.GetIntOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return uint64(i), true
		},
		ConfigUse: func(e *typeEntry, v interface{}, target interface{}) error {
			i := v.(uint64)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.WaitIndex = i
			return nil
		},
	},
	catalogNodesWaitTime: {
		SchemaName: "wait_time",
		Source:     sourceLocalFilter,
		Type:       schema.TypeString,
		ValidateFuncs: []interface{}{
			validateDurationMin("0ns"),
		},
		ConfigRead: func(e *typeEntry, r attrReader) (interface{}, bool) {
			d, ok := r.GetDurationOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return d, true
		},
		ConfigUse: func(e *typeEntry, v interface{}, target interface{}) error {
			d := v.(time.Duration)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.WaitTime = d
			return nil
		},
	},
}

func dataSourceConsulCatalogNodes() *schema.Resource {
	return &schema.Resource{
		Read:   dataSourceConsulCatalogNodesRead,
		Schema: typeEntryMapToSchema(catalogNodesAttrs),
	}
}

func dataSourceConsulCatalogNodesRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)

	dc, err := getDC(d, client)
	if err != nil {
		return err
	}

	queryOpts := &consulapi.QueryOptions{
		Datacenter: dc,
	}

	cfgReader := newConfigReader(d)

	// Construct the query options
	for _, e := range catalogNodesAttrs[catalogNodes].ListSchema {
		// Only evaluate attributes that impact the state
		if e.Source&sourceLocalFilter == 0 {
			continue
		}

		if v, ok := e.ConfigRead(e, cfgReader); ok {
			if err := e.ConfigUse(e, v, queryOpts); err != nil {
				return errwrap.Wrapf(fmt.Sprintf("error writing %q's query option: {{err}}", e.SchemaName), err)
			}
		}
	}

	nodes, meta, err := client.Catalog().Nodes(queryOpts)
	if err != nil {
		return err
	}

	// TODO(sean@): It'd be nice if this data source had a way of filtering out
	// irrelevant data so only the important bits are persisted in the state file.
	// Something like an attribute mask or even a regexp of matching schema
	// attributesknames would be sufficient in the most basic case.  Food for
	// thought.

	l := make([]interface{}, 0, len(nodes))

	for _, node := range nodes {
		mWriter := newMapWriter(make(map[string]interface{}, len(catalogNodeAttrs)))

		// /v1/catalog/nodes returns a list of node objects
		for _, e := range catalogNodesAttrs[catalogNodes].ListSchema {
			// Only evaluate attributes that impact the state
			if e.Source&modifyState == 0 {
				continue
			}

			h := e.MustLookupTypeHandler()

			if v, ok := h.APITest(e, node); ok {
				if err := h.APIToState(e, v, mWriter); err != nil {
					return errwrap.Wrapf(fmt.Sprintf("error writing %q's data to state: {{err}}", e.SchemaName), err)
				}
			}
		}

		l = append(l, mWriter.ToMap())
	}

	dataSourceWriter := newStateWriter(d)
	dataSourceWriter.SetList(catalogNodesAttrs[catalogNodes].SchemaName, l)
	dataSourceWriter.SetString(catalogNodesAttrs[catalogNodesDatacenter].SchemaName, dc)
	const idKeyFmt = "catalog-nodes-%s"
	dataSourceWriter.SetID(fmt.Sprintf(idKeyFmt, dc))

	return nil
}
