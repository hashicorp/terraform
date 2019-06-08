package jsondiags

import (
	"encoding/json"

	"github.com/hashicorp/terraform/tfdiags"
)

var emptyArray = []byte{'[', ']'}

type jsonPos struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Byte   int `json:"byte"`
}

type jsonRange struct {
	Filename string  `json:"filename"`
	Start    jsonPos `json:"start"`
	End      jsonPos `json:"end"`
}

type jsonDiagnostic struct {
	Severity string     `json:"severity,omitempty"`
	Summary  string     `json:"summary,omitempty"`
	Detail   string     `json:"detail,omitempty"`
	Range    *jsonRange `json:"range,omitempty"`
}

// Diagnostics formats the given diagnostics as JSON.
func Diagnostics(diags tfdiags.Diagnostics) []byte {
	if len(diags) == 0 {
		return emptyArray
	}

	jsonDiags := make([]jsonDiagnostic, len(diags))
	for i, diag := range diags {
		var jsonDiag jsonDiagnostic

		switch diag.Severity() {
		case tfdiags.Error:
			jsonDiag.Severity = "error"
		case tfdiags.Warning:
			jsonDiag.Severity = "warning"
		}

		desc := diag.Description()
		jsonDiag.Summary = desc.Summary
		jsonDiag.Detail = desc.Detail

		ranges := diag.Source()
		if ranges.Subject != nil {
			subj := ranges.Subject
			jsonDiag.Range = &jsonRange{
				Filename: subj.Filename,
				Start: jsonPos{
					Line:   subj.Start.Line,
					Column: subj.Start.Column,
					Byte:   subj.Start.Byte,
				},
				End: jsonPos{
					Line:   subj.End.Line,
					Column: subj.End.Column,
					Byte:   subj.End.Byte,
				},
			}
		}

		jsonDiags[i] = jsonDiag
	}

	j, err := json.MarshalIndent(jsonDiags, "", "  ")
	if err != nil {
		// Should never happen because we fully-control the input here
		panic(err)
	}
	return j
}
