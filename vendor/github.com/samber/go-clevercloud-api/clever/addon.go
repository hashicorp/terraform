package clever

import (
	"fmt"
	"strings"
)


//
// TYPES
//
type AddonInput struct {
	Name       string `json:"name"`
	Plan       string `json:"plan"`
	ProviderId string `json:"providerId"`
	Region     string `json:"region"`
}

type AddonOutput struct {
	Id       string              `json:"id"`
	RealId   string              `json:"realId"`
	Name     string              `json:"name"`
	Region   string              `json:"region"`
	Plan     AddonOutputPlan     `json:"plan"`
	Provider AddonOutputProvider `json:"provider"`
	Env      []string
}
type AddonOutputPlan struct {
	Id    string  `json:"id"`
	Name  string  `json:"name"`
	Price float32 `json:"price"`
	Slug  string  `json:"slug"`
}
type AddonOutputProvider struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}


//
// INIT
//
type AvailableAddon struct {
	Id      string               `json:"id"`
	Name    string               `json:"Name"`
	Plans   []AvailableAddonPlan `json:"plans"`
	Regions []string             `json:"regions"`
}
type AvailableAddonPlan struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

var AVAILABLE_ADDONS []*AvailableAddon
var MATCHING_ADDON_PLAN map[string]map[string]string

func (c *Client) loadAddons() error {
	if err := c.jsonRequest("GET", "/products/addonproviders", nil, &AVAILABLE_ADDONS); err != nil {
		return err
	}

	MATCHING_ADDON_PLAN = map[string]map[string]string{}
	for _, addon := range AVAILABLE_ADDONS {
		MATCHING_ADDON_PLAN[addon.Id] = map[string]string{}
		for _, plan := range addon.Plans {
			MATCHING_ADDON_PLAN[addon.Id][plan.Slug] = plan.Id
		}
	}

	return nil
}


//
// ADDONS
//
func (c *Client) GetAddonById(addon_id string) (*AddonOutput, error) {
	var addonOutput AddonOutput
	err := c.get("/organisations/"+c.config.OrgId+"/addons/"+addon_id, &addonOutput)
	if err != nil {
		return nil, err
	}

	return &addonOutput, nil
}

func (c *Client) CreateAddon(addonInput *AddonInput) (*AddonOutput, error) {
	// User set Plan name, but api expect Plan id
	plans, ok := MATCHING_ADDON_PLAN[addonInput.ProviderId]
	if ok == false {
		return nil, fmt.Errorf("Unknown addon: " + addonInput.ProviderId)
	}
	plan, ok := plans[strings.ToLower(addonInput.Plan)]
	if ok == false {
		return nil, fmt.Errorf("Unknown plan: " + addonInput.Plan)
	}
	addonInput.Plan = plan

	var addonOutput AddonOutput
	err := c.post("/organisations/"+c.config.OrgId+"/addons", addonInput, &addonOutput)
	if err != nil {
		return nil, err
	}

	return &addonOutput, nil
}

// Not expected to happen for addons
//func (c *Client) UpdateAddon(addon_id string, addonInput *AddonInput) (*AddonOutput, error) {
//	var addonOutput AddonOutput
//
//	err := c.put("/organisations/"+c.config.OrgId+"/addons/"+addon_id, addonInput, &addonOutput)
//	if err != nil {
//		return nil, err
//	}
//
//	return &addonOutput, nil
//}

func (c *Client) DeleteAddon(addon_id string) error {
	return c.delete("/organisations/" + c.config.OrgId + "/addons/" + addon_id)
}



//
// ADDON ENV VARS
//
func (c *Client) GetAddonEnvById(addon_id string) (map[string]string, error) {
	var envOutput []Env
	err := c.get("/organisations/"+c.config.OrgId+"/addons/"+addon_id+"/env", &envOutput)
	if err != nil {
		return nil, err
	}

	returnedKv := map[string]string{}
	for _, output := range envOutput {
		returnedKv[output.Key] = output.Value
	}

	return returnedKv, nil
}
