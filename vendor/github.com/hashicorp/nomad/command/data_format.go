package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"text/template"
)

//DataFormatter is a transformer of the data.
type DataFormatter interface {
	// TransformData should return transformed string data.
	TransformData(interface{}) (string, error)
}

// DataFormat returns the data formatter specified format.
func DataFormat(format, tmpl string) (DataFormatter, error) {
	switch format {
	case "json":
		if len(tmpl) > 0 {
			return nil, fmt.Errorf("json format does not support template option.")
		}
		return &JSONFormat{}, nil
	case "template":
		return &TemplateFormat{tmpl}, nil
	}
	return nil, fmt.Errorf("Unsupported format is specified.")
}

type JSONFormat struct {
}

// TransformData returns JSON format string data.
func (p *JSONFormat) TransformData(data interface{}) (string, error) {
	out, err := json.MarshalIndent(&data, "", "    ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}

type TemplateFormat struct {
	tmpl string
}

// TransformData returns template format string data.
func (p *TemplateFormat) TransformData(data interface{}) (string, error) {
	var out io.Writer = new(bytes.Buffer)
	if len(p.tmpl) == 0 {
		return "", fmt.Errorf("template needs to be specified the golang templates.")
	}

	t, err := template.New("format").Parse(p.tmpl)
	if err != nil {
		return "", err
	}

	err = t.Execute(out, data)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(out), nil
}
