package mode

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"text/template"
)

func TestLocalInventoryTemplateGeneratesWithAlias(t *testing.T) {

	templateData := inventoryTemplateLocalData{
		Hosts: []inventoryTemplateLocalDataHost{
			inventoryTemplateLocalDataHost{
				Alias:       "testBox",
				AnsibleHost: "10.1.100.34",
			},
		},
		Groups: []string{"group1"},
	}

	tpl := template.Must(template.New("hosts").Parse(inventoryTemplateLocal))
	var buf bytes.Buffer
	err := tpl.Execute(&buf, templateData)
	if err != nil {
		t.Fatalf("Expected template to generate correctly but received: %v", err)
	}
	templateBody := buf.String()
	if strings.Index(templateBody, fmt.Sprintf("[%s]\n%s ansible_host",
		templateData.Groups[0],
		templateData.Hosts[0].Alias)) < 0 {
		t.Fatalf("Expected a group with alias in generated template but got:\n%s", templateBody)
	}
}

func TestLocalInventoryTemplateGeneratesWithoutAlias(t *testing.T) {

	// please refer to mode_local.go writeInventory for details:
	templateData := inventoryTemplateLocalData{
		Hosts: []inventoryTemplateLocalDataHost{
			inventoryTemplateLocalDataHost{
				Alias: "10.1.100.34",
			},
		},
		Groups: []string{"group1"},
	}

	tpl := template.Must(template.New("hosts").Parse(inventoryTemplateLocal))
	var buf bytes.Buffer
	err := tpl.Execute(&buf, templateData)
	if err != nil {
		t.Fatalf("Expected template to generate correctly but received: %v", err)
	}

	templateBody := buf.String()
	if strings.Index(templateBody, fmt.Sprintf("[%s]\n%s",
		templateData.Groups[0],
		templateData.Hosts[0].Alias)) < 0 {
		t.Fatalf("Expected a group with alias in generated template but got: %s", templateBody)
	}
	if strings.Index(templateBody, "ansible_host") > -1 {
		t.Fatalf("Did not expect ansible_host in generated template but got: %s", templateBody)
	}

}
