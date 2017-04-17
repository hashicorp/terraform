package iapi

/*
Currently to get something working and that can be refactored there is a lot of duplicate and overlapping decleration. In
part this is because when a variable is defined it is set to a default value. This has been problematic with having an attrs
struct that has all the variables. That struct then cannot be used to create the JSON for the create, without modification,
because it would try and set values that are not configurable via the API. i.e. for hosts "LastCheck" So to keep things moving
duplicate or near duplicate defintions of structs are being defined but can be revisted and refactored later and test will
be in place to ensure everything still works.
*/

//ServiceStruct stores service results
type ServiceStruct struct {
	Attrs ServiceAttrs `json:"attrs"`
	Joins struct{}     `json:"joins"`
	//	Meta  struct{}     `json:"meta"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type ServiceAttrs struct {
	CheckCommand string `json:"check_command"`
	//	CheckInterval float64       `json:"check_interval"`
	//	DisplayName   string        `json:"display_name"`
	//	Groups        []interface{} `json:"groups"`
	//Name string `json:"name"`
	//	Templates     []string      `json:"templates"`
	//	Type string `json:"type"`
	//	Vars          interface{}   `json:"vars"`
}

// CheckcommandStruct is a struct used to store results from an Icinga2 Checkcommand API call.
type CheckcommandStruct struct {
	Name  string            `json:"name"`
	Type  string            `json:"type"`
	Attrs CheckcommandAttrs `json:"attrs"`
	Joins struct{}          `json:"joins"`
	Meta  struct{}          `json:"meta"`
}

type CheckcommandAttrs struct {
	Arguments interface{} `json:"arguments"`
	Command   []string    `json:"command"`
	Templates []string    `json:"templates"`
	//	Env       interface{} `json:"env"`   				// Available to be set but not supported yet
	//	Package   string      `json:"package"`   		// Available to be set but not supported yet
	//	Timeout   float64     `json:"timeout"`   		// Available to be set but not supported yet
	//	Vars      interface{} `json:"vars"`   			// Available to be set but not supported yet
	//	Zone      string      `json:"zone"`   			// Available to be set but not supported yet
}

// HostgroupStruct is a struct used to store results from an Icinga2 HostGroup API Call. The content are also used to generate the JSON for the CreateHost call
type HostgroupStruct struct {
	Name  string         `json:"name"`
	Type  string         `json:"type"`
	Attrs HostgroupAttrs `json:"attrs"`
	Meta  struct{}       `json:"meta"`
	Joins struct{}       `json:"stuct"`
}

// HostgroupAttrs ...
type HostgroupAttrs struct {
	ActionURL   string   `json:"action_url"`
	DisplayName string   `json:"display_name"`
	Groups      []string `json:"groups"`
	Notes       string   `json:"notes"`
	NotesURL    string   `json:"notes_url"`
	Templates   []string `json:"templates"`
}

// HostStruct is a struct used to store results from an Icinga2 Host API Call. The content are also used to generate the JSON for the CreateHost call
type HostStruct struct {
	Name  string    `json:"name"`
	Type  string    `json:"type"`
	Attrs HostAttrs `json:"attrs"`
	Meta  struct{}  `json:"meta"`
	Joins struct{}  `json:"stuct"`
}

// HostAttrs This is struct lists the attributes that can be set during a CreateHost call. The contents of the struct is converted into JSON
type HostAttrs struct {
	ActionURL    string      `json:"action_url"`
	Address      string      `json:"address"`
	Address6     string      `json:"address6"`
	CheckCommand string      `json:"check_command"`
	DisplayName  string      `json:"display_name"`
	Groups       []string    `json:"groups"`
	Notes        string      `json:"notes"`
	NotesURL     string      `json:"notes_url"`
	Templates    []string    `json:"templates"`
	Vars         interface{} `json:"vars"`
}

// APIResult Stores the results from NewApiRequest
type APIResult struct {
	Error       float64 `json:"error"`
	ErrorString string
	Status      string      `json:"Status"`
	Code        int         `json:"Code"`
	Results     interface{} `json:"results"`
}

// APIStatus stores the results of an Icinga2 API Status Call
type APIStatus struct {
	Results []struct {
		Name     string   `json:"name"`
		Perfdata []string `json:"perfdata"`
		Status   struct {
			API struct {
				ConnEndpoints       []interface{} `json:"conn_endpoints"`
				Identity            string        `json:"identity"`
				NotConnEndpoints    []interface{} `json:"not_conn_endpoints"`
				NumConnEndpoints    int           `json:"num_conn_endpoints"`
				NumEndpoints        int           `json:"num_endpoints"`
				NumNotConnEndpoints int           `json:"num_not_conn_endpoints"`
				Zones               struct {
					Master struct {
						ClientLogLag int      `json:"client_log_lag"`
						Connected    bool     `json:"connected"`
						Endpoints    []string `json:"endpoints"`
						ParentZone   string   `json:"parent_zone"`
					} `json:"master"`
				} `json:"zones"`
			} `json:"api"`
		} `json:"status"`
	} `json:"results"`
}
