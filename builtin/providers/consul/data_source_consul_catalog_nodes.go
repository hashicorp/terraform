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
	_CatalogNodes _TypeKey = iota
	_CatalogNodesAllowStale
	_CatalogNodesDatacenter
	_CatalogNodesNear
	_CatalogNodesRequireConsistent
	_CatalogNodesToken
	_CatalogNodesWaitIndex
	_CatalogNodesWaitTime
)

// node.* attributes
const (
	_CatalogNodeID _TypeKey = iota
	_CatalogNodeName
	_CatalogNodeAddress
	_CatalogNodeTaggedAddresses
	_CatalogNodeMeta
)

// node.tagged_addresses.* attributes
const (
	_CatalogNodeTaggedAddressesLAN _TypeKey = iota
	_CatalogNodeTaggedAddressesWAN
)

var _CatalogNodeAttrs = map[_TypeKey]*_TypeEntry{
	_CatalogNodeID: {
		APIName:    "ID",
		SchemaName: "id",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
		ValidateFuncs: []interface{}{
			_ValidateRegexp(`^[\S]+$`),
		},
		APITest: func(e *_TypeEntry, v interface{}) (interface{}, bool) {
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
	_CatalogNodeName: {
		APIName:    "Name",
		SchemaName: "name",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
		ValidateFuncs: []interface{}{
			_ValidateRegexp(`^[\S]+$`),
		},
		APITest: func(e *_TypeEntry, v interface{}) (interface{}, bool) {
			node := v.(*consulapi.Node)

			if name := node.Node; name != "" {
				return name, true
			}

			return "", false
		},
	},
	_CatalogNodeAddress: {
		APIName:    "Address",
		SchemaName: "address",
		Source:     _SourceAPIResult,
		Type:       schema.TypeString,
		APITest: func(e *_TypeEntry, v interface{}) (interface{}, bool) {
			node := v.(*consulapi.Node)

			if addr := node.Address; addr != "" {
				return addr, true
			}

			return "", false
		},
	},
	_CatalogNodeTaggedAddresses: {
		APIName:    "TaggedAddresses",
		SchemaName: "tagged_addresses",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
		SetMembers: map[_TypeKey]*_TypeEntry{
			_CatalogNodeTaggedAddressesLAN: {
				APIName:    "LAN",
				SchemaName: "lan",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
				APITest: func(e *_TypeEntry, v interface{}) (interface{}, bool) {
					m := v.(map[string]string)

					if addr, found := m[string(e.SchemaName)]; found {
						return addr, true
					}

					return nil, false
				},
			},
			_CatalogNodeTaggedAddressesWAN: {
				APIName:    "WAN",
				SchemaName: "wan",
				Source:     _SourceAPIResult,
				Type:       schema.TypeString,
				APITest: func(e *_TypeEntry, v interface{}) (interface{}, bool) {
					m := v.(map[string]string)

					if addr, found := m[string(e.SchemaName)]; found {
						return addr, true
					}

					return nil, false
				},
			},
		},
		APITest: func(e *_TypeEntry, v interface{}) (interface{}, bool) {
			node := v.(*consulapi.Node)

			if addrs := node.TaggedAddresses; len(addrs) > 0 {
				return _MapStringToMapInterface(addrs), true
			}

			return nil, false
		},
	},
	_CatalogNodeMeta: {
		APIName:    "Meta",
		SchemaName: "meta",
		Source:     _SourceAPIResult,
		Type:       schema.TypeMap,
		APITest: func(e *_TypeEntry, v interface{}) (interface{}, bool) {
			node := v.(*consulapi.Node)

			if meta := node.Meta; len(meta) > 0 {
				return _MapStringToMapInterface(meta), true
			}

			return nil, false
		},
	},
}

var _CatalogNodesAttrs = map[_TypeKey]*_TypeEntry{
	_CatalogNodesAllowStale: {
		SchemaName: "allow_stale",
		Source:     _SourceLocalFilter,
		Type:       schema.TypeBool,
		Default:    true,
		ConfigRead: func(e *_TypeEntry, r _AttrReader) (interface{}, bool) {
			b, ok := r.GetBoolOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return b, true
		},
		ConfigUse: func(e *_TypeEntry, v interface{}, target interface{}) error {
			b := v.(bool)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.AllowStale = b
			return nil
		},
	},
	_CatalogNodesDatacenter: {
		SchemaName: "datacenter",
		Source:     _SourceLocalFilter,
		Type:       schema.TypeString,
		ConfigRead: func(e *_TypeEntry, r _AttrReader) (interface{}, bool) {
			s, ok := r.GetStringOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return s, true
		},
		ConfigUse: func(e *_TypeEntry, v interface{}, target interface{}) error {
			s := v.(string)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.Datacenter = s
			return nil
		},
	},
	_CatalogNodesNear: {
		SchemaName: "near",
		Source:     _SourceLocalFilter,
		Type:       schema.TypeString,
		ConfigRead: func(e *_TypeEntry, r _AttrReader) (interface{}, bool) {
			s, ok := r.GetStringOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return s, true
		},
		ConfigUse: func(e *_TypeEntry, v interface{}, target interface{}) error {
			s := v.(string)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.Near = s
			return nil
		},
	},
	_CatalogNodes: {
		SchemaName: "nodes",
		Source:     _SourceAPIResult,
		Type:       schema.TypeList,
		ListSchema: _CatalogNodeAttrs,
	},
	_CatalogNodesRequireConsistent: {
		SchemaName: "require_consistent",
		Source:     _SourceLocalFilter,
		Type:       schema.TypeBool,
		Default:    false,
		ConfigRead: func(e *_TypeEntry, r _AttrReader) (interface{}, bool) {
			b, ok := r.GetBoolOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return b, true
		},
		ConfigUse: func(e *_TypeEntry, v interface{}, target interface{}) error {
			b := v.(bool)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.RequireConsistent = b
			return nil
		},
	},
	_CatalogNodesToken: {
		SchemaName: "token",
		Source:     _SourceLocalFilter,
		Type:       schema.TypeString,
		ConfigRead: func(e *_TypeEntry, r _AttrReader) (interface{}, bool) {
			s, ok := r.GetStringOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return s, true
		},
		ConfigUse: func(e *_TypeEntry, v interface{}, target interface{}) error {
			s := v.(string)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.Token = s
			return nil
		},
	},
	_CatalogNodesWaitIndex: {
		SchemaName: "wait_index",
		Source:     _SourceLocalFilter,
		Type:       schema.TypeInt,
		ValidateFuncs: []interface{}{
			_ValidateIntMin(0),
		},
		ConfigRead: func(e *_TypeEntry, r _AttrReader) (interface{}, bool) {
			i, ok := r.GetIntOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return uint64(i), true
		},
		ConfigUse: func(e *_TypeEntry, v interface{}, target interface{}) error {
			i := v.(uint64)
			queryOpts := target.(*consulapi.QueryOptions)
			queryOpts.WaitIndex = i
			return nil
		},
	},
	_CatalogNodesWaitTime: {
		SchemaName: "wait_time",
		Source:     _SourceLocalFilter,
		Type:       schema.TypeString,
		ValidateFuncs: []interface{}{
			_ValidateDurationMin("0ns"),
		},
		ConfigRead: func(e *_TypeEntry, r _AttrReader) (interface{}, bool) {
			d, ok := r.GetDurationOK(e.SchemaName)
			if !ok {
				return nil, false
			}

			return d, true
		},
		ConfigUse: func(e *_TypeEntry, v interface{}, target interface{}) error {
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
		Schema: _TypeEntryMapToSchema(_CatalogNodesAttrs),
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

	cfgReader := _NewConfigReader(d)

	// Construct the query options
	for _, e := range _CatalogNodesAttrs[_CatalogNodes].ListSchema {
		// Only evaluate attributes that impact the state
		if e.Source&_SourceLocalFilter == 0 {
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
		mWriter := _NewMapWriter(make(map[string]interface{}, len(_CatalogNodeAttrs)))

		// /v1/catalog/nodes returns a list of node objects
		for _, e := range _CatalogNodesAttrs[_CatalogNodes].ListSchema {
			// Only evaluate attributes that impact the state
			if e.Source&_ModifyState == 0 {
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

	dataSourceWriter := _NewStateWriter(d)
	dataSourceWriter.SetList(_CatalogNodesAttrs[_CatalogNodes].SchemaName, l)
	dataSourceWriter.SetString(_CatalogNodesAttrs[_CatalogNodesDatacenter].SchemaName, dc)
	const idKeyFmt = "catalog-nodes-%s"
	dataSourceWriter.SetID(fmt.Sprintf(idKeyFmt, dc))

	return nil
}
