package rancher

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rancher/go-rancher/v2"
)

const (
	stateRemoved = "removed"
	statePurged  = "purged"
)

// GetActiveOrchestration get the name of the active orchestration for a environment
func getActiveOrchestration(project *client.Project) string {
	return project.Orchestration
}

func removed(state string) bool {
	return state == stateRemoved || state == statePurged
}

func splitID(id string) (envID, resourceID string) {
	if strings.Contains(id, "/") {
		return id[0:strings.Index(id, "/")], id[strings.Index(id, "/")+1:]
	}
	return "", id
}

// NewListOpts wraps around client.NewListOpts()
func NewListOpts() *client.ListOpts {
	return client.NewListOpts()
}

func populateProjectTemplateIDs(config *Config) error {
	cli, err := config.GlobalClient()
	if err != nil {
		return err
	}

	for projectTemplate := range defaultProjectTemplates {
		templates, err := cli.ProjectTemplate.List(&client.ListOpts{
			Filters: map[string]interface{}{
				"isPublic": true,
				"name":     projectTemplate,
				"sort":     "created",
			},
		})
		if err != nil {
			return err
		}

		if len(templates.Data) > 0 {
			defaultProjectTemplates[projectTemplate] = templates.Data[0].Id
		}
	}
	return nil
}

func addHostLabels(command string, labels map[string]interface{}) string {
	result := []string{}
	hostLabels := url.Values{}

	if len(labels) == 0 {
		return command
	}

	tokenizedCommand := strings.Split(command, " ")
	if len(tokenizedCommand) > 0 {
		result = append(result, tokenizedCommand[:3]...)
		for k, v := range labels {
			hostLabels.Add(k, v.(string))
		}
		strHostLabels := hostLabels.Encode()
		result = append(result, "-e", fmt.Sprintf("CATTLE_HOST_LABELS='%s'", strHostLabels))
		result = append(result, tokenizedCommand[3:]...)
	}

	return strings.Join(result, " ")
}
