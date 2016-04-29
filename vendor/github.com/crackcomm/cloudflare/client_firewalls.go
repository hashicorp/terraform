package cloudflare

import (
	"bytes"
	"encoding/json"

	"golang.org/x/net/context"
)

// Firewalls - Cloudflare Fireall Zones API Client.
type Firewalls struct {
	opts *Options
}

// Create - Creates a firewall rule for zone.
func (firewalls *Firewalls) Create(ctx context.Context, id string, firewall *Firewall) (fw *Firewall, err error) {
	buffer := new(bytes.Buffer)
	err = json.NewEncoder(buffer).Encode(firewall)
	if err != nil {
		return
	}
	response, err := httpDo(ctx, firewalls.opts, "POST", apiURL("/zones/%s/firewall/access_rules/rules", id), buffer)
	if err != nil {
		return
	}
	defer response.Body.Close()
	result, err := readResponse(response.Body)
	if err != nil {
		return
	}
	fw = new(Firewall)
	err = json.Unmarshal(result.Result, &fw)
	return
}

// List - Lists all firewall rules for zone.
func (firewalls *Firewalls) List(ctx context.Context, zone string) ([]*Firewall, error) {
	return firewalls.listPages(ctx, zone, 1)
}

// Delete - Deletes firewall by id.
func (firewalls *Firewalls) Delete(ctx context.Context, zone, id string) (err error) {
	response, err := httpDo(ctx, firewalls.opts, "DELETE", apiURL("/zones/%s/firewall/access_rules/rules/%s", zone, id), nil)
	if err != nil {
		return
	}
	defer response.Body.Close()
	_, err = readResponse(response.Body)
	return
}

// listPages - Gets all pages starting from `page`.
func (firewalls *Firewalls) listPages(ctx context.Context, zone string, page int) (list []*Firewall, err error) {
	response, err := httpDo(ctx, firewalls.opts, "GET", apiURL("/zones/%s/firewall/access_rules/rules?page=%d&per_page=50", zone, page), nil)
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
	next, err := firewalls.listPages(ctx, zone, page+1)
	if err != nil {
		return
	}
	return append(list, next...), nil
}
