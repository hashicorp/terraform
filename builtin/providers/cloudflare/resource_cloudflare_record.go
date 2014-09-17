package cloudflare

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/cloudflare"
)

func resource_cloudflare_record_create(
	s *terraform.InstanceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	var err error

	newRecord := cloudflare.CreateRecord{
		Name:     rs.Attributes["name"],
		Priority: rs.Attributes["priority"],
		Type:     rs.Attributes["type"],
		Content:  rs.Attributes["value"],
		Ttl:      rs.Attributes["ttl"],
	}

	log.Printf("[DEBUG] record create configuration: %#v", newRecord)

	rec, err := client.CreateRecord(rs.Attributes["domain"], &newRecord)

	if err != nil {
		return nil, fmt.Errorf("Failed to create record: %s", err)
	}

	rs.ID = rec.Id
	log.Printf("[INFO] record ID: %s", rs.ID)

	record, err := resource_cloudflare_record_retrieve(rs.Attributes["domain"], rs.ID, client)
	if err != nil {
		return nil, fmt.Errorf("Couldn't find record: %s", err)
	}

	return resource_cloudflare_record_update_state(rs, record)
}

func resource_cloudflare_record_update(
	s *terraform.InstanceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client
	rs := s.MergeDiff(d)

	// Cloudflare requires we send all values
	// for an update request, so we just
	// merge out diff and send the current
	// state of affairs to them
	updateRecord := cloudflare.UpdateRecord{
		Name:     rs.Attributes["name"],
		Content:  rs.Attributes["value"],
		Type:     rs.Attributes["type"],
		Ttl:      rs.Attributes["ttl"],
		Priority: rs.Attributes["priority"],
	}

	log.Printf("[DEBUG] record update configuration: %#v", updateRecord)

	err := client.UpdateRecord(rs.Attributes["domain"], rs.ID, &updateRecord)
	if err != nil {
		return rs, fmt.Errorf("Failed to update record: %s", err)
	}

	record, err := resource_cloudflare_record_retrieve(rs.Attributes["domain"], rs.ID, client)
	if err != nil {
		return rs, fmt.Errorf("Couldn't find record: %s", err)
	}

	return resource_cloudflare_record_update_state(rs, record)
}

func resource_cloudflare_record_destroy(
	s *terraform.InstanceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting record: %s, %s", s.Attributes["domain"], s.ID)

	err := client.DestroyRecord(s.Attributes["domain"], s.ID)

	if err != nil {
		return fmt.Errorf("Error deleting record: %s", err)
	}

	return nil
}

func resource_cloudflare_record_refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	rec, err := resource_cloudflare_record_retrieve(s.Attributes["domain"], s.ID, client)
	if err != nil {
		return nil, err
	}

	return resource_cloudflare_record_update_state(s, rec)
}

func resource_cloudflare_record_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"domain":   diff.AttrTypeCreate,
			"name":     diff.AttrTypeUpdate,
			"value":    diff.AttrTypeUpdate,
			"ttl":      diff.AttrTypeUpdate,
			"type":     diff.AttrTypeUpdate,
			"priority": diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"priority",
			"ttl",
			"hostname",
		},

		ComputedAttrsUpdate: []string{},
	}

	return b.Diff(s, c)
}

func resource_cloudflare_record_update_state(
	s *terraform.InstanceState,
	rec *cloudflare.Record) (*terraform.InstanceState, error) {

	s.Attributes["name"] = rec.Name
	s.Attributes["value"] = rec.Value
	s.Attributes["type"] = rec.Type
	s.Attributes["ttl"] = rec.Ttl
	s.Attributes["priority"] = rec.Priority
	s.Attributes["hostname"] = rec.FullName

	return s, nil
}

func resource_cloudflare_record_retrieve(domain string, id string, client *cloudflare.Client) (*cloudflare.Record, error) {
	record, err := client.RetrieveRecord(domain, id)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func resource_cloudflare_record_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"domain",
			"name",
			"value",
			"type",
		},
		Optional: []string{
			"ttl",
			"priority",
		},
	}
}
