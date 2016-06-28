package helpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	credentialsPath = "/.azure/credentials.json"
)

// ToJSON returns the passed item as a pretty-printed JSON string. If any JSON error occurs,
// it returns the empty string.
func ToJSON(v interface{}) string {
	j, _ := json.MarshalIndent(v, "", "  ")
	return string(j)
}

// NewServicePrincipalTokenFromCredentials creates a new ServicePrincipalToken using values of the
// passed credentials map.
func NewServicePrincipalTokenFromCredentials(c map[string]string, scope string) (*azure.ServicePrincipalToken, error) {
	oauthConfig, err := azure.PublicCloud.OAuthConfigForTenant(c["tenantID"])
	if err != nil {
		panic(err)
	}
	return azure.NewServicePrincipalToken(*oauthConfig, c["clientID"], c["clientSecret"], scope)
}

// LoadCredentials reads credentials from a ~/.azure/credentials.json file. See the accompanying
// credentials_sample.json file for an example.
//
// Note: Storing crendentials in a local file must be secured and not shared. It is used here
// simply to reduce code in the examples.
func LoadCredentials() (map[string]string, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("ERROR: Unable to determine current user")
	}

	n := u.HomeDir + credentialsPath
	f, err := os.Open(n)
	if err != nil {
		return nil, fmt.Errorf("ERROR: Unable to locate or open Azure credentials at %s (%v)", n, err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("ERROR: Unable to read %s (%v)", n, err)
	}

	c := map[string]interface{}{}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, fmt.Errorf("ERROR: %s contained invalid JSON (%s)", n, err)
	}

	return ensureValueStrings(c), nil
}

func ensureValueStrings(mapOfInterface map[string]interface{}) map[string]string {
	mapOfStrings := make(map[string]string)
	for key, value := range mapOfInterface {
		mapOfStrings[key] = ensureValueString(value)
	}
	return mapOfStrings
}

func ensureValueString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
