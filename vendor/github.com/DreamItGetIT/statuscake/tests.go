package statuscake

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

const queryStringTag = "querystring"

// Test represents a statuscake Test
type Test struct {
	// ThiTestID is an int, use this to get more details about this test. If not provided will insert a new check, else will update
	TestID int `json:"TestID" querystring:"TestID" querystringoptions:"omitempty"`

	// Sent tfalse To Unpause and true To Pause.
	Paused bool `json:"Paused" querystring:"Paused"`

	// Website name. Tags are stripped out
	WebsiteName string `json:"WebsiteName" querystring:"WebsiteName"`

	// Test location, either an IP (for TCP and Ping) or a fully qualified URL for other TestTypes
	WebsiteURL string `json:"WebsiteURL" querystring:"WebsiteURL"`

	// A Port to use on TCP Tests
	Port int `json:"Port" querystring:"Port"`

	// Contact group ID - will return int of contact group used else 0
	ContactID int `json:"ContactID"`

	// Current status at last test
	Status string `json:"Status"`

	// 7 Day Uptime
	Uptime float64 `json:"Uptime"`

	// Any test locations seperated by a comma (using the Node Location IDs)
	NodeLocations []string `json:"NodeLocations" querystring:"NodeLocations"`

	// Timeout in an int form representing seconds.
	Timeout int `json:"Timeout" querystring:"Timeout"`

	// A URL to ping if a site goes down.
	PingURL string `json:"PingURL" querystring:"PingURL"`

	Confirmation int `json:"Confirmationi,string" querystring:"Confirmation"`

	// The number of seconds between checks.
	CheckRate int `json:"CheckRate" querystring:"CheckRate"`

	// A Basic Auth User account to use to login
	BasicUser string `json:"BasicUser" querystring:"BasicUser"`

	// If BasicUser is set then this should be the password for the BasicUser
	BasicPass string `json:"BasicPass" querystring:"BasicPass"`

	// Set 1 to enable public reporting, 0 to disable
	Public int `json:"Public" querystring:"Public"`

	// A URL to a image to use for public reporting
	LogoImage string `json:"LogoImage" querystring:"LogoImage"`

	// Set to 0 to use branding (default) or 1 to disable public reporting branding
	Branding int `json:"Branding" querystring:"Branding"`

	// Used internally by the statuscake API
	WebsiteHost string `json:"WebsiteHost"`

	// Enable virus checking or not. 1 to enable
	Virus int `json:"Virus" querystring:"Virus"`

	// A string that should either be found or not found.
	FindString string `json:"FindString" querystring:"FindString"`

	// If the above string should be found to trigger a alert. true will trigger if FindString found
	DoNotFind bool `json:"DoNotFind" querystring:"DoNotFind"`

	// What type of test type to use. Accepted values are HTTP, TCP, PING
	TestType string `json:"TestType" querystring:"TestType"`

	// Use 1 to TURN OFF real browser testing
	RealBrowser int `json:"RealBrowser" querystring:"RealBrowser"`

	// How many minutes to wait before sending an alert
	TriggerRate int `json:"TriggerRate" querystring:"TriggerRate"`

	// Tags should be seperated by a comma - no spacing between tags (this,is,a set,of,tags)
	TestTags string `json:"TestTags" querystring:"TestTags"`

	// Comma Seperated List of StatusCodes to Trigger Error on (on Update will replace, so send full list each time)
	StatusCodes string `json:"StatusCodes" querystring:"StatusCodes"`
}

// Validate checks if the Test is valid. If it's invalid, it returns a ValidationError with all invalid fields. It returns nil otherwise.
func (t *Test) Validate() error {
	e := make(ValidationError)

	if t.WebsiteName == "" {
		e["WebsiteName"] = "is required"
	}

	if t.WebsiteURL == "" {
		e["WebsiteURL"] = "is required"
	}

	if t.Timeout != 0 && (t.Timeout < 6 || t.Timeout > 99) {
		e["Timeout"] = "must be 0 or between 6 and 99"
	}

	if t.Confirmation < 0 || t.Confirmation > 9 {
		e["Confirmation"] = "must be between 0 and 9"
	}

	if t.CheckRate < 0 || t.CheckRate > 23999 {
		e["CheckRate"] = "must be between 0 and 23999"
	}

	if t.Public < 0 || t.Public > 1 {
		e["Public"] = "must be 0 or 1"
	}

	if t.Virus < 0 || t.Virus > 1 {
		e["Virus"] = "must be 0 or 1"
	}

	if t.TestType != "HTTP" && t.TestType != "TCP" && t.TestType != "PING" {
		e["TestType"] = "must be HTTP, TCP, or PING"
	}

	if t.RealBrowser < 0 || t.RealBrowser > 1 {
		e["RealBrowser"] = "must be 0 or 1"
	}

	if t.TriggerRate < 0 || t.TriggerRate > 59 {
		e["TriggerRate"] = "must be between 0 and 59"
	}

	if len(e) > 0 {
		return e
	}

	return nil
}

// ToURLValues returns url.Values of all fields required to create/update a Test.
func (t Test) ToURLValues() url.Values {
	values := make(url.Values)
	st := reflect.TypeOf(t)
	sv := reflect.ValueOf(t)
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		tag := sf.Tag.Get(queryStringTag)
		ft := sf.Type
		if ft.Name() == "" && ft.Kind() == reflect.Ptr {
			// Follow pointer.
			ft = ft.Elem()
		}

		v := sv.Field(i)
		options := sf.Tag.Get("querystringoptions")
		omit := options == "omitempty" && isEmptyValue(v)

		if tag != "" && !omit {
			values.Set(tag, valueToQueryStringValue(v))
		}
	}

	return values
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}

	return false
}

func valueToQueryStringValue(v reflect.Value) string {
	if v.Type().Name() == "bool" {
		if v.Bool() {
			return "1"
		}

		return "0"
	}

	if v.Type().Kind() == reflect.Slice {
		if ss, ok := v.Interface().([]string); ok {
			return strings.Join(ss, ",")
		}
	}

	return fmt.Sprint(v)
}

// Tests is a client that implements the `Tests` API.
type Tests interface {
	All() ([]*Test, error)
	Detail(int) (*Test, error)
	Update(*Test) (*Test, error)
	Delete(TestID int) error
}

type tests struct {
	client apiClient
}

func newTests(c apiClient) Tests {
	return &tests{
		client: c,
	}
}

func (tt *tests) All() ([]*Test, error) {
	resp, err := tt.client.get("/Tests", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tests []*Test
	err = json.NewDecoder(resp.Body).Decode(&tests)

	return tests, err
}

func (tt *tests) Update(t *Test) (*Test, error) {
	resp, err := tt.client.put("/Tests/Update", t.ToURLValues())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ur updateResponse
	err = json.NewDecoder(resp.Body).Decode(&ur)
	if err != nil {
		return nil, err
	}

	if !ur.Success {
		return nil, &updateError{Issues: ur.Issues}
	}

	t2 := *t
	t2.TestID = ur.InsertID

	return &t2, err
}

func (tt *tests) Delete(testID int) error {
	resp, err := tt.client.delete("/Tests/Details", url.Values{"TestID": {fmt.Sprint(testID)}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var dr deleteResponse
	err = json.NewDecoder(resp.Body).Decode(&dr)
	if err != nil {
		return err
	}

	if !dr.Success {
		return &deleteError{Message: dr.Error}
	}

	return nil
}

func (tt *tests) Detail(testID int) (*Test, error) {
	resp, err := tt.client.get("/Tests/Details", url.Values{"TestID": {fmt.Sprint(testID)}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dr *detailResponse
	err = json.NewDecoder(resp.Body).Decode(&dr)
	if err != nil {
		return nil, err
	}

	return dr.test(), nil
}
