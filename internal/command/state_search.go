package command

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// StateSearchCommand implements the "terraform state find" command
type StateSearchCommand struct {
	Meta
}

// SearchResult represents a resource matching the search
type SearchResult struct {
	Address    string                 `json:"address,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Module     string                 `json:"module,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	MatchType  string                 `json:"match_type"`
	MatchField string                 `json:"match_field,omitempty"`
}

// Run executes the state find command
func (c *StateSearchCommand) Run(args []string) int {
	var (
		statePath       string
		format          string
		exactMatch      bool
		caseInsensitive bool
	)

	cmdFlags := c.Meta.extendedFlagSet("state find")
	cmdFlags.StringVar(&statePath, "state", "", "Path to state file")
	cmdFlags.StringVar(&format, "format", "text", "Output format (text, json)")
	cmdFlags.BoolVar(&exactMatch, "exact", false, "Exact string match only")
	cmdFlags.BoolVar(&caseInsensitive, "ignore-case", true, "Case-insensitive search (default true)")

	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing flags: %s", err))
		return 1
	}

	args = cmdFlags.Args()
	if len(args) == 0 {
		c.Ui.Error("Search keyword required")
		c.Ui.Error("")
		c.Ui.Error(c.Help())
		return 1
	}

	keyword := args[0]

	// Determine state path
	if statePath == "" {
		statePath = "terraform.tfstate"
	}

	// Read and parse state file
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading state file: %s", err))
		return 1
	}

	var stateJSON map[string]interface{}
	if err := json.Unmarshal(stateData, &stateJSON); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing state file: %s", err))
		return 1
	}

	// Search through the state
	results := c.searchStateJSON(stateJSON, keyword, exactMatch, caseInsensitive)

	if len(results) == 0 {
		c.Ui.Output(fmt.Sprintf("No resources found matching '%s'", keyword))
		return 0
	}

	// Output results in the requested format
	c.outputSearchResults(results, format, keyword)
	return 0
}

func (c *StateSearchCommand) searchStateJSON(stateJSON map[string]interface{}, keyword string, exactMatch, caseInsensitive bool) []SearchResult {
	var results []SearchResult

	searchKeyword := keyword
	if caseInsensitive {
		searchKeyword = strings.ToLower(keyword)
	}

	// Extract resources from the state
	if resources, ok := stateJSON["resources"].([]interface{}); ok {
		for _, res := range resources {
			if resMap, ok := res.(map[string]interface{}); ok {
				results = append(results, c.searchResource(resMap, "", searchKeyword, exactMatch, caseInsensitive)...)
			}
		}
	}

	// Extract resources from modules if present
	if modules, ok := stateJSON["modules"].([]interface{}); ok {
		for _, mod := range modules {
			if modMap, ok := mod.(map[string]interface{}); ok {
				path := ""
				if p, ok := modMap["path"].([]interface{}); ok && len(p) > 0 {
					pathParts := []string{}
					for _, part := range p {
						if s, ok := part.(string); ok {
							pathParts = append(pathParts, s)
						}
					}
					path = strings.Join(pathParts, ".")
				}

				if resources, ok := modMap["resources"].(map[string]interface{}); ok {
					for resAddr, resData := range resources {
						if resMap, ok := resData.(map[string]interface{}); ok {
							results = append(results, c.searchResourceWithAddress(resAddr, resMap, path, searchKeyword, exactMatch, caseInsensitive)...)
						}
					}
				}
			}
		}
	}

	return results
}

func (c *StateSearchCommand) searchResource(resMap map[string]interface{}, modulePath string, keyword string, exactMatch, caseInsensitive bool) []SearchResult {
	var results []SearchResult

	// Extract resource information
	resType, _ := resMap["type"].(string)
	resName, _ := resMap["name"].(string)
	address := resType + "." + resName

	if c.matchesKeyword(resType, keyword, exactMatch, caseInsensitive) {
		results = append(results, SearchResult{
			Address:   address,
			Type:      resType,
			Name:      resName,
			Module:    modulePath,
			MatchType: "type",
		})
		return results
	}

	if c.matchesKeyword(resName, keyword, exactMatch, caseInsensitive) {
		results = append(results, SearchResult{
			Address:   address,
			Type:      resType,
			Name:      resName,
			Module:    modulePath,
			MatchType: "name",
		})
		return results
	}

	// Search in instances
	if instances, ok := resMap["instances"].([]interface{}); ok {
		for _, inst := range instances {
			if instMap, ok := inst.(map[string]interface{}); ok {
				if attrs, ok := instMap["attributes"].(map[string]interface{}); ok {
					for key, val := range attrs {
						if c.matchesKeyword(fmt.Sprintf("%v", val), keyword, exactMatch, caseInsensitive) {
							results = append(results, SearchResult{
								Address:    address,
								Type:       resType,
								Name:       resName,
								Module:     modulePath,
								Attributes: attrs,
								MatchType:  "attribute",
								MatchField: key,
							})
							return results
						}
					}
				}
			}
		}
	}

	return results
}

func (c *StateSearchCommand) searchResourceWithAddress(address string, resMap map[string]interface{}, modulePath string, keyword string, exactMatch, caseInsensitive bool) []SearchResult {
	var results []SearchResult

	// Check if resource address matches
	if c.matchesKeyword(address, keyword, exactMatch, caseInsensitive) {
		results = append(results, SearchResult{
			Address:   address,
			Module:    modulePath,
			MatchType: "address",
		})
		return results
	}

	// Extract type and name from address (type.name format)
	parts := strings.Split(address, ".")
	if len(parts) >= 2 {
		resType := parts[0]
		resName := strings.Join(parts[1:], ".")

		if c.matchesKeyword(resType, keyword, exactMatch, caseInsensitive) {
			results = append(results, SearchResult{
				Address:   address,
				Type:      resType,
				Name:      resName,
				Module:    modulePath,
				MatchType: "type",
			})
			return results
		}

		if c.matchesKeyword(resName, keyword, exactMatch, caseInsensitive) {
			results = append(results, SearchResult{
				Address:   address,
				Type:      resType,
				Name:      resName,
				Module:    modulePath,
				MatchType: "name",
			})
			return results
		}
	}

	// Search in instances
	if instances, ok := resMap["instances"].(map[string]interface{}); ok {
		for _, inst := range instances {
			if instMap, ok := inst.(map[string]interface{}); ok {
				if attrs, ok := instMap["attributes"].(map[string]interface{}); ok {
					foundMatch := false
					var matchField string
					for key, val := range attrs {
						if c.matchesKeyword(fmt.Sprintf("%v", val), keyword, exactMatch, caseInsensitive) {
							foundMatch = true
							matchField = key
							break
						}
					}
					if foundMatch {
						results = append(results, SearchResult{
							Address:    address,
							Module:     modulePath,
							Attributes: attrs,
							MatchType:  "attribute",
							MatchField: matchField,
						})
						return results
					}
				}
			}
		}
	}

	return results
}

func (c *StateSearchCommand) matchesKeyword(text string, keyword string, exactMatch, caseInsensitive bool) bool {
	if caseInsensitive {
		text = strings.ToLower(text)
	}

	if exactMatch {
		return text == keyword
	}
	return strings.Contains(text, keyword)
}

func (c *StateSearchCommand) outputSearchResults(results []SearchResult, format string, keyword string) {
	switch format {
	case "json":
		c.outputJSON(results)
	default:
		c.outputText(results, keyword)
	}
}

func (c *StateSearchCommand) outputText(results []SearchResult, keyword string) {
	c.Ui.Output(fmt.Sprintf("\nSearching for '%s' in state...", keyword))
	c.Ui.Output(fmt.Sprintf("Found %d resource(s):\n", len(results)))
	c.Ui.Output("=====================================\n")

	for _, result := range results {
		c.Ui.Output(fmt.Sprintf("Address: %s", result.Address))
		if result.Type != "" {
			c.Ui.Output(fmt.Sprintf("  Type: %s", result.Type))
		}
		if result.Name != "" {
			c.Ui.Output(fmt.Sprintf("  Name: %s", result.Name))
		}
		if result.Module != "" {
			c.Ui.Output(fmt.Sprintf("  Module: %s", result.Module))
		}
		c.Ui.Output(fmt.Sprintf("  Match: %s", result.MatchType))
		if result.MatchField != "" {
			c.Ui.Output(fmt.Sprintf("  Matched Field: %s", result.MatchField))
		}
		if len(result.Attributes) > 0 {
			c.Ui.Output(fmt.Sprintf("  Attributes: %d field(s)", len(result.Attributes)))
		}
		c.Ui.Output("")
	}

	c.Ui.Output("=====================================")
	c.Ui.Output(fmt.Sprintf("Total: %d resource(s) matched", len(results)))
}

func (c *StateSearchCommand) outputJSON(results []SearchResult) {
	output := map[string]interface{}{
		"results": results,
		"count":   len(results),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error encoding JSON: %s", err))
		return
	}

	c.Ui.Output(string(data))
}

// Help returns the help text
func (c *StateSearchCommand) Help() string {
	helpText := `
Usage: terraform state find [options] <keyword>

Search for resources in the Terraform state file by matching keywords
against resource addresses, types, and attribute values.

This command is useful for finding specific resources without having to
manually review the entire state file. It supports both simple keyword
matching and exact matching.

Options:
  -state=<path>          Path to the state file (default: auto-detected)
  -format=text|json      Output format (default: text)
  -exact                 Exact string match only (default: substring match)
  -ignore-case=true|false Case-insensitive search (default: true)

Examples:
  terraform state find aws_instance
  terraform state find my-resource-name
  terraform state find "10.0.0.1"
  terraform state find -exact web-server
  terraform state find -format=json security_group
  terraform state find -state=/path/to/state postgres

Search Behavior:
  By default, the search looks for the keyword in:
  - Resource types (e.g., "aws_instance", "aws_s3_bucket")
  - Resource names (e.g., "web_server", "database")
  - Resource addresses (e.g., "aws_instance.web[0]")
  - Resource attribute values (recursive JSON search)

  With -exact flag, only exact matches are returned.
  With -ignore-case=false, search is case-sensitive.
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns a short description
func (c *StateSearchCommand) Synopsis() string {
	return "Search for resources in the state file"
}
