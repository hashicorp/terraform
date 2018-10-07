package mode

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"text/template"
)

func TestRemoteInventTemplateGenerates(t *testing.T) {
	originalHosts := []string{"host1", "host2"}
	templateData := inventoryTemplateRemoteData{
		Hosts:  ensureLocalhostInHosts(originalHosts),
		Groups: []string{"group1", "group2"},
	}

	tpl := template.Must(template.New("hosts").Parse(inventoryTemplateRemote))
	var buf bytes.Buffer
	err := tpl.Execute(&buf, templateData)
	if err != nil {
		t.Fatalf("Expected template to generate correctly but received: %v", err)
	}
	templateBody := buf.String()
	if strings.Index(templateBody, fmt.Sprintf("[%s]\nlocalhost ansible_connection=local\n%s",
		templateData.Groups[0],
		originalHosts[0])) < 0 {
		t.Fatalf("Expected a group with alias in generated template but got: %s", templateBody)
	}
}
