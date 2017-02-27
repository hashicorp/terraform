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
	graphDescriptionAttr schemaAttr = "description"
	graphLeftAttr        schemaAttr = "left"
	graphLineStyleAttr   schemaAttr = "line_style"
	graphNameAttr        schemaAttr = "name"
	graphNotesAttr       schemaAttr = "notes"
	graphRightAttr       schemaAttr = "right"
	graphStreamAttr      schemaAttr = "stream"
	graphStreamGroupAttr schemaAttr = "stream_group"
	graphStyleAttr       schemaAttr = "graph_style"
	graphTagsAttr        schemaAttr = "tags"

	// circonus_graph.stream.* resource attribute names
	graphStreamActiveAttr        schemaAttr = "active"
	graphStreamAlphaAttr         schemaAttr = "alpha"
	graphStreamAxisAttr          schemaAttr = "axis"
	graphStreamCAQLAttr          schemaAttr = "caql"
	graphStreamCheckAttr         schemaAttr = "check"
	graphStreamColorAttr         schemaAttr = "color"
	graphStreamFormulaAttr       schemaAttr = "formula"
	graphStreamFormulaLegendAttr schemaAttr = "legend_formula"
	graphStreamFunctionAttr      schemaAttr = "function"
	graphStreamHumanNameAttr     schemaAttr = "name"
	graphStreamMetricTypeAttr    schemaAttr = "metric_type"
	graphStreamNameAttr          schemaAttr = "stream_name"
	graphStreamStackAttr         schemaAttr = "stack"

	// circonus_graph.stream_group.* resource attribute names
	graphStreamGroupActiveAttr    schemaAttr = "active"
	graphStreamGroupAggregateAttr schemaAttr = "aggregate"
	graphStreamGroupAxisAttr      schemaAttr = "axis"
	graphStreamGroupGroupAttr     schemaAttr = "group"
	graphStreamGroupHumanNameAttr schemaAttr = "name"

	// circonus_graph.{left,right}.* resource attribute names
	graphAxisLogarithmicAttr schemaAttr = "logarithmic"
	graphAxisMaxAttr         schemaAttr = "max"
	graphAxisMinAttr         schemaAttr = "min"
)

const (
	apiGraphStyleLine = "line"
)

var graphDescriptions = attrDescrs{
	// circonus_graph.* resource attribute names
	graphDescriptionAttr: "",
	graphLeftAttr:        "",
	graphLineStyleAttr:   "",
	graphNameAttr:        "",
	graphNotesAttr:       "",
	graphRightAttr:       "",
	graphStreamAttr:      "",
	graphStreamGroupAttr: "",
	graphStyleAttr:       "",
	graphTagsAttr:        "",
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
	// circonus_graph.stream_group.* resource attribute names
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

func newGraphResource() *schema.Resource {
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

		Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
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
					Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
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
					}, graphStreamDescriptions),
				},
			},
			graphStreamGroupAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
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
							ValidateFunc: validateStringIn(graphStreamGroupAttr, validAxisAttrs),
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
					}, graphStreamGroupDescriptions),
				},
			},
			graphStyleAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultGraphStyle,
				ValidateFunc: validateStringIn(graphStyleAttr, validGraphStyles),
			},
			graphTagsAttr: tagMakeConfigSchema(graphTagsAttr),
		}, graphDescriptions),
	}
}

func graphCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	g := newGraph()
	cr := newConfigReader(ctxt, d)
	if err := g.ParseConfig(cr); err != nil {
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
			panic(fmt.Sprintf("PROVIDER BUG: Unsupported axis type %q", datapoint.Axis))
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
			panic(fmt.Sprintf("PROVIDER BUG: Unsupported type for derive: %T", datapoint.Derive))
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
			panic(fmt.Sprintf("PROVIDER BUG: Unsupported axis type %q", metricCluster.Axis))
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

	stateSet(d, graphDescriptionAttr, g.Description)
	stateSet(d, graphLeftAttr, leftAxisMap)
	stateSet(d, graphLineStyleAttr, g.LineStyle)
	stateSet(d, graphNameAttr, g.Title)
	stateSet(d, graphNotesAttr, indirect(g.Notes))
	stateSet(d, graphRightAttr, rightAxisMap)
	stateSet(d, graphStreamAttr, streams)
	stateSet(d, graphStreamGroupAttr, streamGroups)
	stateSet(d, graphStyleAttr, g.Style)
	stateSet(d, graphTagsAttr, tagsToState(apiToTags(g.Tags)))

	d.SetId(g.CID)

	return nil
}

func graphUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*providerContext)
	g := newGraph()
	cr := newConfigReader(ctxt, d)
	if err := g.ParseConfig(cr); err != nil {
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
func (g *circonusGraph) ParseConfig(ar attrReader) error {
	g.Datapoints = make([]api.GraphDatapoint, 0, defaultGraphDatapoints)

	{
		leftAxisMap := ar.GetMap(graphLeftAttr)
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

	{
		rightAxisMap := ar.GetMap(graphRightAttr)
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

	if s, ok := ar.GetStringOK(graphDescriptionAttr); ok {
		g.Description = s
	}

	if p := ar.GetStringPtr(graphLineStyleAttr); p != nil {
		g.LineStyle = p
	}

	if s, ok := ar.GetStringOK(graphNameAttr); ok {
		g.Title = s
	}

	if s, ok := ar.GetStringOK(graphNotesAttr); ok {
		g.Notes = &s
	}

	if streamList, ok := ar.GetListOK(graphStreamAttr); ok {
		for _, streamListRaw := range streamList {
			for _, streamListElem := range streamListRaw.([]interface{}) {
				streamAttrs := newInterfaceMap(streamListElem.(map[string]interface{}))
				streamReader := newMapReader(ar.Context(), streamAttrs)
				datapoint := api.GraphDatapoint{}

				if b, ok := streamReader.GetBoolOK(graphStreamActiveAttr); ok {
					datapoint.Hidden = !b
				}

				if f, ok := streamReader.GetFloat64OK(graphStreamAlphaAttr); ok && f != 0 {
					datapoint.Alpha = &f
				}

				if s, ok := streamReader.GetStringOK(graphStreamAxisAttr); ok {
					switch s {
					case "left", "":
						datapoint.Axis = "l"
					case "right":
						datapoint.Axis = "r"
					default:
						panic(fmt.Sprintf("PROVIDER BUG: Unsupported axis attribute %q", s))
					}
				}

				if s, ok := streamReader.GetStringOK(graphStreamCheckAttr); ok {
					re := regexp.MustCompile(config.CheckCIDRegex)
					matches := re.FindStringSubmatch(s)
					if len(matches) == 3 {
						checkID, _ := strconv.ParseUint(matches[2], 10, 64)
						datapoint.CheckID = uint(checkID)
					}
				}

				if s, ok := streamReader.GetStringOK(graphStreamColorAttr); ok {
					datapoint.Color = &s
				}

				if s := streamReader.GetStringPtr(graphStreamFormulaAttr); s != nil {
					datapoint.DataFormula = s
				}

				if s, ok := streamReader.GetStringOK(graphStreamFunctionAttr); ok && s != "" {
					datapoint.Derive = s
				} else {
					datapoint.Derive = false
				}

				if s := streamReader.GetStringPtr(graphStreamFormulaLegendAttr); s != nil {
					datapoint.LegendFormula = s
				}

				if s, ok := streamReader.GetStringOK(graphStreamNameAttr); ok && s != "" {
					datapoint.MetricName = s
				}

				if s, ok := streamReader.GetStringOK(graphStreamMetricTypeAttr); ok && s != "" {
					datapoint.MetricType = s
				}

				if s, ok := streamReader.GetStringOK(graphStreamHumanNameAttr); ok && s != "" {
					datapoint.Name = s
				}

				if s := streamReader.GetStringPtr(graphStreamStackAttr); s != nil && *s != "" {
					u64, _ := strconv.ParseUint(*s, 10, 64)
					u := uint(u64)
					datapoint.Stack = &u
				}

				g.Datapoints = append(g.Datapoints, datapoint)
			}
		}
	}

	if streamGroupList, ok := ar.GetListOK(graphStreamGroupAttr); ok {
		for _, streamGroupListRaw := range streamGroupList {
			for _, streamGroupListElem := range streamGroupListRaw.([]interface{}) {
				streamGroupAttrs := newInterfaceMap(streamGroupListElem.(map[string]interface{}))
				streamGroupReader := newMapReader(ar.Context(), streamGroupAttrs)
				metricCluster := api.GraphMetricCluster{}

				if b, ok := streamGroupReader.GetBoolOK(graphStreamGroupActiveAttr); ok {
					metricCluster.Hidden = !b
				}

				if s, ok := streamGroupReader.GetStringOK(graphStreamGroupAggregateAttr); ok {
					metricCluster.AggregateFunc = s
				}

				if s, ok := streamGroupReader.GetStringOK(graphStreamGroupAxisAttr); ok {
					switch s {
					case "left", "":
						metricCluster.Axis = "l"
					case "right":
						metricCluster.Axis = "r"
					default:
						panic(fmt.Sprintf("PROVIDER BUG: Unsupported axis attribute %q", s))
					}
				}

				if s := streamGroupReader.GetStringPtr(graphStreamFormulaAttr); s != nil {
					metricCluster.DataFormula = s
				}

				if s := streamGroupReader.GetStringPtr(graphStreamFormulaLegendAttr); s != nil {
					metricCluster.LegendFormula = s
				}

				if s, ok := streamGroupReader.GetStringOK(graphStreamGroupGroupAttr); ok && s != "" {
					metricCluster.MetricCluster = s
				}

				if s, ok := streamGroupReader.GetStringOK(graphStreamHumanNameAttr); ok && s != "" {
					metricCluster.Name = s
				}

				if s := streamGroupReader.GetStringPtr(graphStreamStackAttr); s != nil && *s != "" {
					u64, _ := strconv.ParseUint(*s, 10, 64)
					u := uint(u64)
					metricCluster.Stack = &u
				}

				g.MetricClusters = append(g.MetricClusters, metricCluster)
			}
		}
	}

	if p := ar.GetStringPtr(graphStyleAttr); p != nil {
		g.Style = p
	}

	g.Tags = tagsToAPI(ar.GetTags(graphTagsAttr))

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
