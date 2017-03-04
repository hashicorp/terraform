package circonus

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_graph.* resource attribute names
	graphDescriptionAttr   = "description"
	graphLeftAttr          = "left"
	graphLineStyleAttr     = "line_style"
	graphMetricClusterAttr = "metric_cluster"
	graphNameAttr          = "name"
	graphNotesAttr         = "notes"
	graphRightAttr         = "right"
	graphStreamAttr        = "stream"
	graphStyleAttr         = "graph_style"
	graphTagsAttr          = "tags"

	// circonus_graph.stream.* resource attribute names
	graphStreamActiveAttr        = "active"
	graphStreamAlphaAttr         = "alpha"
	graphStreamAxisAttr          = "axis"
	graphStreamCAQLAttr          = "caql"
	graphStreamCheckAttr         = "check"
	graphStreamColorAttr         = "color"
	graphStreamFormulaAttr       = "formula"
	graphStreamFormulaLegendAttr = "legend_formula"
	graphStreamFunctionAttr      = "function"
	graphStreamHumanNameAttr     = "name"
	graphStreamMetricTypeAttr    = "metric_type"
	graphStreamNameAttr          = "stream_name"
	graphStreamStackAttr         = "stack"

	// circonus_graph.metric_cluster.* resource attribute names
	graphStreamGroupActiveAttr    = "active"
	graphStreamGroupAggregateAttr = "aggregate"
	graphStreamGroupAxisAttr      = "axis"
	graphStreamGroupGroupAttr     = "group"
	graphStreamGroupHumanNameAttr = "name"

	// circonus_graph.{left,right}.* resource attribute names
	graphAxisLogarithmicAttr = "logarithmic"
	graphAxisMaxAttr         = "max"
	graphAxisMinAttr         = "min"
)

const (
	apiGraphStyleLine = "line"
)

var graphDescriptions = attrDescrs{
	// circonus_graph.* resource attribute names
	graphDescriptionAttr:   "",
	graphLeftAttr:          "",
	graphLineStyleAttr:     "How the line should change between point. A string containing either 'stepped', 'interpolated' or null.",
	graphNameAttr:          "",
	graphNotesAttr:         "",
	graphRightAttr:         "",
	graphStreamAttr:        "",
	graphMetricClusterAttr: "",
	graphStyleAttr:         "",
	graphTagsAttr:          "",
}

var graphStreamDescriptions = attrDescrs{
	// circonus_graph.stream.* resource attribute names
	graphStreamActiveAttr:        "",
	graphStreamAlphaAttr:         "",
	graphStreamAxisAttr:          "",
	graphStreamCAQLAttr:          "",
	graphStreamCheckAttr:         "",
	graphStreamColorAttr:         "",
	graphStreamFormulaAttr:       "",
	graphStreamFormulaLegendAttr: "",
	graphStreamFunctionAttr:      "",
	graphStreamMetricTypeAttr:    "",
	graphStreamHumanNameAttr:     "",
	graphStreamNameAttr:          "",
	graphStreamStackAttr:         "",
}

var graphStreamGroupDescriptions = attrDescrs{
	// circonus_graph.metric_cluster.* resource attribute names
	graphStreamGroupActiveAttr:    "",
	graphStreamGroupAggregateAttr: "",
	graphStreamGroupAxisAttr:      "",
	graphStreamGroupGroupAttr:     "",
	graphStreamGroupHumanNameAttr: "",
}

var graphStreamAxisOptionDescriptions = attrDescrs{
	// circonus_graph.if.value.over.* resource attribute names
	graphAxisLogarithmicAttr: "",
	graphAxisMaxAttr:         "",
	graphAxisMinAttr:         "",
}

func resourceGraph() *schema.Resource {
	makeConflictsWith := func(in ...schemaAttr) []string {
		out := make([]string, 0, len(in))
		for _, attr := range in {
			out = append(out, string(graphStreamAttr)+"."+string(attr))
		}
		return out
	}

	return &schema.Resource{
		Create: graphCreate,
		Read:   graphRead,
		Update: graphUpdate,
		Delete: graphDelete,
		Exists: graphExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: convertToHelperSchema(graphDescriptions, map[schemaAttr]*schema.Schema{
			graphDescriptionAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: suppressWhitespace,
			},
			graphLeftAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Elem:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateGraphAxisOptions,
			},
			graphLineStyleAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultGraphLineStyle,
				ValidateFunc: validateStringIn(graphLineStyleAttr, validGraphLineStyles),
			},
			graphNameAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(graphNameAttr, `.+`),
			},
			graphNotesAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			graphRightAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Elem:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateGraphAxisOptions,
			},
			graphStreamAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(graphStreamDescriptions, map[schemaAttr]*schema.Schema{
						graphStreamActiveAttr: &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						graphStreamAlphaAttr: &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
							ValidateFunc: validateFuncs(
								validateFloatMin(graphStreamAlphaAttr, 0.0),
								validateFloatMax(graphStreamAlphaAttr, 1.0),
							),
						},
						graphStreamAxisAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "left",
							ValidateFunc: validateStringIn(graphStreamAxisAttr, validAxisAttrs),
						},
						graphStreamCAQLAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ValidateFunc:  validateRegexp(graphStreamCAQLAttr, `.+`),
							ConflictsWith: makeConflictsWith(graphStreamCheckAttr, graphStreamNameAttr),
						},
						graphStreamCheckAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ValidateFunc:  validateRegexp(graphStreamCheckAttr, config.CheckCIDRegex),
							ConflictsWith: makeConflictsWith(graphStreamCAQLAttr),
						},
						graphStreamColorAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRegexp(graphStreamColorAttr, `^#[0-9a-fA-F]{6}$`),
						},
						graphStreamFormulaAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRegexp(graphStreamFormulaAttr, `^.+$`),
						},
						graphStreamFormulaLegendAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRegexp(graphStreamFormulaLegendAttr, `^.+$`),
						},
						graphStreamFunctionAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultGraphFunction,
							ValidateFunc: validateStringIn(graphStreamFunctionAttr, validGraphFunctionValues),
						},
						graphStreamMetricTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateStringIn(graphStreamMetricTypeAttr, validMetricTypes),
						},
						graphStreamHumanNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRegexp(graphStreamHumanNameAttr, `.+`),
						},
						graphStreamNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRegexp(graphStreamNameAttr, `^[\S]+$`),
						},
						graphStreamStackAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRegexp(graphStreamStackAttr, `^[\d]*$`),
						},
					}),
				},
			},
			graphMetricClusterAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: convertToHelperSchema(graphStreamGroupDescriptions, map[schemaAttr]*schema.Schema{
						graphStreamGroupActiveAttr: &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						graphStreamGroupAggregateAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "none",
							ValidateFunc: validateStringIn(graphStreamGroupAggregateAttr, validAggregateFuncs),
						},
						graphStreamGroupAxisAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "left",
							ValidateFunc: validateStringIn(graphMetricClusterAttr, validAxisAttrs),
						},
						graphStreamGroupGroupAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRegexp(graphStreamGroupGroupAttr, config.MetricClusterCIDRegex),
						},
						graphStreamGroupHumanNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateRegexp(graphStreamHumanNameAttr, `.+`),
						},
					}),
				},
			},
			graphStyleAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultGraphStyle,
				ValidateFunc: validateStringIn(graphStyleAttr, validGraphStyles),
			},
			graphTagsAttr: tagMakeConfigSchema(graphTagsAttr),
		}),
	}
}

func graphCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	g := newGraph()
	if err := g.ParseConfig(d); err != nil {
		return errwrap.Wrapf("error parsing graph schema during create: {{err}}", err)
	}

	if err := g.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating graph: {{err}}", err)
	}

	d.SetId(g.CID)

	return graphRead(d, meta)
}

func graphExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	g, err := ctxt.client.FetchGraph(api.CIDType(&cid))
	if err != nil {
		if strings.Contains(err.Error(), defaultCirconus404ErrorString) {
			return false, nil
		}

		return false, err
	}

	if g.CID == "" {
		return false, nil
	}

	return true, nil
}

// graphRead pulls data out of the Graph object and stores it into the
// appropriate place in the statefile.
func graphRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	g, err := loadGraph(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	d.SetId(g.CID)

	streams := make([]interface{}, 0, len(g.Datapoints))
	for _, datapoint := range g.Datapoints {
		dataPointAttrs := make(map[string]interface{}, 13) // 13 == len(members in api.GraphDatapoint)

		dataPointAttrs[string(graphStreamActiveAttr)] = !datapoint.Hidden

		if datapoint.Alpha != nil && *datapoint.Alpha != 0 {
			dataPointAttrs[string(graphStreamAlphaAttr)] = *datapoint.Alpha
		}

		switch datapoint.Axis {
		case "l", "":
			dataPointAttrs[string(graphStreamAxisAttr)] = "left"
		case "r":
			dataPointAttrs[string(graphStreamAxisAttr)] = "right"
		default:
			return fmt.Errorf("PROVIDER BUG: Unsupported axis type %q", datapoint.Axis)
		}

		if datapoint.CAQL != nil {
			dataPointAttrs[string(graphStreamCAQLAttr)] = *datapoint.CAQL
		}

		if datapoint.CheckID != 0 {
			dataPointAttrs[string(graphStreamCheckAttr)] = fmt.Sprintf("%s/%d", config.CheckPrefix, datapoint.CheckID)
		}

		if datapoint.Color != nil {
			dataPointAttrs[string(graphStreamColorAttr)] = *datapoint.Color
		}

		if datapoint.DataFormula != nil {
			dataPointAttrs[string(graphStreamFormulaAttr)] = *datapoint.DataFormula
		}

		switch datapoint.Derive.(type) {
		case bool:
		case string:
			dataPointAttrs[string(graphStreamFunctionAttr)] = datapoint.Derive.(string)
		default:
			return fmt.Errorf("PROVIDER BUG: Unsupported type for derive: %T", datapoint.Derive)
		}

		if datapoint.LegendFormula != nil {
			dataPointAttrs[string(graphStreamFormulaLegendAttr)] = *datapoint.LegendFormula
		}

		if datapoint.MetricName != "" {
			dataPointAttrs[string(graphStreamNameAttr)] = datapoint.MetricName
		}

		if datapoint.MetricType != "" {
			dataPointAttrs[string(graphStreamMetricTypeAttr)] = datapoint.MetricType
		}

		if datapoint.Name != "" {
			dataPointAttrs[string(graphStreamHumanNameAttr)] = datapoint.Name
		}

		if datapoint.Stack != nil {
			dataPointAttrs[string(graphStreamStackAttr)] = fmt.Sprintf("%d", *datapoint.Stack)
		}

		streams = append(streams, dataPointAttrs)
	}

	streamGroups := make([]interface{}, 0, len(g.MetricClusters))
	for _, metricCluster := range g.MetricClusters {
		streamGroupAttrs := make(map[string]interface{}, 8) // 8 == len(num struct attrs in api.GraphMetricCluster)

		streamGroupAttrs[string(graphStreamGroupActiveAttr)] = !metricCluster.Hidden

		if metricCluster.AggregateFunc != "" {
			streamGroupAttrs[string(graphStreamGroupAggregateAttr)] = metricCluster.AggregateFunc
		}

		switch metricCluster.Axis {
		case "l", "":
			streamGroupAttrs[string(graphStreamGroupAxisAttr)] = "left"
		case "r":
			streamGroupAttrs[string(graphStreamGroupAxisAttr)] = "right"
		default:
			return fmt.Errorf("PROVIDER BUG: Unsupported axis type %q", metricCluster.Axis)
		}

		if metricCluster.DataFormula != nil {
			streamGroupAttrs[string(graphStreamFormulaAttr)] = *metricCluster.DataFormula
		}

		if metricCluster.LegendFormula != nil {
			streamGroupAttrs[string(graphStreamFormulaLegendAttr)] = *metricCluster.LegendFormula
		}

		if metricCluster.MetricCluster != "" {
			streamGroupAttrs[string(graphStreamGroupGroupAttr)] = metricCluster.MetricCluster
		}

		if metricCluster.Name != "" {
			streamGroupAttrs[string(graphStreamHumanNameAttr)] = metricCluster.Name
		}

		if metricCluster.Stack != nil {
			streamGroupAttrs[string(graphStreamStackAttr)] = fmt.Sprintf("%d", *metricCluster.Stack)
		}

		streamGroups = append(streamGroups, streamGroupAttrs)
	}

	leftAxisMap := make(map[string]interface{}, 3)
	if g.LogLeftY != nil {
		leftAxisMap[string(graphAxisLogarithmicAttr)] = fmt.Sprintf("%d", *g.LogLeftY)
	}

	if g.MaxLeftY != nil {
		leftAxisMap[string(graphAxisMaxAttr)] = strconv.FormatFloat(*g.MaxLeftY, 'f', -1, 64)
	}

	if g.MinLeftY != nil {
		leftAxisMap[string(graphAxisMinAttr)] = strconv.FormatFloat(*g.MinLeftY, 'f', -1, 64)
	}

	rightAxisMap := make(map[string]interface{}, 3)
	if g.LogRightY != nil {
		rightAxisMap[string(graphAxisLogarithmicAttr)] = fmt.Sprintf("%d", *g.LogRightY)
	}

	if g.MaxRightY != nil {
		rightAxisMap[string(graphAxisMaxAttr)] = strconv.FormatFloat(*g.MaxRightY, 'f', -1, 64)
	}

	if g.MinRightY != nil {
		rightAxisMap[string(graphAxisMinAttr)] = strconv.FormatFloat(*g.MinRightY, 'f', -1, 64)
	}

	d.Set(graphDescriptionAttr, g.Description)

	if err := d.Set(graphLeftAttr, leftAxisMap); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store graph %q attribute: {{err}}", graphLeftAttr), err)
	}

	d.Set(graphLineStyleAttr, g.LineStyle)
	d.Set(graphNameAttr, g.Title)
	d.Set(graphNotesAttr, indirect(g.Notes))

	if err := d.Set(graphRightAttr, rightAxisMap); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store graph %q attribute: {{err}}", graphRightAttr), err)
	}

	if err := d.Set(graphStreamAttr, streams); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store graph %q attribute: {{err}}", graphStreamAttr), err)
	}

	if err := d.Set(graphMetricClusterAttr, streamGroups); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store graph %q attribute: {{err}}", graphMetricClusterAttr), err)
	}

	d.Set(graphStyleAttr, g.Style)

	if err := d.Set(graphTagsAttr, tagsToState(apiToTags(g.Tags))); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store graph %q attribute: {{err}}", graphTagsAttr), err)
	}

	return nil
}

func graphUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	g := newGraph()
	if err := g.ParseConfig(d); err != nil {
		return err
	}

	g.CID = d.Id()
	if err := g.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update graph %q: {{err}}", d.Id()), err)
	}

	return graphRead(d, meta)
}

func graphDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)

	cid := d.Id()
	if _, err := ctxt.client.DeleteGraphByCID(api.CIDType(&cid)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete graph %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

type circonusGraph struct {
	api.Graph
}

func newGraph() circonusGraph {
	g := circonusGraph{
		Graph: *api.NewGraph(),
	}

	return g
}

func loadGraph(ctxt *providerContext, cid api.CIDType) (circonusGraph, error) {
	var g circonusGraph
	ng, err := ctxt.client.FetchGraph(cid)
	if err != nil {
		return circonusGraph{}, err
	}
	g.Graph = *ng

	return g, nil
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus Graph object.  ParseConfig and graphRead() must be kept in sync.
func (g *circonusGraph) ParseConfig(d *schema.ResourceData) error {
	g.Datapoints = make([]api.GraphDatapoint, 0, defaultGraphDatapoints)

	if v, found := d.GetOk(graphLeftAttr); found {
		listRaw := v.(map[string]interface{})
		leftAxisMap := make(map[string]interface{}, len(listRaw))
		for k, v := range listRaw {
			leftAxisMap[k] = v
		}

		if v, ok := leftAxisMap[string(graphAxisLogarithmicAttr)]; ok {
			i64, _ := strconv.ParseInt(v.(string), 10, 64)
			i := int(i64)
			g.LogLeftY = &i
		}

		if v, ok := leftAxisMap[string(graphAxisMaxAttr)]; ok && v.(string) != "" {
			f, _ := strconv.ParseFloat(v.(string), 64)
			g.MaxLeftY = &f
		}

		if v, ok := leftAxisMap[string(graphAxisMinAttr)]; ok && v.(string) != "" {
			f, _ := strconv.ParseFloat(v.(string), 64)
			g.MinLeftY = &f
		}
	}

	if v, found := d.GetOk(graphRightAttr); found {
		listRaw := v.(map[string]interface{})
		rightAxisMap := make(map[string]interface{}, len(listRaw))
		for k, v := range listRaw {
			rightAxisMap[k] = v
		}

		if v, ok := rightAxisMap[string(graphAxisLogarithmicAttr)]; ok {
			i64, _ := strconv.ParseInt(v.(string), 10, 64)
			i := int(i64)
			g.LogRightY = &i
		}

		if v, ok := rightAxisMap[string(graphAxisMaxAttr)]; ok && v.(string) != "" {
			f, _ := strconv.ParseFloat(v.(string), 64)
			g.MaxRightY = &f
		}

		if v, ok := rightAxisMap[string(graphAxisMinAttr)]; ok && v.(string) != "" {
			f, _ := strconv.ParseFloat(v.(string), 64)
			g.MinRightY = &f
		}
	}

	if v, found := d.GetOk(graphDescriptionAttr); found {
		g.Description = v.(string)
	}

	if v, found := d.GetOk(graphLineStyleAttr); found {
		switch v.(type) {
		case string:
			s := v.(string)
			g.LineStyle = &s
		case *string:
			g.LineStyle = v.(*string)
		default:
			return fmt.Errorf("PROVIDER BUG: unsupported type for %q: %T", graphLineStyleAttr, v)
		}
	}

	if v, found := d.GetOk(graphNameAttr); found {
		g.Title = v.(string)
	}

	if v, found := d.GetOk(graphNotesAttr); found {
		s := v.(string)
		g.Notes = &s
	}

	if listRaw, found := d.GetOk(graphStreamAttr); found {
		streamList := listRaw.([]interface{})
		for _, streamListElem := range streamList {
			streamAttrs := newInterfaceMap(streamListElem.(map[string]interface{}))
			datapoint := api.GraphDatapoint{}

			if v, found := streamAttrs[graphStreamActiveAttr]; found {
				datapoint.Hidden = !(v.(bool))
			}

			if v, found := streamAttrs[graphStreamAlphaAttr]; found {
				f := v.(float64)
				if f != 0 {
					datapoint.Alpha = &f
				}
			}

			if v, found := streamAttrs[graphStreamAxisAttr]; found {
				switch v.(string) {
				case "left", "":
					datapoint.Axis = "l"
				case "right":
					datapoint.Axis = "r"
				default:
					return fmt.Errorf("PROVIDER BUG: Unsupported axis attribute %q: %q", graphStreamAxisAttr, v.(string))
				}
			}

			if v, found := streamAttrs[graphStreamCheckAttr]; found {
				re := regexp.MustCompile(config.CheckCIDRegex)
				matches := re.FindStringSubmatch(v.(string))
				if len(matches) == 3 {
					checkID, _ := strconv.ParseUint(matches[2], 10, 64)
					datapoint.CheckID = uint(checkID)
				}
			}

			if v, found := streamAttrs[graphStreamColorAttr]; found {
				s := v.(string)
				datapoint.Color = &s
			}

			if v, found := streamAttrs[graphStreamFormulaAttr]; found {
				switch v.(type) {
				case string:
					s := v.(string)
					datapoint.DataFormula = &s
				case *string:
					datapoint.DataFormula = v.(*string)
				default:
					return fmt.Errorf("PROVIDER BUG: unsupported type for %q: %T", graphStreamAttr, v)
				}
			}

			if v, found := streamAttrs[graphStreamFunctionAttr]; found {
				s := v.(string)
				if s != "" {
					datapoint.Derive = s
				} else {
					datapoint.Derive = false
				}
			} else {
				datapoint.Derive = false
			}

			if v, found := streamAttrs[graphStreamFormulaLegendAttr]; found {
				switch u := v.(type) {
				case string:
					datapoint.LegendFormula = &u
				case *string:
					datapoint.LegendFormula = u
				default:
					return fmt.Errorf("PROVIDER BUG: unsupported type for %q: %T", graphStreamAttr, v)
				}
			}

			if v, found := streamAttrs[graphStreamNameAttr]; found {
				s := v.(string)
				if s != "" {
					datapoint.MetricName = s
				}
			}

			if v, found := streamAttrs[graphStreamMetricTypeAttr]; found {
				s := v.(string)
				if s != "" {
					datapoint.MetricType = s
				}
			}

			if v, found := streamAttrs[graphStreamHumanNameAttr]; found {
				s := v.(string)
				if s != "" {
					datapoint.Name = s
				}
			}

			if v, found := streamAttrs[graphStreamStackAttr]; found {
				var stackStr string
				switch u := v.(type) {
				case string:
					stackStr = u
				case *string:
					if u != nil {
						stackStr = *u
					}
				default:
					return fmt.Errorf("PROVIDER BUG: unsupported type for %q: %T", graphStreamStackAttr, v)
				}

				if stackStr != "" {
					u64, _ := strconv.ParseUint(stackStr, 10, 64)
					u := uint(u64)
					datapoint.Stack = &u
				}
			}

			g.Datapoints = append(g.Datapoints, datapoint)
		}
	}

	if listRaw, found := d.GetOk(graphMetricClusterAttr); found {
		streamGroupList := listRaw.([]interface{})

		for _, streamGroupListRaw := range streamGroupList {
			for _, streamGroupListElem := range streamGroupListRaw.([]interface{}) {
				streamGroupAttrs := newInterfaceMap(streamGroupListElem.(map[string]interface{}))

				metricCluster := api.GraphMetricCluster{}

				if v, found := streamGroupAttrs[graphStreamGroupActiveAttr]; found {
					metricCluster.Hidden = !(v.(bool))
				}

				if v, found := streamGroupAttrs[graphStreamGroupAggregateAttr]; found {
					metricCluster.AggregateFunc = v.(string)
				}

				if v, found := streamGroupAttrs[graphStreamGroupAxisAttr]; found {
					switch v.(string) {
					case "left", "":
						metricCluster.Axis = "l"
					case "right":
						metricCluster.Axis = "r"
					default:
						return fmt.Errorf("PROVIDER BUG: Unsupported axis attribute %q: %q", graphStreamGroupAxisAttr, v.(string))
					}
				}

				if v, found := streamGroupAttrs[graphStreamFormulaAttr]; found {
					switch v.(type) {
					case string:
						s := v.(string)
						metricCluster.DataFormula = &s
					case *string:
						metricCluster.DataFormula = v.(*string)
					default:
						return fmt.Errorf("PROVIDER BUG: unsupported type for %q: %T", graphStreamFormulaAttr, v)
					}
				}

				if v, found := streamGroupAttrs[graphStreamFormulaLegendAttr]; found {
					switch v.(type) {
					case string:
						s := v.(string)
						metricCluster.LegendFormula = &s
					case *string:
						metricCluster.LegendFormula = v.(*string)
					default:
						return fmt.Errorf("PROVIDER BUG: unsupported type for %q: %T", graphStreamFormulaLegendAttr, v)
					}
				}

				if v, found := streamGroupAttrs[graphStreamGroupGroupAttr]; found {
					s := v.(string)
					if s != "" {
						metricCluster.MetricCluster = s
					}
				}

				if v, found := streamGroupAttrs[graphStreamHumanNameAttr]; found {
					s := v.(string)
					if s != "" {
						metricCluster.Name = s
					}
				}

				if v, found := streamGroupAttrs[graphStreamStackAttr]; found {
					var stackStr string
					switch u := v.(type) {
					case string:
						stackStr = u
					case *string:
						if u != nil {
							stackStr = *u
						}
					default:
						return fmt.Errorf("PROVIDER BUG: unsupported type for %q: %T", graphStreamStackAttr, v)
					}

					if stackStr != "" {
						u64, _ := strconv.ParseUint(stackStr, 10, 64)
						u := uint(u64)
						metricCluster.Stack = &u
					}
				}

				g.MetricClusters = append(g.MetricClusters, metricCluster)
			}
		}
	}

	if v, found := d.GetOk(graphStyleAttr); found {
		switch v.(type) {
		case string:
			s := v.(string)
			g.Style = &s
		case *string:
			g.Style = v.(*string)
		default:
			return fmt.Errorf("PROVIDER BUG: unsupported type for %q: %T", graphStyleAttr, v)
		}
	}

	if v, found := d.GetOk(graphTagsAttr); found {
		g.Tags = derefStringList(flattenSet(v.(*schema.Set)))
	}

	if err := g.Validate(); err != nil {
		return err
	}

	return nil
}

func (g *circonusGraph) Create(ctxt *providerContext) error {
	ng, err := ctxt.client.CreateGraph(&g.Graph)
	if err != nil {
		return err
	}

	g.CID = ng.CID

	return nil
}

func (g *circonusGraph) Update(ctxt *providerContext) error {
	_, err := ctxt.client.UpdateGraph(&g.Graph)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update graph %s: {{err}}", g.CID), err)
	}

	return nil
}

func (g *circonusGraph) Validate() error {
	for i, datapoint := range g.Datapoints {
		if *g.Style == apiGraphStyleLine && datapoint.Alpha != nil && *datapoint.Alpha != 0 {
			return fmt.Errorf("%s can not be set on graphs with style %s", graphStreamAlphaAttr, apiGraphStyleLine)
		}

		if datapoint.CheckID != 0 && datapoint.MetricName == "" {
			return fmt.Errorf("Error with stream[%d] name=%q: %s is set, missing attribute %s must also be set", i, datapoint.Name, graphStreamCheckAttr, graphStreamNameAttr)
		}

		if datapoint.CheckID == 0 && datapoint.MetricName != "" {
			return fmt.Errorf("Error with stream[%d] name=%q: %s is set, missing attribute %s must also be set", i, datapoint.Name, graphStreamNameAttr, graphStreamCheckAttr)
		}

		if datapoint.CAQL != nil && (datapoint.CheckID != 0 || datapoint.MetricName != "") {
			return fmt.Errorf("Error with stream[%d] name=%q: %q attribute is mutually exclusive with attributes %s or %s", i, datapoint.Name, graphStreamCAQLAttr, graphStreamNameAttr, graphStreamCheckAttr)
		}
	}

	return nil
}
