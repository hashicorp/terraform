package oneandone

import (
	"errors"
	"net/http"
	"reflect"
	"time"
)

// Struct to hold the required information for accessing the API.
//
// Instances of this type contain the URL of the endpoint to access the API as well as the API access token to be used.
// They offer also all methods that allow to access the various objects that are returned by top level resources of
// the API.
type API struct {
	Endpoint string
	Client   *restClient
}

type ApiPtr struct {
	api *API
}

type idField struct {
	Id string `json:"id,omitempty"`
}

type typeField struct {
	Type string `json:"type,omitempty"`
}

type nameField struct {
	Name string `json:"name,omitempty"`
}

type descField struct {
	Description string `json:"description,omitempty"`
}

type countField struct {
	Count int `json:"count,omitempty"`
}

type serverIps struct {
	ServerIps []string `json:"server_ips"`
}

type servers struct {
	Servers []string `json:"servers"`
}

type ApiInstance interface {
	GetState() (string, error)
}

const (
	datacenterPathSegment      = "datacenters"
	dvdIsoPathSegment          = "dvd_isos"
	firewallPolicyPathSegment  = "firewall_policies"
	imagePathSegment           = "images"
	loadBalancerPathSegment    = "load_balancers"
	logPathSegment             = "logs"
	monitorCenterPathSegment   = "monitoring_center"
	monitorPolicyPathSegment   = "monitoring_policies"
	pingPathSegment            = "ping"
	pingAuthPathSegment        = "ping_auth"
	pricingPathSegment         = "pricing"
	privateNetworkPathSegment  = "private_networks"
	publicIpPathSegment        = "public_ips"
	rolePathSegment            = "roles"
	serverPathSegment          = "servers"
	serverAppliancePathSegment = "server_appliances"
	sharedStoragePathSegment   = "shared_storages"
	usagePathSegment           = "usages"
	userPathSegment            = "users"
	vpnPathSegment             = "vpns"
)

// Struct to hold the status of an API object.
//
// Values of this type are used to represent the status of API objects like servers, firewall policies and the like.
//
// The value of the "State" field can represent fixed states like "ACTIVE" or "POWERED_ON" but also transitional
// states like "POWERING_ON" or "CONFIGURING".
//
// For fixed states the "Percent" field is empty where as for transitional states it contains the progress of the
// transition in percent.
type Status struct {
	State   string `json:"state"`
	Percent int    `json:"percent"`
}

type statusState struct {
	State string `json:"state,omitempty"`
}

type Identity struct {
	idField
	nameField
}

type License struct {
	nameField
}

// Creates a new API instance.
//
// Explanations about given token and url information can be found online under the following url TODO add url!
func New(token string, url string) *API {
	api := new(API)
	api.Endpoint = url
	api.Client = newRestClient(token)
	return api
}

// Converts a given integer value into a pointer of the same type.
func Int2Pointer(input int) *int {
	result := new(int)
	*result = input
	return result
}

// Converts a given boolean value into a pointer of the same type.
func Bool2Pointer(input bool) *bool {
	result := new(bool)
	*result = input
	return result
}

// Performs busy-waiting for types that implement ApiInstance interface.
func (api *API) WaitForState(in ApiInstance, state string, sec time.Duration, count int) error {
	if in != nil {
		for i := 0; i < count; i++ {
			s, err := in.GetState()
			if err != nil {
				return err
			}
			if s == state {
				return nil
			}
			time.Sleep(sec * time.Second)
		}
		return errors.New(reflect.ValueOf(in).Type().String() + " operation timeout.")
	}
	return nil
}

// Waits until instance is deleted for types that implement ApiInstance interface.
func (api *API) WaitUntilDeleted(in ApiInstance) error {
	var err error
	for in != nil {
		_, err = in.GetState()
		if err != nil {
			if apiError, ok := err.(apiError); ok && apiError.httpStatusCode == http.StatusNotFound {
				return nil
			} else {
				return err
			}
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}
