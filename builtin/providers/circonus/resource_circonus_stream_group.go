package circonus

/*
 * Note to future readers: The `circonus_stream_group` resource is actually a
 * facade over the metric_cluster endpoint.
 */

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_stream_group.* resource attribute names
	_StreamGroupDescriptionAttr _SchemaAttr = "description"
	_StreamGroupNameAttr        _SchemaAttr = "name"
	_StreamGroupGroupAttr       _SchemaAttr = "group"
	_StreamGroupTagsAttr        _SchemaAttr = "tags"

	// circonus_stream_group.group.* resource attribute names
	_StreamGroupQueryAttr _SchemaAttr = "query"
	_StreamGroupTypeAttr  _SchemaAttr = "type"
)

var _StreamGroupDescriptions = _AttrDescrs{
	_StreamGroupDescriptionAttr: "A description of the stream group",
	_StreamGroupNameAttr:        "The name of the stream group",
	_StreamGroupGroupAttr:       "A stream group query definition",
	_StreamGroupTagsAttr:        "A list of tags assigned to the stream group",
}

var _StreamGroupGroupDescriptions = _AttrDescrs{
	_StreamGroupQueryAttr: "A query of metric streams",
	_StreamGroupTypeAttr:  "The operation to perform on the matching stream group",
}

func _NewStreamGroupResource() *schema.Resource {
	return &schema.Resource{
		Create: _StreamGroupCreate,
		Read:   _StreamGroupRead,
		Update: _StreamGroupUpdate,
		Delete: _StreamGroupDelete,
		Exists: _StreamGroupExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_StreamGroupDescriptionAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(v interface{}) string {
					return strings.TrimSpace(v.(string))
				},
			},
			_StreamGroupNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			_StreamGroupGroupAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
						_StreamGroupQueryAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: _ValidateRegexp(_StreamGroupQueryAttr, `.+`),
						},
						_StreamGroupTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: _ValidateStringIn(_StreamGroupTypeAttr, _SupportedStreamGroupTypes),
						},
					}, _StreamGroupGroupDescriptions),
				},
			},
			_StreamGroupTagsAttr: _TagMakeConfigSchema(_StreamGroupTagsAttr),
		}, _StreamGroupDescriptions),
	}
}

func _StreamGroupCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)
	sg := _NewStreamGroup()
	cr := _NewConfigReader(ctxt, d)
	if err := sg.ParseConfig(cr); err != nil {
		return errwrap.Wrapf("error parsing stream group schema during create: {{err}}", err)
	}

	if err := sg.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating stream group: {{err}}", err)
	}

	d.SetId(sg.CID)

	return _StreamGroupRead(d, meta)
}

func _StreamGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	sg, err := ctxt.client.FetchMetricCluster(api.CIDType(&cid), "")
	if err != nil {
		return false, err
	}

	if sg.CID == "" {
		return false, nil
	}

	return true, nil
}

// _StreamGroupRead pulls data out of the MetricCluster object and stores it
// into the appropriate place in the statefile.
func _StreamGroupRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	sg, err := _LoadStreamGroup(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	groups := schema.NewSet(_StreamGroupGroupChecksum, nil)
	for _, g := range sg.Queries {
		groupAttrs := map[string]interface{}{
			string(_StreamGroupQueryAttr): g.Query,
			string(_StreamGroupTypeAttr):  g.Type,
		}

		groups.Add(groupAttrs)
	}

	_StateSet(d, _StreamGroupDescriptionAttr, sg.Description)
	_StateSet(d, _StreamGroupNameAttr, sg.Name)
	_StateSet(d, _StreamGroupTagsAttr, tagsToState(apiToTags(sg.Tags)))

	d.SetId(sg.CID)

	return nil
}

func _StreamGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)
	sg := _NewStreamGroup()
	cr := _NewConfigReader(ctxt, d)
	if err := sg.ParseConfig(cr); err != nil {
		return err
	}

	sg.CID = d.Id()
	if err := sg.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update stream group %q: {{err}}", d.Id()), err)
	}

	return _StreamGroupRead(d, meta)
}

func _StreamGroupDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	if _, err := ctxt.client.DeleteMetricClusterByCID(api.CIDType(&cid)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete stream group %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

func _StreamGroupGroupChecksum(v interface{}) int {
	m := v.(map[string]interface{})
	ar := _NewMapReader(nil, m)

	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	fmt.Fprint(b, ar.GetString(_StreamGroupQueryAttr))
	fmt.Fprint(b, ar.GetString(_StreamGroupTypeAttr))

	s := b.String()
	return hashcode.String(s)
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus MetricCluster object.
func (sg *_StreamGroup) ParseConfig(ar _AttrReader) error {
	if s, ok := ar.GetStringOK(_StreamGroupDescriptionAttr); ok {
		sg.Description = s
	}

	if s, ok := ar.GetStringOK(_StreamGroupNameAttr); ok {
		sg.Name = s
	}

	if groupList, ok := ar.GetSetAsListOK(_StreamGroupGroupAttr); ok {
		sg.Queries = make([]api.MetricQuery, 0, len(groupList))

		for _, groupListRaw := range groupList {
			groupAttrs := _NewInterfaceMap(groupListRaw)
			gr := _NewMapReader(ar.Context(), groupAttrs)

			var query string
			if s, ok := gr.GetStringOK(_StreamGroupQueryAttr); ok {
				query = s
			}

			var queryType string
			if s, ok := gr.GetStringOK(_StreamGroupTypeAttr); ok {
				queryType = s
			}

			sg.Queries = append(sg.Queries, api.MetricQuery{
				Query: query,
				Type:  queryType,
			})
		}
	}

	sg.Tags = tagsToAPI(ar.GetTags(_StreamGroupTagsAttr))

	if err := sg.Validate(); err != nil {
		return err
	}

	return nil
}
