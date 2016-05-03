package cloudflare

import (
	"bytes"
	"encoding/json"

	"golang.org/x/net/context"
)

// Zones - Cloudflare Zones API Client.
type Zones struct {
	opts *Options
}

// Create - Creates a zone.
func (zones *Zones) Create(ctx context.Context, domain string) (zone *Zone, err error) {
	buffer := new(bytes.Buffer)
	err = json.NewEncoder(buffer).Encode(struct {
		Name string `json:"name"`
	}{
		Name: domain,
	})
	if err != nil {
		return
	}
	response, err := httpDo(ctx, zones.opts, "POST", apiURL("/zones"), buffer)
	if err != nil {
		return
	}
	defer response.Body.Close()
	result, err := readResponse(response.Body)
	if err != nil {
		return
	}
	zone = new(Zone)
	err = json.Unmarshal(result.Result, &zone)
	return
}

// List - Lists all zones.
func (zones *Zones) List(ctx context.Context) ([]*Zone, error) {
	return zones.listPages(ctx, 1)
}

// Details - Requests Zone details by ID.
func (zones *Zones) Details(ctx context.Context, id string) (zone *Zone, err error) {
	response, err := httpDo(ctx, zones.opts, "GET", apiURL("/zones/%s", id), nil)
	if err != nil {
		return
	}
	defer response.Body.Close()
	result, err := readResponse(response.Body)
	if err != nil {
		return
	}
	zone = new(Zone)
	err = json.Unmarshal(result.Result, &zone)
	return
}

// Patch - Patches a zone. It has a limited possibilities.
func (zones *Zones) Patch(ctx context.Context, id string, patch *ZonePatch) (err error) {
	buffer := new(bytes.Buffer)
	err = json.NewEncoder(buffer).Encode(patch)
	if err != nil {
		return
	}
	response, err := httpDo(ctx, zones.opts, "POST", apiURL("/zones/%s", id), buffer)
	if err != nil {
		return
	}
	defer response.Body.Close()
	_, err = readResponse(response.Body)
	return
}

// Delete - Deletes zone by id.
func (zones *Zones) Delete(ctx context.Context, id string) (err error) {
	response, err := httpDo(ctx, zones.opts, "DELETE", apiURL("/zones/%s", id), nil)
	if err != nil {
		return
	}
	defer response.Body.Close()
	_, err = readResponse(response.Body)
	return
}

// listPages - Gets all pages starting from `page`.
func (zones *Zones) listPages(ctx context.Context, page int) (list []*Zone, err error) {
	response, err := httpDo(ctx, zones.opts, "GET", apiURL("/zones?page=%d&per_page=50", page), nil)
	if err != nil {
		return
	}
	defer response.Body.Close()
	result, err := readResponse(response.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(result.Result, &list)
	if err != nil {
		return
	}
	if result.ResultInfo == nil || page >= result.ResultInfo.TotalPages {
		return
	}
	next, err := zones.listPages(ctx, page+1)
	if err != nil {
		return
	}
	return append(list, next...), nil
}
