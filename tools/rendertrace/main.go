package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"
)

func main() {
	os.Exit(realMain(os.Args))
}

func realMain(args []string) int {
	// TODO: proper argument parsing
	filename := args[1]

	var rawTrace []RawSpan
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(src, &rawTrace)
	if err != nil {
		log.Fatal(err)
	}

	minTime := time.Now()
	var totalDuration time.Duration

	spans := make(map[string]*Span)
	var rootSpans []*Span
	for _, raw := range rawTrace {
		span := Span{
			Operation: raw.Operation,
			ParentID:  raw.ParentID,
			StartTime: raw.StartTime,
			Duration:  raw.Duration,
		}
		if len(raw.Log) != 0 && len(raw.Log[0].Values) == 1 {
			for _, v := range raw.Log[0].Values {
				if v, ok := v.(string); ok {
					span.Object = v
				}
			}
		}
		if span.Operation == "planModule" && span.Object == "" {
			span.Object = "(root module)"
		}
		spans[raw.ID] = &span
		if raw.ParentID == "" {
			rootSpans = append(rootSpans, &span)
		}

		if raw.StartTime.Before(minTime) {
			minTime = raw.StartTime
		}
	}

	for _, span := range spans {
		if span.ParentID != "" {
			spans[span.ParentID].Children = append(spans[span.ParentID].Children, span)
		}
	}
	for _, span := range spans {
		sort.Slice(span.Children, func(i, j int) bool {
			return span.Children[i].StartTime.Before(span.Children[j].StartTime)
			//iEnd := span.Children[i].StartTime.Add(span.Children[i].Duration)
			//jEnd := span.Children[j].StartTime.Add(span.Children[j].Duration)
			//return iEnd.Before(jEnd)
		})

		endTimeRel := span.StartTime.Add(span.Duration).Sub(minTime)
		if endTimeRel > totalDuration {
			totalDuration = endTimeRel
		}
	}
	sort.Slice(rootSpans, func(i, j int) bool {
		return rootSpans[i].StartTime.Before(rootSpans[j].StartTime)
	})

	var renderSpans func([]*Span, int)
	renderSpans = func(spans []*Span, depth int) {
		for _, span := range spans {
			startTimeRel := span.StartTime.Sub(minTime)
			startTimePct := (float64(startTimeRel) * 100.0) / float64(totalDuration)
			durationPct := (float64(span.Duration) * 100.0) / float64(totalDuration)
			remainPct := 100.0 - durationPct - startTimePct

			fmt.Println("<tr class='span'>")
			fmt.Printf(
				`<th><div style="padding-left: %dpx;padding-right:10px"><div class="operation-name">%s</div><div class="object-name">%s</div></div></th>`,
				depth*20,
				html.EscapeString(span.Operation),
				html.EscapeString(span.Object),
			)
			fmt.Println("\n<td>")
			fmt.Println("<table class='timespan' width='100%' border=0 cellpadding=0 cellspacing=0><tr>")
			fmt.Printf("<td width='%f%%' class='space'></td>\n", startTimePct)
			fmt.Printf("<td width='%f%%' class='bar'></td>\n", durationPct)
			fmt.Printf("<td width='%f%%' class='space'></td>\n", remainPct)
			fmt.Println("</tr></table>")
			fmt.Println("</td>")
			fmt.Println("</tr>")
			renderSpans(span.Children, depth+1)
		}
	}

	// Table-based-layouting like it's 1995
	fmt.Println("<!doctype html>")
	fmt.Println("<style>")
	fmt.Print(inlineCSS)
	fmt.Println("</style>")
	fmt.Println("<table width='100%' border=0 cellpadding=0 cellspacing=0><tbody>")
	renderSpans(rootSpans, 0)
	fmt.Println("</tbody></table>")

	return 0
}

type Span struct {
	Operation string
	Object    string
	ParentID  string
	StartTime time.Time
	Duration  time.Duration
	Children  []*Span
}

type RawSpan struct {
	ID        string        `json:"id"`
	Operation string        `json:"operation"`
	StartTime time.Time     `json:"startTime"`
	Duration  time.Duration `json:"duration"`
	ParentID  string        `json:"parentId,omitempty"`
	Log       []RawSpanLog  `json:"log,omitempty"`
}

type RawSpanLog struct {
	Time   time.Time              `json:"time"`
	Values map[string]interface{} `json:"values"`
}

const inlineCSS = `
body {
  font-family: sans-serif;
  font-size: 13px;
}
table {
  width: 100%;
  border-collapse: collapse;
  border 1px solid #aaaaaa;
}
.span {
	border-bottom 1px solid #aaaaaa;
}
.span:nth-child(even) {background: #DDD}
.span:nth-child(odd) {background: #FFF}
th {
  width: 10%;
  text-align: left;
}
.timespan {
	height: 30px;
}
.span:nth-child(even) .bar {
	background: #00e;
}
.span:nth-child(odd) .bar {
	background: #00f;
}
.operation-name {
  font-weight: bold;
}
.object-name {
  font-weight: normal;
}
`
