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
	_GraphDescriptionAttr _SchemaAttr = "description"
	_GraphLeftAttr        _SchemaAttr = "left"
	_GraphLineStyleAttr   _SchemaAttr = "line_style"
	_GraphNameAttr        _SchemaAttr = "name"
	_GraphNotesAttr       _SchemaAttr = "notes"
	_GraphRightAttr       _SchemaAttr = "right"
	_GraphStreamAttr      _SchemaAttr = "stream"
	_GraphStreamGroupAttr _SchemaAttr = "stream_group"
	_GraphStyleAttr       _SchemaAttr = "graph_style"
	_GraphTagsAttr        _SchemaAttr = "tags"

	// circonus_graph.stream.* resource attribute names
	_GraphStreamActiveAttr        _SchemaAttr = "active"
	_GraphStreamAlphaAttr         _SchemaAttr = "alpha"
	_GraphStreamAxisAttr          _SchemaAttr = "axis"
	_GraphStreamCAQLAttr          _SchemaAttr = "caql"
	_GraphStreamCheckAttr         _SchemaAttr = "check"
	_GraphStreamColorAttr         _SchemaAttr = "color"
	_GraphStreamFormulaAttr       _SchemaAttr = "formula"
	_GraphStreamFormulaLegendAttr _SchemaAttr = "legend_formula"
	_GraphStreamFunctionAttr      _SchemaAttr = "function"
	_GraphStreamHumanNameAttr     _SchemaAttr = "name"
	_GraphStreamMetricTypeAttr    _SchemaAttr = "metric_type"
	_GraphStreamNameAttr          _SchemaAttr = "stream_name"
	_GraphStreamStackAttr         _SchemaAttr = "stack"

	// circonus_graph.stream_group.* resource attribute names
	_GraphStreamGroupActiveAttr    _SchemaAttr = "active"
	_GraphStreamGroupAggregateAttr _SchemaAttr = "aggregate"
	_GraphStreamGroupAxisAttr      _SchemaAttr = "axis"
	_GraphStreamGroupGroupAttr     _SchemaAttr = "group"
	_GraphStreamGroupHumanNameAttr _SchemaAttr = "name"

	// circonus_graph.{left,right}.* resource attribute names
	_GraphAxisLogarithmicAttr _SchemaAttr = "logarithmic"
	_GraphAxisMaxAttr         _SchemaAttr = "max"
	_GraphAxisMinAttr         _SchemaAttr = "min"
)

const (
	_APIGraphStyleLine = "line"
)

var _GraphDescriptions = _AttrDescrs{
	// circonus_graph.* resource attribute names
	_GraphDescriptionAttr: "",
	_GraphLeftAttr:        "",
	_GraphLineStyleAttr:   "",
	_GraphNameAttr:        "",
	_GraphNotesAttr:       "",
	_GraphRightAttr:       "",
	_GraphStreamAttr:      "",
	_GraphStreamGroupAttr: "",
	_GraphStyleAttr:       "",
	_GraphTagsAttr:        "",
}

var _GraphStreamDescriptions = _AttrDescrs{
	// circonus_graph.stream.* resource attribute names
	_GraphStreamActiveAttr:        "",
	_GraphStreamAlphaAttr:         "",
	_GraphStreamAxisAttr:          "",
	_GraphStreamCAQLAttr:          "",
	_GraphStreamCheckAttr:         "",
	_GraphStreamColorAttr:         "",
	_GraphStreamFormulaAttr:       "",
	_GraphStreamFormulaLegendAttr: "",
	_GraphStreamFunctionAttr:      "",
	_GraphStreamMetricTypeAttr:    "",
	_GraphStreamHumanNameAttr:     "",
	_GraphStreamNameAttr:          "",
	_GraphStreamStackAttr:         "",
}

var _GraphStreamGroupDescriptions = _AttrDescrs{
	// circonus_graph.stream_group.* resource attribute names
	_GraphStreamGroupActiveAttr:    "",
	_GraphStreamGroupAggregateAttr: "",
	_GraphStreamGroupAxisAttr:      "",
	_GraphStreamGroupGroupAttr:     "",
	_GraphStreamGroupHumanNameAttr: "",
}

var _GraphStreamAxisOptionDescriptions = _AttrDescrs{
	// circonus_graph.if.value.over.* resource attribute names
	_GraphAxisLogarithmicAttr: "",
	_GraphAxisMaxAttr:         "",
	_GraphAxisMinAttr:         "",
}

func _NewGraphResource() *schema.Resource {
	makeConflictsWith := func(in ..._SchemaAttr) []string {
		out := make([]string, 0, len(in))
		for _, attr := range in {
			out = append(out, string(_GraphStreamAttr)+"."+string(attr))
		}
		return out
	}

	return &schema.Resource{
		Create: _GraphCreate,
		Read:   _GraphRead,
		Update: _GraphUpdate,
		Delete: _GraphDelete,
		Exists: _GraphExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_GraphDescriptionAttr: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				StateFunc: suppressWhitespace,
			},
			_GraphLeftAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Elem:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateGraphAxisOptions,
			},
			_GraphLineStyleAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultGraphLineStyle,
				ValidateFunc: _ValidateStringIn(_GraphLineStyleAttr, _ValidGraphLineStyles),
			},
			_GraphNameAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: _ValidateRegexp(_GraphNameAttr, `.+`),
			},
			_GraphNotesAttr: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			_GraphRightAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Elem:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateGraphAxisOptions,
			},
			_GraphStreamAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
						_GraphStreamActiveAttr: &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						_GraphStreamAlphaAttr: &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
							ValidateFunc: _ValidateFuncs(
								_ValidateFloatMin(_GraphStreamAlphaAttr, 0.0),
								_ValidateFloatMax(_GraphStreamAlphaAttr, 1.0),
							),
						},
						_GraphStreamAxisAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "left",
							ValidateFunc: _ValidateStringIn(_GraphStreamAxisAttr, _ValidAxisAttrs),
						},
						_GraphStreamCAQLAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ValidateFunc:  _ValidateRegexp(_GraphStreamCAQLAttr, `.+`),
							ConflictsWith: makeConflictsWith(_GraphStreamCheckAttr, _GraphStreamNameAttr),
						},
						_GraphStreamCheckAttr: &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ValidateFunc:  _ValidateRegexp(_GraphStreamCheckAttr, config.CheckCIDRegex),
							ConflictsWith: makeConflictsWith(_GraphStreamCAQLAttr),
						},
						_GraphStreamColorAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamColorAttr, `^#[0-9a-fA-F]{6}$`),
						},
						_GraphStreamFormulaAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamFormulaAttr, `^.+$`),
						},
						_GraphStreamFormulaLegendAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamFormulaLegendAttr, `^.+$`),
						},
						_GraphStreamFunctionAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultGraphFunction,
							ValidateFunc: _ValidateStringIn(_GraphStreamFunctionAttr, _ValidGraphFunctionValues),
						},
						_GraphStreamMetricTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: _ValidateStringIn(_GraphStreamMetricTypeAttr, _ValidMetricTypes),
						},
						_GraphStreamHumanNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamHumanNameAttr, `.+`),
						},
						_GraphStreamNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamNameAttr, `^[\S]+$`),
						},
						_GraphStreamStackAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamStackAttr, `^[\d]*$`),
						},
					}, _GraphStreamDescriptions),
				},
			},
			_GraphStreamGroupAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
						_GraphStreamGroupActiveAttr: &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						_GraphStreamGroupAggregateAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "none",
							ValidateFunc: _ValidateStringIn(_GraphStreamGroupAggregateAttr, _ValidAggregateFuncs),
						},
						_GraphStreamGroupAxisAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "left",
							ValidateFunc: _ValidateStringIn(_GraphStreamGroupAttr, _ValidAxisAttrs),
						},
						_GraphStreamGroupGroupAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamGroupGroupAttr, config.MetricClusterCIDRegex),
						},
						_GraphStreamGroupHumanNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamHumanNameAttr, `.+`),
						},
					}, _GraphStreamGroupDescriptions),
				},
			},
			_GraphStyleAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultGraphStyle,
				ValidateFunc: _ValidateStringIn(_GraphStyleAttr, _ValidGraphStyles),
			},
			_GraphTagsAttr: _TagMakeConfigSchema(_GraphTagsAttr),
		}, _GraphDescriptions),
	}
}

func _GraphCreate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)
	g := _NewGraph()
	cr := _NewConfigReader(ctxt, d)
	if err := g.ParseConfig(cr); err != nil {
		return errwrap.Wrapf("error parsing graph schema during create: {{err}}", err)
	}

	if err := g.Create(ctxt); err != nil {
		return errwrap.Wrapf("error creating graph: {{err}}", err)
	}

	d.SetId(g.CID)

	return _GraphRead(d, meta)
}

func _GraphExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctxt := meta.(*_ProviderContext)

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

// _GraphRead pulls data out of the Graph object and stores it into the
// appropriate place in the statefile.
func _GraphRead(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	g, err := _LoadGraph(ctxt, api.CIDType(&cid))
	if err != nil {
		return err
	}

	streams := make([]interface{}, 0, len(g.Datapoints))
	for _, datapoint := range g.Datapoints {
		dataPointAttrs := make(map[string]interface{}, 13) // 13 == len(members in api.GraphDatapoint)

		dataPointAttrs[string(_GraphStreamActiveAttr)] = !datapoint.Hidden

		if datapoint.Alpha != nil && *datapoint.Alpha != 0 {
			dataPointAttrs[string(_GraphStreamAlphaAttr)] = *datapoint.Alpha
		}

		switch datapoint.Axis {
		case "l", "":
			dataPointAttrs[string(_GraphStreamAxisAttr)] = "left"
		case "r":
			dataPointAttrs[string(_GraphStreamAxisAttr)] = "right"
		default:
			panic(fmt.Sprintf("PROVIDER BUG: Unsupported axis type %q", datapoint.Axis))
		}

		if datapoint.CAQL != nil {
			dataPointAttrs[string(_GraphStreamCAQLAttr)] = *datapoint.CAQL
		}

		if datapoint.CheckID != 0 {
			dataPointAttrs[string(_GraphStreamCheckAttr)] = fmt.Sprintf("%s/%d", config.CheckPrefix, datapoint.CheckID)
		}

		if datapoint.Color != nil {
			dataPointAttrs[string(_GraphStreamColorAttr)] = *datapoint.Color
		}

		if datapoint.DataFormula != nil {
			dataPointAttrs[string(_GraphStreamFormulaAttr)] = *datapoint.DataFormula
		}

		switch datapoint.Derive.(type) {
		case bool:
		case string:
			dataPointAttrs[string(_GraphStreamFunctionAttr)] = datapoint.Derive.(string)
		default:
			panic(fmt.Sprintf("PROVIDER BUG: Unsupported type for derive: %T", datapoint.Derive))
		}

		if datapoint.LegendFormula != nil {
			dataPointAttrs[string(_GraphStreamFormulaLegendAttr)] = *datapoint.LegendFormula
		}

		if datapoint.MetricName != "" {
			dataPointAttrs[string(_GraphStreamNameAttr)] = datapoint.MetricName
		}

		if datapoint.MetricType != "" {
			dataPointAttrs[string(_GraphStreamMetricTypeAttr)] = datapoint.MetricType
		}

		if datapoint.Name != "" {
			dataPointAttrs[string(_GraphStreamHumanNameAttr)] = datapoint.Name
		}

		if datapoint.Stack != nil {
			dataPointAttrs[string(_GraphStreamStackAttr)] = fmt.Sprintf("%d", *datapoint.Stack)
		}

		streams = append(streams, dataPointAttrs)
	}

	streamGroups := make([]interface{}, 0, len(g.MetricClusters))
	for _, metricCluster := range g.MetricClusters {
		streamGroupAttrs := make(map[string]interface{}, 8) // 8 == len(num struct attrs in api.GraphMetricCluster)

		streamGroupAttrs[string(_GraphStreamGroupActiveAttr)] = !metricCluster.Hidden

		if metricCluster.AggregateFunc != "" {
			streamGroupAttrs[string(_GraphStreamGroupAggregateAttr)] = metricCluster.AggregateFunc
		}

		switch metricCluster.Axis {
		case "l", "":
			streamGroupAttrs[string(_GraphStreamGroupAxisAttr)] = "left"
		case "r":
			streamGroupAttrs[string(_GraphStreamGroupAxisAttr)] = "right"
		default:
			panic(fmt.Sprintf("PROVIDER BUG: Unsupported axis type %q", metricCluster.Axis))
		}

		if metricCluster.DataFormula != nil {
			streamGroupAttrs[string(_GraphStreamFormulaAttr)] = *metricCluster.DataFormula
		}

		if metricCluster.LegendFormula != nil {
			streamGroupAttrs[string(_GraphStreamFormulaLegendAttr)] = *metricCluster.LegendFormula
		}

		if metricCluster.MetricCluster != "" {
			streamGroupAttrs[string(_GraphStreamGroupGroupAttr)] = metricCluster.MetricCluster
		}

		if metricCluster.Name != "" {
			streamGroupAttrs[string(_GraphStreamHumanNameAttr)] = metricCluster.Name
		}

		if metricCluster.Stack != nil {
			streamGroupAttrs[string(_GraphStreamStackAttr)] = fmt.Sprintf("%d", *metricCluster.Stack)
		}

		streamGroups = append(streamGroups, streamGroupAttrs)
	}

	leftAxisMap := make(map[string]interface{}, 3)
	if g.LogLeftY != nil {
		leftAxisMap[string(_GraphAxisLogarithmicAttr)] = fmt.Sprintf("%d", *g.LogLeftY)
	}

	if g.MaxLeftY != nil {
		leftAxisMap[string(_GraphAxisMaxAttr)] = strconv.FormatFloat(*g.MaxLeftY, 'f', -1, 64)
	}

	if g.MinLeftY != nil {
		leftAxisMap[string(_GraphAxisMinAttr)] = strconv.FormatFloat(*g.MinLeftY, 'f', -1, 64)
	}

	rightAxisMap := make(map[string]interface{}, 3)
	if g.LogRightY != nil {
		rightAxisMap[string(_GraphAxisLogarithmicAttr)] = fmt.Sprintf("%d", *g.LogRightY)
	}

	if g.MaxRightY != nil {
		rightAxisMap[string(_GraphAxisMaxAttr)] = strconv.FormatFloat(*g.MaxRightY, 'f', -1, 64)
	}

	if g.MinRightY != nil {
		rightAxisMap[string(_GraphAxisMinAttr)] = strconv.FormatFloat(*g.MinRightY, 'f', -1, 64)
	}

	_StateSet(d, _GraphDescriptionAttr, g.Description)
	_StateSet(d, _GraphLeftAttr, leftAxisMap)
	_StateSet(d, _GraphLineStyleAttr, g.LineStyle)
	_StateSet(d, _GraphNameAttr, g.Title)
	_StateSet(d, _GraphNotesAttr, _Indirect(g.Notes))
	_StateSet(d, _GraphRightAttr, rightAxisMap)
	_StateSet(d, _GraphStreamAttr, streams)
	_StateSet(d, _GraphStreamGroupAttr, streamGroups)
	_StateSet(d, _GraphStyleAttr, g.Style)
	_StateSet(d, _GraphTagsAttr, tagsToState(apiToTags(g.Tags)))

	d.SetId(g.CID)

	return nil
}

func _GraphUpdate(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)
	g := _NewGraph()
	cr := _NewConfigReader(ctxt, d)
	if err := g.ParseConfig(cr); err != nil {
		return err
	}

	g.CID = d.Id()
	if err := g.Update(ctxt); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to update graph %q: {{err}}", d.Id()), err)
	}

	return _GraphRead(d, meta)
}

func _GraphDelete(d *schema.ResourceData, meta interface{}) error {
	ctxt := meta.(*_ProviderContext)

	cid := d.Id()
	if _, err := ctxt.client.DeleteGraphByCID(api.CIDType(&cid)); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("unable to delete graph %q: {{err}}", d.Id()), err)
	}

	d.SetId("")

	return nil
}

type _Graph struct {
	api.Graph
}

func _NewGraph() _Graph {
	g := _Graph{
		Graph: *api.NewGraph(),
	}

	return g
}

func _LoadGraph(ctxt *_ProviderContext, cid api.CIDType) (_Graph, error) {
	var g _Graph
	ng, err := ctxt.client.FetchGraph(cid)
	if err != nil {
		return _Graph{}, err
	}
	g.Graph = *ng

	return g, nil
}

// ParseConfig reads Terraform config data and stores the information into a
// Circonus Graph object.  ParseConfig and _GraphRead() must be kept in sync.
func (g *_Graph) ParseConfig(ar _AttrReader) error {
	g.Datapoints = make([]api.GraphDatapoint, 0, defaultGraphDatapoints)

	{
		leftAxisMap := ar.GetMap(_GraphLeftAttr)
		if v, ok := leftAxisMap[string(_GraphAxisLogarithmicAttr)]; ok {
			i64, _ := strconv.ParseInt(v.(string), 10, 64)
			i := int(i64)
			g.LogLeftY = &i
		}

		if v, ok := leftAxisMap[string(_GraphAxisMaxAttr)]; ok && v.(string) != "" {
			f, _ := strconv.ParseFloat(v.(string), 64)
			g.MaxLeftY = &f
		}

		if v, ok := leftAxisMap[string(_GraphAxisMinAttr)]; ok && v.(string) != "" {
			f, _ := strconv.ParseFloat(v.(string), 64)
			g.MinLeftY = &f
		}
	}

	{
		rightAxisMap := ar.GetMap(_GraphRightAttr)
		if v, ok := rightAxisMap[string(_GraphAxisLogarithmicAttr)]; ok {
			i64, _ := strconv.ParseInt(v.(string), 10, 64)
			i := int(i64)
			g.LogRightY = &i
		}

		if v, ok := rightAxisMap[string(_GraphAxisMaxAttr)]; ok && v.(string) != "" {
			f, _ := strconv.ParseFloat(v.(string), 64)
			g.MaxRightY = &f
		}

		if v, ok := rightAxisMap[string(_GraphAxisMinAttr)]; ok && v.(string) != "" {
			f, _ := strconv.ParseFloat(v.(string), 64)
			g.MinRightY = &f
		}
	}

	if s, ok := ar.GetStringOK(_GraphDescriptionAttr); ok {
		g.Description = s
	}

	if s, ok := ar.GetStringOK(_GraphLineStyleAttr); ok {
		g.LineStyle = s
	}

	if s, ok := ar.GetStringOK(_GraphNameAttr); ok {
		g.Title = s
	}

	if s, ok := ar.GetStringOK(_GraphNotesAttr); ok {
		g.Notes = &s
	}

	if streamList, ok := ar.GetListOK(_GraphStreamAttr); ok {
		for _, streamListRaw := range streamList {
			for _, streamListElem := range streamListRaw.([]interface{}) {
				streamAttrs := _NewInterfaceMap(streamListElem.(map[string]interface{}))
				streamReader := _NewMapReader(ar.Context(), streamAttrs)
				datapoint := api.GraphDatapoint{}

				if b, ok := streamReader.GetBoolOK(_GraphStreamActiveAttr); ok {
					datapoint.Hidden = !b
				}

				if f, ok := streamReader.GetFloat64OK(_GraphStreamAlphaAttr); ok && f != 0 {
					datapoint.Alpha = &f
				}

				if s, ok := streamReader.GetStringOK(_GraphStreamAxisAttr); ok {
					switch s {
					case "left", "":
						datapoint.Axis = "l"
					case "right":
						datapoint.Axis = "r"
					default:
						panic(fmt.Sprintf("PROVIDER BUG: Unsupported axis attribute %q", s))
					}
				}

				if s, ok := streamReader.GetStringOK(_GraphStreamCheckAttr); ok {
					re := regexp.MustCompile(config.CheckCIDRegex)
					matches := re.FindStringSubmatch(s)
					if len(matches) == 3 {
						checkID, _ := strconv.ParseUint(matches[2], 10, 64)
						datapoint.CheckID = uint(checkID)
					}
				}

				if s, ok := streamReader.GetStringOK(_GraphStreamColorAttr); ok {
					datapoint.Color = &s
				}

				if s := streamReader.GetStringPtr(_GraphStreamFormulaAttr); s != nil {
					datapoint.DataFormula = s
				}

				if s, ok := streamReader.GetStringOK(_GraphStreamFunctionAttr); ok && s != "" {
					datapoint.Derive = s
				} else {
					datapoint.Derive = false
				}

				if s := streamReader.GetStringPtr(_GraphStreamFormulaLegendAttr); s != nil {
					datapoint.LegendFormula = s
				}

				if s, ok := streamReader.GetStringOK(_GraphStreamNameAttr); ok && s != "" {
					datapoint.MetricName = s
				}

				if s, ok := streamReader.GetStringOK(_GraphStreamMetricTypeAttr); ok && s != "" {
					datapoint.MetricType = s
				}

				if s, ok := streamReader.GetStringOK(_GraphStreamHumanNameAttr); ok && s != "" {
					datapoint.Name = s
				}

				if s := streamReader.GetStringPtr(_GraphStreamStackAttr); s != nil && *s != "" {
					u64, _ := strconv.ParseUint(*s, 10, 64)
					u := uint(u64)
					datapoint.Stack = &u
				}

				g.Datapoints = append(g.Datapoints, datapoint)
			}
		}
	}

	if streamGroupList, ok := ar.GetListOK(_GraphStreamGroupAttr); ok {
		for _, streamGroupListRaw := range streamGroupList {
			for _, streamGroupListElem := range streamGroupListRaw.([]interface{}) {
				streamGroupAttrs := _NewInterfaceMap(streamGroupListElem.(map[string]interface{}))
				streamGroupReader := _NewMapReader(ar.Context(), streamGroupAttrs)
				metricCluster := api.GraphMetricCluster{}

				if b, ok := streamGroupReader.GetBoolOK(_GraphStreamGroupActiveAttr); ok {
					metricCluster.Hidden = !b
				}

				if s, ok := streamGroupReader.GetStringOK(_GraphStreamGroupAggregateAttr); ok {
					metricCluster.AggregateFunc = s
				}

				if s, ok := streamGroupReader.GetStringOK(_GraphStreamGroupAxisAttr); ok {
					switch s {
					case "left", "":
						metricCluster.Axis = "l"
					case "right":
						metricCluster.Axis = "r"
					default:
						panic(fmt.Sprintf("PROVIDER BUG: Unsupported axis attribute %q", s))
					}
				}

				if s := streamGroupReader.GetStringPtr(_GraphStreamFormulaAttr); s != nil {
					metricCluster.DataFormula = s
				}

				if s := streamGroupReader.GetStringPtr(_GraphStreamFormulaLegendAttr); s != nil {
					metricCluster.LegendFormula = s
				}

				if s, ok := streamGroupReader.GetStringOK(_GraphStreamGroupGroupAttr); ok && s != "" {
					metricCluster.MetricCluster = s
				}

				if s, ok := streamGroupReader.GetStringOK(_GraphStreamHumanNameAttr); ok && s != "" {
					metricCluster.Name = s
				}

				if s := streamGroupReader.GetStringPtr(_GraphStreamStackAttr); s != nil && *s != "" {
					u64, _ := strconv.ParseUint(*s, 10, 64)
					u := uint(u64)
					metricCluster.Stack = &u
				}

				g.MetricClusters = append(g.MetricClusters, metricCluster)
			}
		}
	}

	if s, ok := ar.GetStringOK(_GraphStyleAttr); ok {
		g.Style = s
	}

	g.Tags = tagsToAPI(ar.GetTags(_GraphTagsAttr))

	if err := g.Validate(); err != nil {
		return err
	}

	return nil
}

func (g *_Graph) Create(ctxt *_ProviderContext) error {
	ng, err := ctxt.client.CreateGraph(&g.Graph)
	if err != nil {
		return err
	}

	g.CID = ng.CID

	return nil
}

func (g *_Graph) Update(ctxt *_ProviderContext) error {
	_, err := ctxt.client.UpdateGraph(&g.Graph)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to update graph %s: {{err}}", g.CID), err)
	}

	return nil
}

func (g *_Graph) Validate() error {
	for i, datapoint := range g.Datapoints {
		if g.Style == _APIGraphStyleLine && datapoint.Alpha != nil && *datapoint.Alpha != 0 {
			return fmt.Errorf("%s can not be set on graphs with style %s", _GraphStreamAlphaAttr, _APIGraphStyleLine)
		}

		if datapoint.CheckID != 0 && datapoint.MetricName == "" {
			return fmt.Errorf("Error with stream[%d] name=%q: %s is set, missing attribute %s must also be set", i, datapoint.Name, _GraphStreamCheckAttr, _GraphStreamNameAttr)
		}

		if datapoint.CheckID == 0 && datapoint.MetricName != "" {
			return fmt.Errorf("Error with stream[%d] name=%q: %s is set, missing attribute %s must also be set", i, datapoint.Name, _GraphStreamNameAttr, _GraphStreamCheckAttr)
		}

		if datapoint.CAQL != nil && (datapoint.CheckID != 0 || datapoint.MetricName != "") {
			return fmt.Errorf("Error with stream[%d] name=%q: %q attribute is mutually exclusive with attributes %s or %s", i, datapoint.Name, _GraphStreamCAQLAttr, _GraphStreamNameAttr, _GraphStreamCheckAttr)
		}
	}

	return nil
}
