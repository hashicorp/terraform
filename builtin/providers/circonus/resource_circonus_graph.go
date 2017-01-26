package circonus

import (
	"fmt"
	"regexp"
	"strconv"

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
	_GraphStreamActiveAttr    _SchemaAttr = "active"
	_GraphAlphaAttr           _SchemaAttr = "alpha"
	_GraphStreamAxisAttr      _SchemaAttr = "axis"
	_GraphCAQLAttr            _SchemaAttr = "caql"
	_GraphCheckAttr           _SchemaAttr = "check"
	_GraphColorAttr           _SchemaAttr = "color"
	_GraphFormulaAttr         _SchemaAttr = "formula"
	_GraphFormulaLegendAttr   _SchemaAttr = "legend_formula"
	_GraphFunctionAttr        _SchemaAttr = "function"
	_GraphMetricTypeAttr      _SchemaAttr = "metric_type"
	_GraphStreamHumanNameAttr _SchemaAttr = "name"
	_GraphStreamNameAttr      _SchemaAttr = "stream_name"
	_GraphStreamStackAttr     _SchemaAttr = "stack"

	// circonus_graph.stream_group.* resource attribute names
	_GraphStreamGroupActiveAttr    _SchemaAttr = "active"
	_GraphAggregateAttr            _SchemaAttr = "aggregate"
	_GraphStreamGroupAxisAttr      _SchemaAttr = "axis"
	_GraphGroupAttr                _SchemaAttr = "group"
	_GraphStreamGroupHumanNameAttr _SchemaAttr = "name"

	// circonus_graph.{left,right}.* resource attribute names
	_GraphLogarithmicAttr _SchemaAttr = "logarithmic"
	_GraphMaxAttr         _SchemaAttr = "max"
	_GraphMinAttr         _SchemaAttr = "min"
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
	_GraphStreamActiveAttr:    "",
	_GraphAlphaAttr:           "",
	_GraphStreamAxisAttr:      "",
	_GraphCAQLAttr:            "",
	_GraphCheckAttr:           "",
	_GraphColorAttr:           "",
	_GraphFormulaAttr:         "",
	_GraphFormulaLegendAttr:   "",
	_GraphFunctionAttr:        "",
	_GraphMetricTypeAttr:      "",
	_GraphStreamHumanNameAttr: "",
	_GraphStreamNameAttr:      "",
	_GraphStreamStackAttr:     "",
}

var _GraphStreamGroupDescriptions = _AttrDescrs{
	// circonus_graph.stream_group.* resource attribute names
	_GraphStreamGroupActiveAttr:    "",
	_GraphAggregateAttr:            "",
	_GraphStreamGroupAxisAttr:      "",
	_GraphGroupAttr:                "",
	_GraphStreamGroupHumanNameAttr: "",
}

var _GraphStreamAxisOptionDescriptions = _AttrDescrs{
	// circonus_graph.if.value.over.* resource attribute names
	_GraphLogarithmicAttr: "",
	_GraphMaxAttr:         "",
	_GraphMinAttr:         "",
}

func _NewGraphResource() *schema.Resource {
	// makeConflictsWith := func(in ..._SchemaAttr) []string {
	// 	out := make([]string, 0, len(in))
	// 	for _, attr := range in {
	// 		out = append(out, string(_GraphStreamAttr)+"."+string(attr))
	// 	}
	// 	return out
	// }

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
				Type:     schema.TypeString,
				Optional: true,
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
						_GraphAlphaAttr: &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
							ValidateFunc: _ValidateFuncs(
								_ValidateFloatMin(_GraphAlphaAttr, 0.0),
								_ValidateFloatMax(_GraphAlphaAttr, 1.0),
							),
						},
						_GraphStreamAxisAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "left",
							ValidateFunc: _ValidateStringIn(_GraphStreamAxisAttr, _ValidAxisAttrs),
						},
						_GraphCAQLAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphCAQLAttr, `.+`),
							// ConflictsWith: makeConflictsWith(_GraphCheckAttr, _GraphStreamAttr),
						},
						_GraphCheckAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphCheckAttr, config.CheckCIDRegex),
							// ConflictsWith: makeConflictsWith(_GraphCAQLAttr),
						},
						_GraphColorAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphColorAttr, `^#[0-9a-fA-F]{6}$`),
						},
						_GraphFormulaAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphFormulaAttr, `^.+$`),
						},
						_GraphFormulaLegendAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphFormulaLegendAttr, `^.+$`),
						},
						_GraphFunctionAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateStringIn(_GraphFunctionAttr, _ValidGraphFunctionValues),
						},
						_GraphMetricTypeAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: _ValidateStringIn(_GraphMetricTypeAttr, _ValidMetricTypes),
						},
						_GraphStreamHumanNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamHumanNameAttr, `.+`),
						},
						_GraphStreamNameAttr: &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: _ValidateRegexp(_GraphStreamNameAttr, `^[\S]+$`),
						},
						_GraphStreamStackAttr: &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ValidateFunc: _ValidateFuncs(
								_ValidateIntMin(_GraphStreamStackAttr, 0),
							),
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
						_GraphAggregateAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "none",
							ValidateFunc: _ValidateStringIn(_GraphAggregateAttr, _ValidAggregateFuncs),
						},
						_GraphStreamGroupAxisAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "left",
							ValidateFunc: _ValidateStringIn(_GraphStreamGroupAttr, _ValidAxisAttrs),
						},
						_GraphGroupAttr: &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: _ValidateRegexp(_GraphGroupAttr, config.MetricClusterCIDRegex),
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
	for i, datapoint := range g.Datapoints {
		dataPointAttrs := make(map[string]interface{}, 13) // 13 == len(members in api.GraphDatapoint)

		dataPointAttrs[string(_GraphStreamActiveAttr)] = !datapoint.Hidden

		if datapoint.Alpha != "" {
			f, err := strconv.ParseFloat(datapoint.Alpha, 32)
			if err != nil {
				return errwrap.Wrapf(fmt.Sprintf("Unable to parse datapoint %d's alpha %q: {{err}}", i, datapoint.Alpha), err)
			}
			dataPointAttrs[string(_GraphAlphaAttr)] = f
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
			dataPointAttrs[string(_GraphCAQLAttr)] = *datapoint.CAQL
		}

		if datapoint.CheckID != 0 {
			dataPointAttrs[string(_GraphCheckAttr)] = fmt.Sprintf("%s/%d", config.CheckPrefix, datapoint.CheckID)
		}

		if datapoint.Color != "" {
			dataPointAttrs[string(_GraphColorAttr)] = datapoint.Color
		}

		if datapoint.DataFormula != nil {
			dataPointAttrs[string(_GraphFormulaAttr)] = *datapoint.DataFormula
		}

		switch datapoint.Derive.(type) {
		case bool:
		case string:
			dataPointAttrs[string(_GraphFunctionAttr)] = datapoint.Derive.(string)
		default:
			panic(fmt.Sprintf("PROVIDER BUG: Unsupported type for derive: %T", datapoint.Derive))
		}

		if datapoint.LegendFormula != nil {
			dataPointAttrs[string(_GraphFormulaLegendAttr)] = *datapoint.LegendFormula
		}

		if datapoint.MetricName != "" {
			dataPointAttrs[string(_GraphStreamNameAttr)] = datapoint.MetricName
		}

		if datapoint.MetricType != "" {
			dataPointAttrs[string(_GraphMetricTypeAttr)] = datapoint.MetricType
		}

		if datapoint.Name != "" {
			dataPointAttrs[string(_GraphStreamHumanNameAttr)] = datapoint.Name
		}

		if datapoint.Stack != nil {
			dataPointAttrs[string(_GraphStreamStackAttr)] = *datapoint.Stack
		}

		streams = append(streams, dataPointAttrs)
	}

	streamGroups := make([]interface{}, 0, len(g.MetricClusters))
	for _, metricCluster := range g.MetricClusters {
		streamGroupAttrs := make(map[string]interface{}, 8) // 8 == len(num struct attrs in api.GraphMetricCluster)

		streamGroupAttrs[string(_GraphStreamGroupActiveAttr)] = !metricCluster.Hidden

		if metricCluster.AggregateFunc != "" {
			streamGroupAttrs[string(_GraphAggregateAttr)] = metricCluster.AggregateFunc
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
			streamGroupAttrs[string(_GraphFormulaAttr)] = *metricCluster.DataFormula
		}

		if metricCluster.LegendFormula != nil {
			streamGroupAttrs[string(_GraphFormulaLegendAttr)] = *metricCluster.LegendFormula
		}

		if metricCluster.MetricCluster != "" {
			streamGroupAttrs[string(_GraphGroupAttr)] = metricCluster.MetricCluster
		}

		if metricCluster.Name != "" {
			streamGroupAttrs[string(_GraphStreamHumanNameAttr)] = metricCluster.Name
		}

		if metricCluster.Stack != nil {
			streamGroupAttrs[string(_GraphStreamStackAttr)] = *metricCluster.Stack
		}

		streamGroups = append(streamGroups, streamGroupAttrs)
	}

	leftAxisMap := make(map[string]interface{}, 3)
	leftAxisMap[string(_GraphLogarithmicAttr)] = fmt.Sprintf("%d", g.LogLeftY)

	if g.MaxLeftY != nil && *g.MaxLeftY != "" {
		leftAxisMap[string(_GraphMaxAttr)] = *g.MaxLeftY
	}

	if g.MinLeftY != nil && *g.MinLeftY != "" {
		leftAxisMap[string(_GraphMinAttr)] = *g.MinLeftY
	}

	rightAxisMap := make(map[string]interface{}, 3)
	rightAxisMap[string(_GraphLogarithmicAttr)] = fmt.Sprintf("%d", g.LogRightY)

	if g.MaxRightY != nil && *g.MaxRightY != "" {
		rightAxisMap[string(_GraphMaxAttr)] = *g.MaxRightY
	}

	if g.MinRightY != nil && *g.MinRightY != "" {
		rightAxisMap[string(_GraphMinAttr)] = *g.MinRightY
	}

	_StateSet(d, _GraphDescriptionAttr, g.Description)
	_StateSet(d, _GraphLeftAttr, leftAxisMap)
	_StateSet(d, _GraphLineStyleAttr, g.LineStyle)
	_StateSet(d, _GraphNameAttr, g.Title)
	_StateSet(d, _GraphNotesAttr, _Indirect(g.Notes))
	_StateSet(d, _GraphRightAttr, rightAxisMap)
	_StateSet(d, _GraphStreamAttr, streams)
	_StateSet(d, _GraphStreamGroupAttr, streamGroups)
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
		leftMap := ar.GetMap(_GraphLeftAttr)
		if v, ok := leftMap[string(_GraphLogarithmicAttr)]; ok && v.(string) != "0" {
			switch v.(string) {
			case "0":
				g.LogLeftY = 0
			case "1":
				g.LogLeftY = 1
			default:
				panic(fmt.Sprintf("PROVIDER BUG: unsupported log attribute: %q", v.(string)))
			}
		}

		if v, ok := leftMap[string(_GraphMaxAttr)]; ok && v.(string) != "" {
			s := v.(string)
			g.MaxLeftY = &s
		}

		if v, ok := leftMap[string(_GraphMinAttr)]; ok && v.(string) != "" {
			s := v.(string)
			g.MinLeftY = &s
		}
	}

	{
		rightMap := ar.GetMap(_GraphRightAttr)
		if v, ok := rightMap[string(_GraphLogarithmicAttr)]; ok && v.(string) != "0" {
			switch v.(string) {
			case "0":
				g.LogRightY = 0
			case "1":
				g.LogRightY = 1
			default:
				panic(fmt.Sprintf("PROVIDER BUG: unsupported log attribute: %q", v.(string)))
			}
		}

		if v, ok := rightMap[string(_GraphMaxAttr)]; ok && v.(string) != "" {
			s := v.(string)
			g.MaxRightY = &s
		}

		if v, ok := rightMap[string(_GraphMinAttr)]; ok && v.(string) != "" {
			s := v.(string)
			g.MinRightY = &s
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

				if f, ok := streamReader.GetFloat64OK(_GraphAlphaAttr); ok {
					datapoint.Alpha = fmt.Sprintf("%f", f)
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

				if s, ok := streamReader.GetStringOK(_GraphCheckAttr); ok {
					re := regexp.MustCompile(config.CheckCIDRegex)
					matches := re.FindStringSubmatch(s)
					if len(matches) == 3 {
						checkID, _ := strconv.ParseUint(matches[2], 10, 64)
						datapoint.CheckID = uint(checkID)
					}
				}

				if s, ok := streamReader.GetStringOK(_GraphColorAttr); ok {
					datapoint.Color = s
				}

				if s := streamReader.GetStringPtr(_GraphFormulaAttr); s != nil {
					datapoint.DataFormula = s
				}

				if s, ok := streamReader.GetStringOK(_GraphFunctionAttr); ok && s != "" {
					datapoint.Derive = s
				} else {
					datapoint.Derive = false
				}

				if s := streamReader.GetStringPtr(_GraphFormulaLegendAttr); s != nil {
					datapoint.LegendFormula = s
				}

				if s, ok := streamReader.GetStringOK(_GraphStreamNameAttr); ok && s != "" {
					datapoint.MetricName = s
				}

				if s, ok := streamReader.GetStringOK(_GraphMetricTypeAttr); ok && s != "" {
					datapoint.MetricType = s
				}

				if s, ok := streamReader.GetStringOK(_GraphStreamHumanNameAttr); ok && s != "" {
					datapoint.Name = s
				}

				if i, ok := streamReader.GetIntOK(_GraphStreamStackAttr); ok {
					u := uint(i)
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

				if s, ok := streamGroupReader.GetStringOK(_GraphAggregateAttr); ok {
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

				if s := streamGroupReader.GetStringPtr(_GraphFormulaAttr); s != nil {
					metricCluster.DataFormula = s
				}

				if s := streamGroupReader.GetStringPtr(_GraphFormulaLegendAttr); s != nil {
					metricCluster.LegendFormula = s
				}

				if s, ok := streamGroupReader.GetStringOK(_GraphGroupAttr); ok && s != "" {
					metricCluster.MetricCluster = s
				}

				if s, ok := streamGroupReader.GetStringOK(_GraphStreamHumanNameAttr); ok && s != "" {
					metricCluster.Name = s
				}

				if i, ok := streamGroupReader.GetIntOK(_GraphStreamStackAttr); ok {
					u := uint(i)
					metricCluster.Stack = &u
				}

				g.MetricClusters = append(g.MetricClusters, metricCluster)
			}
		}
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
	return nil
}
