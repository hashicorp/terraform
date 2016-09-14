package datadog

type TextSize struct {
	Size int
	Auto bool
}

type TileDef struct {
	Events   []TileDefEvent      `json:"events,omitempty"`
	Markers  []TimeseriesMarker  `json:"markers,omitempty"`
	Requests []TimeseriesRequest `json:"requests,omitempty"`
	Viz      string              `json:"viz,omitempty"`
}

type TimeseriesRequest struct {
	Query              string                 `json:"q,omitempty"`
	Type               string                 `json:"type,omitempty"`
	ConditionalFormats []ConditionalFormat    `json:"conditional_formats,omitempty"`
	Style              TimeseriesRequestStyle `json:"style,omitempty"`
}

type TimeseriesRequestStyle struct {
	Palette string `json:"palette,omitempty"`
}

type TimeseriesMarker struct {
	Label string `json:"label,omitempty"`
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type TileDefEvent struct {
	Query string `json:"q"`
}

type AlertValueWidget struct {
	TitleSize    int    `json:"title_size,omitempty"`
	Title        bool   `json:"title,omitempty"`
	TitleAlign   string `json:"title_align,omitempty"`
	TextAlign    string `json:"text_align,omitempty"`
	TitleText    string `json:"title_text,omitempty"`
	Precision    int    `json:"precision,omitempty"`
	AlertId      int    `json:"alert_id,omitempty"`
	Timeframe    string `json:"timeframe,omitempty"`
	AddTimeframe bool   `json:"add_timeframe,omitempty"`
	Y            int    `json:"y,omitempty"`
	X            int    `json:"x,omitempty"`
	TextSize     string `json:"text_size,omitempty"`
	Height       int    `json:"height,omitempty"`
	Width        int    `json:"width,omitempty"`
	Type         string `json:"type,omitempty"`
	Unit         string `json:"unit,omitempty"`
}

type ChangeWidget struct {
	TitleSize  int     `json:"title_size,omitempty"`
	Title      bool    `json:"title,omitempty"`
	TitleAlign string  `json:"title_align,omitempty"`
	TitleText  string  `json:"title_text,omitempty"`
	Height     int     `json:"height,omitempty"`
	Width      int     `json:"width,omitempty"`
	X          int     `json:"y,omitempty"`
	Y          int     `json:"x,omitempty"`
	Aggregator string  `json:"aggregator,omitempty"`
	TileDef    TileDef `json:"tile_def,omitempty"`
}

type GraphWidget struct {
	TitleSize  int     `json:"title_size,omitempty"`
	Title      bool    `json:"title,omitempty"`
	TitleAlign string  `json:"title_align,omitempty"`
	TitleText  string  `json:"title_text,omitempty"`
	Height     int     `json:"height,omitempty"`
	Width      int     `json:"width,omitempty"`
	X          int     `json:"y,omitempty"`
	Y          int     `json:"x,omitempty"`
	Type       string  `json:"type,omitempty"`
	Timeframe  string  `json:"timeframe,omitempty"`
	LegendSize int     `json:"legend_size,omitempty"`
	Legend     bool    `json:"legend,omitempty"`
	TileDef    TileDef `json:"tile_def,omitempty"`
}

type EventTimelineWidget struct {
	TitleSize  int    `json:"title_size,omitempty"`
	Title      bool   `json:"title,omitempty"`
	TitleAlign string `json:"title_align,omitempty"`
	TitleText  string `json:"title_text,omitempty"`
	Height     int    `json:"height,omitempty"`
	Width      int    `json:"width,omitempty"`
	X          int    `json:"y,omitempty"`
	Y          int    `json:"x,omitempty"`
	Type       string `json:"type,omitempty"`
	Timeframe  string `json:"timeframe,omitempty"`
	Query      string `json:"query,omitempty"`
}

type AlertGraphWidget struct {
	TitleSize    int    `json:"title_size,omitempty"`
	VizType      string `json:"timeseries,omitempty"`
	Title        bool   `json:"title,omitempty"`
	TitleAlign   string `json:"title_align,omitempty"`
	TitleText    string `json:"title_text,omitempty"`
	Height       int    `json:"height,omitempty"`
	Width        int    `json:"width,omitempty"`
	X            int    `json:"y,omitempty"`
	Y            int    `json:"x,omitempty"`
	AlertId      int    `json:"alert_id,omitempty"`
	Timeframe    string `json:"timeframe,omitempty"`
	Type         string `json:"type,omitempty"`
	AddTimeframe bool   `json:"add_timeframe,omitempty"`
}

type HostMapWidget struct {
	TitleSize  int     `json:"title_size,omitempty"`
	Title      bool    `json:"title,omitempty"`
	TitleAlign string  `json:"title_align,omitempty"`
	TitleText  string  `json:"title_text,omitempty"`
	Height     int     `json:"height,omitempty"`
	Width      int     `json:"width,omitempty"`
	X          int     `json:"y,omitempty"`
	Y          int     `json:"x,omitempty"`
	Query      string  `json:"query,omitempty"`
	Timeframe  string  `json:"timeframe,omitempty"`
	LegendSize int     `json:"legend_size,omitempty"`
	Type       string  `json:"type,omitempty"`
	Legend     bool    `json:"legend,omitempty"`
	TileDef    TileDef `json:"tile_def,omitempty"`
}

type CheckStatusWidget struct {
	TitleSize  int    `json:"title_size,omitempty"`
	Title      bool   `json:"title,omitempty"`
	TitleAlign string `json:"title_align,omitempty"`
	TextAlign  string `json:"text_align,omitempty"`
	TitleText  string `json:"title_text,omitempty"`
	Height     int    `json:"height,omitempty"`
	Width      int    `json:"width,omitempty"`
	X          int    `json:"y,omitempty"`
	Y          int    `json:"x,omitempty"`
	Tags       string `json:"tags,omitempty"`
	Timeframe  string `json:"timeframe,omitempty"`
	TextSize   string `json:"text_size,omitempty"`
	Type       string `json:"type,omitempty"`
	Check      string `json:"check,omitempty"`
	Group      string `json:"group,omitempty"`
	Grouping   string `json:"grouping,omitempty"`
}

type IFrameWidget struct {
	TitleSize  int    `json:"title_size,omitempty"`
	Title      bool   `json:"title,omitempty"`
	Url        string `json:"url,omitempty"`
	TitleAlign string `json:"title_align,omitempty"`
	TitleText  string `json:"title_text,omitempty"`
	Height     int    `json:"height,omitempty"`
	Width      int    `json:"width,omitempty"`
	X          int    `json:"y,omitempty"`
	Y          int    `json:"x,omitempty"`
	Type       string `json:"type,omitempty"`
}

type NoteWidget struct {
	TitleSize    int    `json:"title_size,omitempty"`
	Title        bool   `json:"title,omitempty"`
	RefreshEvery int    `json:"refresh_every,omitempty"`
	TickPos      string `json:"tick_pos,omitempty"`
	TitleAlign   string `json:"title_align,omitempty"`
	TickEdge     string `json:"tick_edge,omitempty"`
	TextAlign    string `json:"text_align,omitempty"`
	TitleText    string `json:"title_text,omitempty"`
	Height       int    `json:"height,omitempty"`
	Color        string `json:"bgcolor,omitempty"`
	Html         string `json:"html,omitempty"`
	Y            int    `json:"y,omitempty"`
	X            int    `json:"x,omitempty"`
	FontSize     int    `json:"font_size,omitempty"`
	Tick         bool   `json:"tick,omitempty"`
	Note         string `json:"type,omitempty"`
	Width        int    `json:"width,omitempty"`
	AutoRefresh  bool   `json:"auto_refresh,omitempty"`
}

type TimeseriesWidget struct {
	Height     int      `json:"height,omitempty"`
	Legend     bool     `json:"legend,omitempty"`
	TileDef    TileDef  `json:"tile_def,omitempty"`
	Timeframe  string   `json:"timeframe,omitempty"`
	Title      bool     `json:"title,omitempty"`
	TitleAlign string   `json:"title_align,omitempty"`
	TitleSize  TextSize `json:"title_size,omitempty"`
	TitleText  string   `json:"title_text,omitempty"`
	Type       string   `json:"type,omitempty"`
	Width      int      `json:"width,omitempty"`
	X          int      `json:"x,omitempty"`
	Y          int      `json:"y,omitempty"`
}

type QueryValueWidget struct {
	Timeframe           string              `json:"timeframe,omitempty"`
	TimeframeAggregator string              `json:"aggr,omitempty"`
	Aggregator          string              `json:"aggregator,omitempty"`
	CalcFunc            string              `json:"calc_func,omitempty"`
	ConditionalFormats  []ConditionalFormat `json:"conditional_formats,omitempty"`
	Height              int                 `json:"height,omitempty"`
	IsValidQuery        bool                `json:"is_valid_query,omitempty,omitempty"`
	Metric              string              `json:"metric,omitempty"`
	MetricType          string              `json:"metric_type,omitempty"`
	Precision           int                 `json:"precision,omitempty"`
	Query               string              `json:"query,omitempty"`
	ResultCalcFunc      string              `json:"res_calc_func,omitempty"`
	Tags                []string            `json:"tags,omitempty"`
	TextAlign           string              `json:"text_align,omitempty"`
	TextSize            TextSize            `json:"text_size,omitempty"`
	Title               bool                `json:"title,omitempty"`
	TitleAlign          string              `json:"title_align,omitempty"`
	TitleSize           TextSize            `json:"title_size,omitempty"`
	TitleText           string              `json:"title_text,omitempty"`
	Type                string              `json:"type,omitempty"`
	Unit                string              `json:"auto,omitempty"`
	Width               int                 `json:"width,omitempty"`
	X                   int                 `json:"x,omitempty"`
	Y                   int                 `json:"y,omitempty"`
}
type ConditionalFormat struct {
	Color      string `json:"color,omitempty"`
	Comparator string `json:"comparator,omitempty"`
	Inverted   bool   `json:"invert,omitempty"`
	Value      int    `json:"value,omitempty"`
}

type ToplistWidget struct {
	Height     int      `json:"height,omitempty"`
	Legend     bool     `json:"legend,omitempty"`
	LegendSize int      `json:"legend_size,omitempty"`
	TileDef    TileDef  `json:"tile_def,omitempty"`
	Timeframe  string   `json:"timeframe,omitempty"`
	Title      bool     `json:"title,omitempty"`
	TitleAlign string   `json:"title_align,omitempty"`
	TitleSize  TextSize `json:"title_size,omitempty"`
	TitleText  string   `json:"title_text,omitempty"`
	Type       string   `json:"type,omitempty"`
	Width      int      `json:"width,omitempty"`
	X          int      `json:"x,omitempty"`
	Y          int      `json:"y,omitempty"`
}

type EventStreamWidget struct {
	EventSize  string   `json:"event_size,omitempty"`
	Height     int      `json:"height,omitempty"`
	Query      string   `json:"query,omitempty"`
	Timeframe  string   `json:"timeframe,omitempty"`
	Title      bool     `json:"title,omitempty"`
	TitleAlign string   `json:"title_align,omitempty"`
	TitleSize  TextSize `json:"title_size,omitempty"`
	TitleText  string   `json:"title_text,omitempty"`
	Type       string   `json:"type,omitempty"`
	Width      int      `json:"width,omitempty"`
	X          int      `json:"x,omitempty"`
	Y          int      `json:"y,omitempty"`
}

type FreeTextWidget struct {
	Color     string `json:"color,omitempty"`
	FontSize  string `json:"font_size,omitempty"`
	Height    int    `json:"height,omitempty"`
	Text      string `json:"text,omitempty"`
	TextAlign string `json:"text_align,omitempty"`
	Type      string `json:"type,omitempty"`
	Width     int    `json:"width,omitempty"`
	X         int    `json:"x,omitempty"`
	Y         int    `json:"y,omitempty"`
}

type ImageWidget struct {
	Height     int      `json:"height,omitempty"`
	Sizing     string   `json:"sizing,omitempty"`
	Title      bool     `json:"title,omitempty"`
	TitleAlign string   `json:"title_align,omitempty"`
	TitleSize  TextSize `json:"title_size,omitempty"`
	TitleText  string   `json:"title_text,omitempty"`
	Type       string   `json:"type,omitempty"`
	Url        string   `json:"url,omitempty"`
	Width      int      `json:"width,omitempty"`
	X          int      `json:"x,omitempty"`
	Y          int      `json:"y,omitempty"`
}
