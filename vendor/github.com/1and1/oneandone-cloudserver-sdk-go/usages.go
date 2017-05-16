package oneandone

import (
	"net/http"
	"time"
)

type Usages struct {
	Images         []usage `json:"IMAGES,omitempty"`
	LoadBalancers  []usage `json:"LOAD BALANCERS,omitempty"`
	PublicIPs      []usage `json:"PUBLIC IP,omitempty"`
	Servers        []usage `json:"SERVERS,omitempty"`
	SharedStorages []usage `json:"SHARED STORAGE,omitempty"`
	ApiPtr
}

type usage struct {
	Identity
	Site     int            `json:"site"`
	Services []usageService `json:"services,omitempty"`
}

type usageService struct {
	AverageAmmount string         `json:"avg_amount,omitempty"`
	Unit           string         `json:"unit,omitempty"`
	Usage          int            `json:"usage"`
	Details        []usageDetails `json:"detail,omitempty"`
	typeField
}

type usageDetails struct {
	AverageAmmount string `json:"avg_amount,omitempty"`
	StartDate      string `json:"start_date,omitempty"`
	EndDate        string `json:"end_date,omitempty"`
	Unit           string `json:"unit,omitempty"`
	Usage          int    `json:"usage,omitempty"`
}

// GET /usages
func (api *API) ListUsages(period string, sd *time.Time, ed *time.Time, args ...interface{}) (*Usages, error) {
	result := new(Usages)
	url, err := processQueryParamsExt(createUrl(api, usagePathSegment), period, sd, ed, args...)
	if err != nil {
		return nil, err
	}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}
