package circonus

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
	// circonus_metric_cluster.* resource attribute names
	metricClusterDescriptionAttr = "description"
	metricClusterNameAttr        = "name"
	metricClusterQueryAttr       = "query"
	metricClusterTagsAttr        = "tags"

	// circonus_metric_cluster.* out parameters
	metricClusterIDAttr = "id"

	// circonus_metric_cluster.query.* resource attribute names
	metricClusterDefinitionAttr = "definition"
	metricClusterTypeAttr       = "type"
)

var metricClusterDescriptions = attrDescrs{
	metricClusterDescriptionAttr: "A description of the metric cluster",
	metricClusterIDAttr:          "The ID of this metric cluster",
	metricClusterNameAttr:        "The name of the metric cluster",
	metricClusterQueryAttr:       "A metric cluster query definition",
	metricClusterTagsAttr:        "A list of tags assigned to the metric cluster",
}

var metricClusterQueryDescriptions = attrDescrs{
	metricClusterDefinitionAttr: "A query to select a collection of metric streams",
	metricClusterTypeAttr:       "The operation to perform on the matching metric streams",
}

func resourceMetricCluster() *schema.Resource {
	return &schema.Resource{
		Create: metricClusterCreate,
		Read:   metricClusterRead,
		Update: metricClusterUpdate,
		Delete: metricClusterDelete,
		Exists: metricClusterExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: convertToHelperSchema(metricClusterDescriptions, map[schemaAttr]*schema.Schema{
			metricClusterDescriptionAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: suppressWhitespace,
			},
			metricClusterNameAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			metricClusterQueryAttr: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(metricClusterQueryDescriptions, map[schemaAttr]*schema.Schema{
						metricClusterDefinitionAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateRegexp(metricClusterDefinitionAttr, `.+`),
						},
						metricClusterTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateStringIn(metricClusterTypeAttr, supportedMetricClusterTypes),
						},
					}),
				},
			},
			metricClusterTagsAttr: tagMakeConfigSchema(metricClusterTagsAttr),

			// Out parameters
			metricClusterIDAttr: &schema.Schema{
				Computed: true,
				Type:     schema.TypeString,
			},
		}),
	}
}

func metricClusterCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	mc := newMetricCluster()

	if err := mc.ParseConfig(d); err != nil {
		return errwrap.Wrapf("error parsing metric cluster schema during create: {{err}}", err)
	}

	if err := mc.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating metric cluster: {{err}}", err)
	}

	d.SetId(mc.CID)

	return metricClusterRead(d, meta)
}

func metricClusterExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	mc, err := ctxt.client.FetchMetricCluster(api.CIDType(&cid), "")
	if err != nil {
		if strings.Contains(err.Error(), defaultCirconus404ErrorString) {
			return false, nil
		}

		return false, err
	}

	if mc.CID == "" {
		return false, nil
	}

	return true, nil
}

// metricClusterRead pulls data out of the MetricCluster object and stores it
// into the appropriate place in the statefile.
func metricClusterRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	mc, err := loadMetricCluster(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	d.SetId(mc.CID)

	queries := schema.NewSet(metricClusterQueryChecksum, nil)
	for _, query := range mc.Queries {
		queryAttrs := map[string]interface{}{
			string(metricClusterDefinitionAttr): query.Query,
			string(metricClusterTypeAttr):       query.Type,
		}

		queries.Add(queryAttrs)
	}

	d.Set(metricClusterDescriptionAttr, mc.Description)
	d.Set(metricClusterNameAttr, mc.Name)

	if err := d.Set(metricClusterTagsAttr, tagsToState(apiToTags(mc.Tags))); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store metric cluster %q attribute: {{err}}", metricClusterTagsAttr), err)
	}

	d.Set(metricClusterIDAttr, mc.CID)

	return nil
}

func metricClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	mc := newMetricCluster()

	if err := mc.ParseConfig(d); err != nil {
		return err
	}

	mc.CID = d.Id()
	if err := mc.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update metric cluster %q: {{err}}", d.Id()), err)
	}

	return metricClusterRead(d, meta)
}

func metricClusterDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	if _, err := ctxt.client.DeleteMetricClusterByCID(api.CIDType(&cid)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete metric cluster %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

func metricClusterQueryChecksum(v interface{}) int {
	m := v.(map[string]interface{})

	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	if v, found := m[metricClusterDefinitionAttr]; found {
		fmt.Fprint(b, v.(string))
	}

	if v, found := m[metricClusterTypeAttr]; found {
		fmt.Fprint(b, v.(string))
	}

	s := b.String()
	return hashcode.String(s)
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus MetricCluster object.
func (mc *circonusMetricCluster) ParseConfig(d *schema.ResourceData) error {
	if v, found := d.GetOk(metricClusterDescriptionAttr); found {
		mc.Description = v.(string)
	}

	if v, found := d.GetOk(metricClusterNameAttr); found {
		mc.Name = v.(string)
	}

	if queryListRaw, found := d.GetOk(metricClusterQueryAttr); found {
		queryList := queryListRaw.(*schema.Set).List()

		mc.Queries = make([]api.MetricQuery, 0, len(queryList))

		for _, queryRaw := range queryList {
			queryAttrs := newInterfaceMap(queryRaw)

			var query string
			if v, found := queryAttrs[metricClusterDefinitionAttr]; found {
				query = v.(string)
			}

			var queryType string
			if v, found := queryAttrs[metricClusterTypeAttr]; found {
				queryType = v.(string)
			}

			mc.Queries = append(mc.Queries, api.MetricQuery{
				Query: query,
				Type:  queryType,
			})
		}
	}

	if v, found := d.GetOk(metricClusterTagsAttr); found {
		mc.Tags = derefStringList(flattenSet(v.(*schema.Set)))
	}

	if err := mc.Validate(); err != nil {
		return err
	}

	return nil
}
