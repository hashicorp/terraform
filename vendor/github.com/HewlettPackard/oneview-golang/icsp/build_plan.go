package icsp

import (
	"encoding/json"
	"strings"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

// URLEndPoint(s) export this constant
const (
	URLEndPointBuildPlan = "/rest/os-deployment-build-plans"
)

// BuildPlanItem
type BuildPlanItem struct {
	CfgFileDownload  bool          `json:"cfgFileDownload,omitempty"`  // cfgFileDownload - Boolean that indicates whether the current step is used for downloading configuration file or uploading it
	CfgFileOverwrite bool          `json:"cfgFileOverwrite,omitempty"` // cfgFileOverwrite - Flag that indicates whether or not to overwrite the file on the target server if the step type is 'Config File' and it is a download
	CodeType         string        `json:"codeType,omitempty"`         // codeType -  Supported types for scripts: OGFS, Python, Unix, Windows .BAT, Windows VBScript. Supported types for packages: Install ZIP. Supported type for configuration file: Config File
	ID               string        `json:"id:omitempty"`               // id - System-assigned id of the Step
	Name             string        `json:"name,omitempty"`             // name  - name of step
	Parameters       string        `json:"parameters,omitempty"`       // parameters -  Additional parameters that affect the operations of the Step
	Type             string        `json:"type,omitempty"`             // type - TYpe of the step
	URI              utils.Nstring `json:"uri,omitempty"`              // uri - The canonical URI of the Step
}

// BuildPlanHistory
type BuildPlanHistory struct {
	Summary string `json:"summary,omitempty"` // summary - A time ordered array of change log entries. An empty array is returned if no entries were found
	User    string `json:"user,omitempty"`    // user - User to whom log entries belong
	Time    string `json:"time,omitempty"`    // time - Time window for log entries. Default is 90 days
}

// BuildPlanCustAttrs
type BuildPlanCustAttrs struct {
	Attribute string `json:"attribute,omitempty"` // Attribute - Name of the name/value custom attribute pair associated with this OS Build Plan
	Value     string `json:"value,omitempty"`     // Value - Value of the name/value custom attribute pair associated with this OS Build Plan
}

// OSDBuildPlan struct
type OSDBuildPlan struct {
	Arch               string               `json:"arch,omitempty"`
	BuildPlanHistory   []BuildPlanHistory   `json:"buildPlanHistory,omitempty"` // buildPlanHistory array
	BuildPlanStepType  string               `json:"buildPlanStepType,omitempty"`
	IsCustomerContent  bool                 `json:"isCustomerContent,omitempty"`
	OS                 string               `json:"os,omitempty"`
	BuildPlanCustAttrs []BuildPlanCustAttrs `json:"buildPlanCustAttrs,omitempty"`
	BuildPlanItems     []BuildPlanItem      `json:"buildPlanItems,omitempty"`
	ModifiedBy         string               `json:"modifiedBy,omitempty"`
	CreatedBy          string               `json:"createdBy,omitempty"`
	LifeCycle          string               `json:"lifeCycle,omitstring"`
	Description        string               `json:"description,omitempty"`
	Status             string               `json:"status,omitempty"`
	Name               string               `json:"name,omitempty"`
	ETAG               string               `json:"eTag,omitempty"` // eTag Entity tag/version ID of the resource
	Modified           string               `json:"modified,omitempty"`
	Created            string               `json:"created,omitempty"`
	URI                utils.Nstring        `json:"uri,omitempty"` // uri The canonical URI of the buildplan
}

// OSBuildPlan
type OSBuildPlan struct {
	Category    string         `json:"category,omitempty"`    //Category - Resource category used for authorizations and resource type groupings
	Count       int            `json:"count,omiitempty"`      // Count - The actual number of resources returned in the specified page
	Created     string         `json:"created,omitempty"`     // Created - Date and time when the resource was created
	ETag        string         `json:"eTag,omitempty"`        // ETag - Entity tag/version ID of the resource, the same value that is returned in the ETag header on a GET of the resource
	Members     []OSDBuildPlan `json:"members,omitempty"`     // Members - array of BuildPlans
	Modified    string         `json:"modified,omitempty"`    // Modified -
	NextPageURI utils.Nstring  `json:"nextPageUri,omitempty"` // NextPageURI - URI pointing to the page of resources following the list of resources contained in the specified collection
	PrevPageURI utils.Nstring  `json:"prevPageURI,omitempty"` // PrevPageURI - URI pointing to the page of resources preceding the list of resources contained in the specified collection
	Start       int            `json:"start,omitempty"`       // Start - The row or record number of the first resource returned in the specified page
	Total       int            `json:"total,omitempty"`       // Total -  The total number of resources that would be returned from the query (including any filters), without pagination or enforced resource limits
	URI         utils.Nstring  `json:"uri,omitempty"`         //  URI -
	Type        string         `json:"type,omitempty"`        // Type - type of paging
}

// GetAllBuildPlans - returns all OS build plans
// returns BuildPlan
// note: this call is crap slow...API: should include filters/query params
func (c *ICSPClient) GetAllBuildPlans() (OSBuildPlan, error) {
	var (
		uri   = URLEndPointBuildPlan
		plans OSBuildPlan
	)

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return plans, err
	}

	// log.Debugf("GetAllBuildPlans %s", data)
	log.Debugf("GetAllBuildPlans completed")
	if err := json.Unmarshal([]byte(data), &plans); err != nil {
		return plans, err
	}
	return plans, err
}

// GetBuildPlanByName -  returns a build plan
func (c *ICSPClient) GetBuildPlanByName(planName string) (OSDBuildPlan, error) {

	var bldplan OSDBuildPlan
	plans, err := c.GetAllBuildPlans()
	if err != nil {
		return bldplan, err
	}
	log.Debugf("GetBuildPlanByName: server count: %d", plans.Count)
	// grab the target
	for _, plan := range plans.Members {
		if strings.EqualFold(plan.Name, planName) {
			log.Debugf("plan name: %v", plan.Name)
			bldplan = plan
			break
		}
	}
	return bldplan, nil
}

func (c *ICSPClient) GetBuildPlanByUri(uri utils.Nstring) (OSDBuildPlan, error) {

	var bldplan OSDBuildPlan
	// grab the target
	data, err := c.RestAPICall(rest.GET, uri.String(), nil)
	if err != nil {
		return bldplan, err
	}
	log.Debugf("GetBuildPlan %s", data)
	if err := json.Unmarshal([]byte(data), &bldplan); err != nil {
		return bldplan, err
	}

	return bldplan, nil
}
