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
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_stream_group.* resource attribute names
	streamGroupDescriptionAttr = "description"
	streamGroupNameAttr        = "name"
	streamGroupGroupAttr       = "group"
	streamGroupTagsAttr        = "tags"

	// circonus_stream_group.* out parameters
	streamGroupIDAttr = "id"

	// circonus_stream_group.group.* resource attribute names
	streamGroupQueryAttr = "query"
	streamGroupTypeAttr  = "type"
)

var streamGroupDescriptions = attrDescrs{
	streamGroupDescriptionAttr: "A description of the stream group",
	streamGroupIDAttr:          "The ID of this stream group",
	streamGroupNameAttr:        "The name of the stream group",
	streamGroupGroupAttr:       "A stream group query definition",
	streamGroupTagsAttr:        "A list of tags assigned to the stream group",
}

var streamGroupGroupDescriptions = attrDescrs{
	streamGroupQueryAttr: "A query of metric streams",
	streamGroupTypeAttr:  "The operation to perform on the matching stream group",
}

func newStreamGroupResource() *schema.Resource {
	return &schema.Resource{
		Create: streamGroupCreate,
		Read:   streamGroupRead,
		Update: streamGroupUpdate,
		Delete: streamGroupDelete,
		Exists: streamGroupExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: convertToHelperSchema(streamGroupDescriptions, map[schemaAttr]*schema.Schema{
			streamGroupDescriptionAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: suppressWhitespace,
			},
			streamGroupNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			streamGroupGroupAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(streamGroupGroupDescriptions, map[schemaAttr]*schema.Schema{
						streamGroupQueryAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateRegexp(streamGroupQueryAttr, `.+`),
						},
						streamGroupTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateStringIn(streamGroupTypeAttr, supportedStreamGroupTypes),
						},
					}),
				},
			},
			streamGroupTagsAttr: tagMakeConfigSchema(streamGroupTagsAttr),

			// Out parameters
			streamGroupIDAttr: &schema.Schema{
				Computed:     true,
				Type:         schema.TypeString,
				ValidateFunc: validateRegexp(streamGroupIDAttr, config.MetricClusterCIDRegex),
			},
		}),
	}
}

func streamGroupCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	sg := newStreamGroup()

	if err := sg.ParseConfig(d); err != nil {
		return errwrap.Wrapf("error parsing stream group schema during create: {{err}}", err)
	}

	if err := sg.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating stream group: {{err}}", err)
	}

	d.SetId(sg.CID)

	return streamGroupRead(d, meta)
}

func streamGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	sg, err := ctxt.client.FetchMetricCluster(api.CIDType(&cid), "")
	if err != nil {
		if strings.Contains(err.Error(), defaultCirconus404ErrorString) {
			return false, nil
		}

		return false, err
	}

	if sg.CID == "" {
		return false, nil
	}

	return true, nil
}

// streamGroupRead pulls data out of the MetricCluster object and stores it
// into the appropriate place in the statefile.
func streamGroupRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	sg, err := loadStreamGroup(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	d.SetId(sg.CID)

	groups := schema.NewSet(streamGroupGroupChecksum, nil)
	for _, g := range sg.Queries {
		groupAttrs := map[string]interface{}{
			string(streamGroupQueryAttr): g.Query,
			string(streamGroupTypeAttr):  g.Type,
		}

		groups.Add(groupAttrs)
	}

	d.Set(streamGroupDescriptionAttr, sg.Description)
	d.Set(streamGroupNameAttr, sg.Name)

	if err := d.Set(streamGroupTagsAttr, tagsToState(apiToTags(sg.Tags))); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store stream group %q attribute: {{err}}", streamGroupTagsAttr), err)
	}

	d.Set(streamGroupIDAttr, sg.CID)

	return nil
}

func streamGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	sg := newStreamGroup()

	if err := sg.ParseConfig(d); err != nil {
		return err
	}

	sg.CID = d.Id()
	if err := sg.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update stream group %q: {{err}}", d.Id()), err)
	}

	return streamGroupRead(d, meta)
}

func streamGroupDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	if _, err := ctxt.client.DeleteMetricClusterByCID(api.CIDType(&cid)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete stream group %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

func streamGroupGroupChecksum(v interface{}) int {
	m := v.(map[string]interface{})

	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	if v, found := m[streamGroupQueryAttr]; found {
		fmt.Fprint(b, v.(string))
	}

	if v, found := m[streamGroupTypeAttr]; found {
		fmt.Fprint(b, v.(string))
	}

	s := b.String()
	return hashcode.String(s)
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus MetricCluster object.
func (sg *circonusStreamGroup) ParseConfig(d *schema.ResourceData) error {
	if v, found := d.GetOk(streamGroupDescriptionAttr); found {
		sg.Description = v.(string)
	}

	if v, found := d.GetOk(streamGroupNameAttr); found {
		sg.Name = v.(string)
	}

	if groupListRaw, found := d.GetOk(streamGroupGroupAttr); found {
		groupList := groupListRaw.(*schema.Set).List()

		sg.Queries = make([]api.MetricQuery, 0, len(groupList))

		for _, groupListRaw := range groupList {
			groupAttrs := newInterfaceMap(groupListRaw)

			var query string
			if v, found := groupAttrs[streamGroupQueryAttr]; found {
				query = v.(string)
			}

			var queryType string
			if v, found := groupAttrs[streamGroupTypeAttr]; found {
				queryType = v.(string)
			}

			sg.Queries = append(sg.Queries, api.MetricQuery{
				Query: query,
				Type:  queryType,
			})
		}
	}

	if v, found := d.GetOk(streamGroupTagsAttr); found {
		sg.Tags = derefStringList(flattenSet(v.(*schema.Set)))
	}

	if err := sg.Validate(); err != nil {
		return err
	}

	return nil
}
