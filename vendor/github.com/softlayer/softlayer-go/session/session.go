/**
 * Copyright 2016 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package session

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/softlayer/softlayer-go/config"
	"github.com/softlayer/softlayer-go/sl"
)

// DefaultEndpoint is the default endpoint for API calls, when no override
// is provided.
const DefaultEndpoint = "https://api.softlayer.com/rest/v3"

// TransportHandler
type TransportHandler interface {
	// DoRequest is the protocol-specific handler for making API requests.
	//
	// sess is a reference to the current session object, where authentication and
	// endpoint information can be found.
	//
	// service and method are the SoftLayer service name and method name, exactly as they
	// are documented at http://sldn.softlayer.com/reference/softlayerapi (i.e., with the
	// 'SoftLayer_' prefix and properly cased.
	//
	// args is a slice of arguments required for the service method being invoked.  The
	// types of each argument varies. See the method definition in the services package
	// for the expected type of each argument.
	//
	// options is an sl.Options struct, containing any mask, filter, or result limit values
	// to be applied.
	//
	// pResult is a pointer to a variable to be populated with the result of the API call.
	// DoRequest should ensure that the native API response (i.e., XML or JSON) is correctly
	// unmarshaled into the result structure.
	//
	// A sl.Error is returned, and can be (with a type assertion) inspected for details of
	// the error (http code, API error message, etc.), or simply handled as a generic error,
	// (in which case no type assertion would be necessary)
	DoRequest(
		sess *Session,
		service string,
		method string,
		args []interface{},
		options *sl.Options,
		pResult interface{}) error
}

const DefaultTimeout = time.Second * 60

// Session stores the information required for communication with the SoftLayer
// API
type Session struct {
	// UserName is the name of the SoftLayer API user
	UserName string

	// ApiKey is the secret for making API calls
	APIKey string

	// Endpoint is the SoftLayer API endpoint to communicate with
	Endpoint string

	// UserId is the user id for token-based authentication
	UserId int

	// AuthToken is the token secret for token-based authentication
	AuthToken string

	// Debug controls logging of request details (URI, parameters, etc.)
	Debug bool

	// The handler whose DoRequest() function will be called for each API request.
	// Handles the request and any response parsing specific to the desired protocol
	// (e.g., REST).  Set automatically for a new Session, based on the
	// provided Endpoint.
	TransportHandler TransportHandler

	// Timeout specifies a time limit for http requests made by this
	// session. Requests that take longer that the specified timeout
	// will result in an error.
	Timeout time.Duration
}

// New creates and returns a pointer to a new session object.  It takes up to
// three parameters, all of which are optional.  If specified, they will be
// interpreted in the following sequence:
//
// 1. UserName
// 2. Api Key
// 3. Endpoint
// 4. Timeout
//
// If one or more are omitted, New() will attempt to retrieve these values from
// the environment, and the ~/.softlayer config file, in that order.
func New(args ...interface{}) *Session {
	keys := map[string]int{"username": 0, "api_key": 1, "endpoint_url": 2, "timeout": 3}
	values := []string{"", "", "", ""}

	for i := 0; i < len(args); i++ {
		values[i] = args[i].(string)
	}

	// Default to the environment variables

	// Prioritize SL_USERNAME
	envFallback("SL_USERNAME", &values[keys["username"]])
	envFallback("SOFTLAYER_USERNAME", &values[keys["username"]])

	// Prioritize SL_API_KEY
	envFallback("SL_API_KEY", &values[keys["api_key"]])
	envFallback("SOFTLAYER_API_KEY", &values[keys["api_key"]])

	// Prioritize SL_ENDPOINT_URL
	envFallback("SL_ENDPOINT_URL", &values[keys["endpoint_url"]])
	envFallback("SOFTLAYER_ENDPOINT_URL", &values[keys["endpoint_url"]])

	envFallback("SL_TIMEOUT", &values[keys["timeout"]])
	envFallback("SOFTLAYER_TIMEOUT", &values[keys["timeout"]])

	// Read ~/.softlayer for configuration
	var homeDir string
	u, err := user.Current()
	if err != nil {
		for _, name := range []string{"HOME", "USERPROFILE"} { // *nix, windows
			if dir := os.Getenv(name); dir != "" {
				homeDir = dir
				break
			}
		}
	} else {
		homeDir = u.HomeDir
	}

	if homeDir != "" {
		configPath := fmt.Sprintf("%s/.softlayer", homeDir)
		if _, err = os.Stat(configPath); !os.IsNotExist(err) {
			// config file exists
			file, err := config.LoadFile(configPath)
			if err != nil {
				log.Println(fmt.Sprintf("[WARN] session: Could not parse %s : %s", configPath, err))
			} else {
				for k, v := range keys {
					value, ok := file.Get("softlayer", k)
					if ok && values[v] == "" {
						values[v] = value
					}
				}
			}
		}
	} else {
		log.Println("[WARN] session: home dir could not be determined. Skipping read of ~/.softlayer.")
	}

	endpointURL := values[keys["endpoint_url"]]
	if endpointURL == "" {
		endpointURL = DefaultEndpoint
	}

	sess := &Session{
		UserName: values[keys["username"]],
		APIKey:   values[keys["api_key"]],
		Endpoint: endpointURL,
	}

	timeout := values[keys["timeout"]]
	if timeout != "" {
		timeoutDuration, err := time.ParseDuration(fmt.Sprintf("%ss", timeout))
		if err == nil {
			sess.Timeout = timeoutDuration
		}
	}

	return sess
}

// DoRequest hands off the processing to the assigned transport handler. It is
// normally called internally by the service objects, but is exported so that it can
// be invoked directly by client code in exceptional cases where direct control is
// needed over one of the parameters.
//
// For a description of parameters, see TransportHandler.DoRequest in this package
func (r *Session) DoRequest(service string, method string, args []interface{}, options *sl.Options, pResult interface{}) error {
	if r.TransportHandler == nil {
		r.TransportHandler = getDefaultTransport(r.Endpoint)
	}

	return r.TransportHandler.DoRequest(r, service, method, args, options, pResult)
}

func envFallback(keyName string, value *string) {
	if *value == "" {
		*value = os.Getenv(keyName)
	}
}

func getDefaultTransport(endpointURL string) TransportHandler {
	var transportHandler TransportHandler

	if strings.Contains(endpointURL, "/xmlrpc/") {
		transportHandler = &XmlRpcTransport{}
	} else {
		transportHandler = &RestTransport{}
	}

	return transportHandler
}
