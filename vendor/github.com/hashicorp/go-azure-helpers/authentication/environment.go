package authentication

import (
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
)

// DetermineEnvironment determines what the Environment name is within
// the Azure SDK for Go and then returns the association environment, if it exists.
func DetermineEnvironment(name string) (*azure.Environment, error) {
	// detect cloud from environment
	env, envErr := azure.EnvironmentFromName(name)

	if envErr != nil {
		// try again with wrapped value to support readable values like german instead of AZUREGERMANCLOUD
		wrapped := fmt.Sprintf("AZURE%sCLOUD", name)
		env, envErr = azure.EnvironmentFromName(wrapped)
		if envErr != nil {
			return nil, fmt.Errorf("An Azure Environment with name %q was not found: %+v", name, envErr)
		}
	}

	return &env, nil
}

// LoadEnvironmentFromUrl attempts to load the specified environment from the endpoint.
// if the endpoint is an empty string, or an environment can't be
// found at the endpoint url then an error is returned
func LoadEnvironmentFromUrl(endpoint string) (*azure.Environment, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("Endpoint was not set!")
	}

	env, err := azure.EnvironmentFromURL(endpoint)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving Environment from Endpoint %q: %+v", endpoint, err)
	}

	return &env, nil
}

func normalizeEnvironmentName(input string) string {
	// Environment is stored as `Azure{Environment}Cloud`
	output := strings.ToLower(input)
	output = strings.TrimPrefix(output, "azure")
	output = strings.TrimSuffix(output, "cloud")

	// however Azure Public is `AzureCloud` in the CLI Profile and not `AzurePublicCloud`.
	if output == "" {
		return "public"
	}
	return output
}
