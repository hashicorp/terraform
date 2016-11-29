package integration

import (
	"testing"

	"github.com/zorkian/go-datadog-api"
)

func TestAlertValueWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.AlertValueWidget

	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.TitleText = "foo"
	expected.TitleAlign = "center"
	expected.TitleSize = 1
	expected.Title = true
	expected.TextSize = "auto"
	expected.Precision = 2
	expected.AlertId = 1
	expected.Type = "alert_value"
	expected.Unit = "auto"
	expected.AddTimeframe = false

	w := datadog.Widget{AlertValueWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].AlertValueWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "text_size", actualWidget.TextSize, expected.TextSize)
	assertEquals(t, "precision", actualWidget.Precision, expected.Precision)
	assertEquals(t, "alert_id", actualWidget.AlertId, expected.AlertId)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
	assertEquals(t, "unit", actualWidget.Unit, expected.Unit)
	assertEquals(t, "add_timeframe", actualWidget.AddTimeframe, expected.AddTimeframe)
}

func TestChangeWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.ChangeWidget

	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.TitleText = "foo"
	expected.TitleAlign = "center"
	expected.TitleSize = 1
	expected.Title = true
	expected.Aggregator = "min"
	expected.TileDef = datadog.TileDef{}

	w := datadog.Widget{ChangeWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].ChangeWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "aggregator", actualWidget.Aggregator, expected.Aggregator)
	assertTileDefEquals(t, actualWidget.TileDef, expected.TileDef)
}

func TestGraphWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.GraphWidget

	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.TitleText = "foo"
	expected.TitleAlign = "center"
	expected.TitleSize = 1
	expected.Title = true
	expected.Timeframe = "1d"
	expected.Type = "alert_graph"
	expected.Legend = true
	expected.LegendSize = 5
	expected.TileDef = datadog.TileDef{}

	w := datadog.Widget{GraphWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].GraphWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
	assertEquals(t, "timeframe", actualWidget.Timeframe, expected.Timeframe)
	assertEquals(t, "legend", actualWidget.Legend, expected.Legend)
	assertEquals(t, "legend_size", actualWidget.LegendSize, expected.LegendSize)
	assertTileDefEquals(t, actualWidget.TileDef, expected.TileDef)
}

func TestEventTimelineWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.EventTimelineWidget

	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.TitleText = "foo"
	expected.TitleAlign = "center"
	expected.TitleSize = 1
	expected.Title = true
	expected.Query = "avg:system.load.1{foo} by {bar}"
	expected.Timeframe = "1d"
	expected.Type = "alert_graph"

	w := datadog.Widget{EventTimelineWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].EventTimelineWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
	assertEquals(t, "query", actualWidget.Query, expected.Query)
	assertEquals(t, "timeframe", actualWidget.Timeframe, expected.Timeframe)
}

func TestAlertGraphWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.AlertGraphWidget

	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.TitleText = "foo"
	expected.TitleAlign = "center"
	expected.TitleSize = 1
	expected.Title = true
	expected.VizType = ""
	expected.Timeframe = "1d"
	expected.AddTimeframe = false
	expected.AlertId = 1
	expected.Type = "alert_graph"

	w := datadog.Widget{AlertGraphWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].AlertGraphWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
	assertEquals(t, "viz_type", actualWidget.VizType, expected.VizType)
	assertEquals(t, "timeframe", actualWidget.Timeframe, expected.Timeframe)
	assertEquals(t, "add_timeframe", actualWidget.AddTimeframe, expected.AddTimeframe)
	assertEquals(t, "alert_id", actualWidget.AlertId, expected.AlertId)
}

func TestHostMapWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.HostMapWidget

	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.TitleText = "foo"
	expected.TitleAlign = "center"
	expected.TitleSize = 1
	expected.Title = true
	expected.Type = "check_status"
	expected.Query = "avg:system.load.1{foo} by {bar}"
	expected.Timeframe = "1d"
	expected.Legend = true
	expected.LegendSize = 5
	expected.TileDef = datadog.TileDef{}

	w := datadog.Widget{HostMapWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].HostMapWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
	assertEquals(t, "query", actualWidget.Query, expected.Query)
	assertEquals(t, "timeframe", actualWidget.Timeframe, expected.Timeframe)
	assertEquals(t, "query", actualWidget.Query, expected.Query)
	assertEquals(t, "legend", actualWidget.Legend, expected.Legend)
	assertEquals(t, "legend_size", actualWidget.LegendSize, expected.LegendSize)
	assertTileDefEquals(t, actualWidget.TileDef, expected.TileDef)
}

func TestCheckStatusWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.CheckStatusWidget

	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.TitleText = "foo"
	expected.TitleAlign = "center"
	expected.TitleSize = 1
	expected.Title = true
	expected.Type = "check_status"
	expected.Tags = "foo"
	expected.Timeframe = "1d"
	expected.Timeframe = "1d"
	expected.Check = "datadog.agent.up"
	expected.Group = "foo"
	expected.Grouping = "check"

	w := datadog.Widget{CheckStatusWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].CheckStatusWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
	assertEquals(t, "tags", actualWidget.Tags, expected.Tags)
	assertEquals(t, "timeframe", actualWidget.Timeframe, expected.Timeframe)
	assertEquals(t, "check", actualWidget.Check, expected.Check)
	assertEquals(t, "group", actualWidget.Group, expected.Group)
	assertEquals(t, "grouping", actualWidget.Grouping, expected.Grouping)
}

func TestIFrameWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.IFrameWidget

	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.TitleText = "foo"
	expected.TitleAlign = "center"
	expected.TitleSize = 1
	expected.Title = true
	expected.Url = "http://www.example.com"
	expected.Type = "iframe"

	w := datadog.Widget{IFrameWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].IFrameWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "url", actualWidget.Url, expected.Url)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
}

func TestNoteWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.NoteWidget

	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.TitleText = "foo"
	expected.TitleAlign = "center"
	expected.TitleSize = 1
	expected.Title = true
	expected.Color = "green"
	expected.FontSize = 5
	expected.RefreshEvery = 60
	expected.TickPos = "foo"
	expected.TickEdge = "bar"
	expected.Html = "<strong>baz</strong>"
	expected.Tick = false
	expected.Note = "quz"
	expected.AutoRefresh = false

	w := datadog.Widget{NoteWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].NoteWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "color", actualWidget.Color, expected.Color)
	assertEquals(t, "front_size", actualWidget.FontSize, expected.FontSize)
	assertEquals(t, "refresh_every", actualWidget.RefreshEvery, expected.RefreshEvery)
	assertEquals(t, "tick_pos", actualWidget.TickPos, expected.TickPos)
	assertEquals(t, "tick_edge", actualWidget.TickEdge, expected.TickEdge)
	assertEquals(t, "tick", actualWidget.Tick, expected.Tick)
	assertEquals(t, "html", actualWidget.Html, expected.Html)
	assertEquals(t, "note", actualWidget.Note, expected.Note)
	assertEquals(t, "auto_refresh", actualWidget.AutoRefresh, expected.AutoRefresh)
}

func TestToplistWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.ToplistWidget
	expected.X = 1
	expected.Y = 1
	expected.Width = 5
	expected.Height = 5
	expected.Type = "toplist"
	expected.TitleText = "foo"
	expected.TitleSize.Auto = false
	expected.TitleSize.Size = 5
	expected.TitleAlign = "center"
	expected.Title = false
	expected.Timeframe = "5m"
	expected.Legend = false
	expected.LegendSize = 5

	w := datadog.Widget{ToplistWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].ToplistWidget

	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "legend", actualWidget.Legend, expected.Legend)
	assertEquals(t, "legend_size", actualWidget.LegendSize, expected.LegendSize)
}

func TestEventSteamWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.EventStreamWidget
	expected.EventSize = "1"
	expected.Width = 1
	expected.Height = 1
	expected.X = 1
	expected.Y = 1
	expected.Query = "foo"
	expected.Timeframe = "5w"
	expected.Title = false
	expected.TitleAlign = "center"
	expected.TitleSize.Auto = false
	expected.TitleSize.Size = 5
	expected.TitleText = "bar"
	expected.Type = "baz"

	w := datadog.Widget{EventStreamWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].EventStreamWidget

	assertEquals(t, "event_size", actualWidget.EventSize, expected.EventSize)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "query", actualWidget.Query, expected.Query)
	assertEquals(t, "timeframe", actualWidget.Timeframe, expected.Timeframe)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
}

func TestImageWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.ImageWidget

	expected.Width = 1
	expected.Height = 1
	expected.X = 1
	expected.Y = 1
	expected.Title = false
	expected.TitleAlign = "center"
	expected.TitleSize.Auto = false
	expected.TitleSize.Size = 5
	expected.TitleText = "bar"
	expected.Type = "baz"
	expected.Url = "qux"
	expected.Sizing = "quuz"

	w := datadog.Widget{ImageWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].ImageWidget

	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "title_align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title_size", actualWidget.TitleSize, expected.TitleSize)
	assertEquals(t, "title_text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
	assertEquals(t, "url", actualWidget.Url, expected.Url)
	assertEquals(t, "sizing", actualWidget.Sizing, expected.Sizing)
}

func TestFreeTextWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.FreeTextWidget

	expected.X = 1
	expected.Y = 1
	expected.Height = 10
	expected.Width = 10
	expected.Text = "Test"
	expected.FontSize = "16"
	expected.TextAlign = "center"

	w := datadog.Widget{FreeTextWidget: expected}

	board.Widgets = append(board.Widgets, w)

	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].FreeTextWidget

	assertEquals(t, "font-size", actualWidget.FontSize, expected.FontSize)
	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "text", actualWidget.Text, expected.Text)
	assertEquals(t, "text-align", actualWidget.TextAlign, expected.TextAlign)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
}

func TestTimeseriesWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.TimeseriesWidget
	expected.X = 1
	expected.Y = 1
	expected.Width = 20
	expected.Height = 30
	expected.Title = true
	expected.TitleAlign = "centre"
	expected.TitleSize = datadog.TextSize{Size: 16}
	expected.TitleText = "Test"
	expected.Timeframe = "1m"

	w := datadog.Widget{TimeseriesWidget: expected}

	board.Widgets = append(board.Widgets, w)
	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].TimeseriesWidget

	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "title-align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title-size.size", actualWidget.TitleSize.Size, expected.TitleSize.Size)
	assertEquals(t, "title-size.auto", actualWidget.TitleSize.Auto, expected.TitleSize.Auto)
	assertEquals(t, "title-text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
	assertEquals(t, "timeframe", actualWidget.Timeframe, expected.Timeframe)
	assertEquals(t, "legend", actualWidget.Legend, expected.Legend)
	assertTileDefEquals(t, actualWidget.TileDef, expected.TileDef)
}

func TestQueryValueWidget(t *testing.T) {
	board := createTestScreenboard(t)
	defer cleanUpScreenboard(t, board.Id)

	expected := datadog.Widget{}.QueryValueWidget
	expected.X = 1
	expected.Y = 1
	expected.Width = 20
	expected.Height = 30
	expected.Title = true
	expected.TitleAlign = "centre"
	expected.TitleSize = datadog.TextSize{Size: 16}
	expected.TitleText = "Test"
	expected.Timeframe = "1m"
	expected.TimeframeAggregator = "sum"
	expected.Aggregator = "min"
	expected.Query = "docker.containers.running"
	expected.MetricType = "standard"
	/* TODO: add test for conditional formats
	"conditional_formats": [{
		"comparator": ">",
		"color": "white_on_red",
		"custom_bg_color": null,
		"value": 1,
		"invert": false,
		"custom_fg_color": null}],
	*/
	expected.IsValidQuery = true
	expected.ResultCalcFunc = "raw"
	expected.Aggregator = "avg"
	expected.CalcFunc = "raw"

	w := datadog.Widget{QueryValueWidget: expected}

	board.Widgets = append(board.Widgets, w)
	if err := client.UpdateScreenboard(board); err != nil {
		t.Fatalf("Updating a screenboard failed: %s", err)
	}

	actual, err := client.GetScreenboard(board.Id)
	if err != nil {
		t.Fatalf("Retrieving a screenboard failed: %s", err)
	}

	actualWidget := actual.Widgets[0].QueryValueWidget

	assertEquals(t, "height", actualWidget.Height, expected.Height)
	assertEquals(t, "width", actualWidget.Width, expected.Width)
	assertEquals(t, "x", actualWidget.X, expected.X)
	assertEquals(t, "y", actualWidget.Y, expected.Y)
	assertEquals(t, "title", actualWidget.Title, expected.Title)
	assertEquals(t, "title-align", actualWidget.TitleAlign, expected.TitleAlign)
	assertEquals(t, "title-size.size", actualWidget.TitleSize.Size, expected.TitleSize.Size)
	assertEquals(t, "title-size.auto", actualWidget.TitleSize.Auto, expected.TitleSize.Auto)
	assertEquals(t, "title-text", actualWidget.TitleText, expected.TitleText)
	assertEquals(t, "type", actualWidget.Type, expected.Type)
	assertEquals(t, "timeframe", actualWidget.Timeframe, expected.Timeframe)
	assertEquals(t, "timeframe-aggregator", actualWidget.TimeframeAggregator, expected.TimeframeAggregator)
	assertEquals(t, "aggregator", actualWidget.Aggregator, expected.Aggregator)
	assertEquals(t, "query", actualWidget.Query, expected.Query)
	assertEquals(t, "is_valid_query", actualWidget.IsValidQuery, expected.IsValidQuery)
	assertEquals(t, "res_calc_func", actualWidget.ResultCalcFunc, expected.ResultCalcFunc)
	assertEquals(t, "aggr", actualWidget.Aggregator, expected.Aggregator)
}

func assertTileDefEquals(t *testing.T, actual datadog.TileDef, expected datadog.TileDef) {
	assertEquals(t, "num-events", len(actual.Events), len(expected.Events))
	assertEquals(t, "num-requests", len(actual.Requests), len(expected.Requests))
	assertEquals(t, "viz", actual.Viz, expected.Viz)

	for i, event := range actual.Events {
		assertEquals(t, "event-query", event.Query, expected.Events[i].Query)
	}

	for i, request := range actual.Requests {
		assertEquals(t, "request-query", request.Query, expected.Requests[i].Query)
		assertEquals(t, "request-type", request.Type, expected.Requests[i].Type)
	}
}

func assertEquals(t *testing.T, attribute string, a, b interface{}) {
	if a != b {
		t.Errorf("The two %s values '%v' and '%v' are not equal", attribute, a, b)
	}
}
