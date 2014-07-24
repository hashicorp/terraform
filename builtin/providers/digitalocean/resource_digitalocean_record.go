package digitalocean

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/digitalocean"
)

func resource_digitalocean_record_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	var err error

	newRecord := digitalocean.CreateRecord{
		Type:     rs.Attributes["type"],
		Name:     rs.Attributes["name"],
		Data:     rs.Attributes["value"],
		Priority: rs.Attributes["priority"],
		Port:     rs.Attributes["port"],
		Weight:   rs.Attributes["weight"],
	}

	log.Printf("[DEBUG] record create configuration: %#v", newRecord)

	recId, err := client.CreateRecord(rs.Attributes["domain"], &newRecord)

	if err != nil {
		return nil, fmt.Errorf("Failed to create record: %s", err)
	}

	rs.ID = recId
	log.Printf("[INFO] Record ID: %s", rs.ID)

	record, err := resource_digitalocean_record_retrieve(rs.Attributes["domain"], rs.ID, client)
	if err != nil {
		return nil, fmt.Errorf("Couldn't find record: %s", err)
	}

	return resource_digitalocean_record_update_state(rs, record)
}

func resource_digitalocean_record_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client
	rs := s.MergeDiff(d)

	updateRecord := digitalocean.UpdateRecord{}

	if attr, ok := d.Attributes["name"]; ok {
		updateRecord.Name = attr.New
	}

	log.Printf("[DEBUG] record update configuration: %#v", updateRecord)

	err := client.UpdateRecord(rs.Attributes["domain"], rs.ID, &updateRecord)
	if err != nil {
		return rs, fmt.Errorf("Failed to update record: %s", err)
	}

	record, err := resource_digitalocean_record_retrieve(rs.Attributes["domain"], rs.ID, client)
	if err != nil {
		return rs, fmt.Errorf("Couldn't find record: %s", err)
	}

	return resource_digitalocean_record_update_state(rs, record)
}

func resource_digitalocean_record_destroy(
	s *terraform.ResourceState,
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

func resource_digitalocean_record_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	rec, err := resource_digitalocean_record_retrieve(s.Attributes["domain"], s.ID, client)
	if err != nil {
		return nil, err
	}

	return resource_digitalocean_record_update_state(s, rec)
}

func resource_digitalocean_record_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"domain":   diff.AttrTypeCreate,
			"name":     diff.AttrTypeUpdate,
			"type":     diff.AttrTypeCreate,
			"value":    diff.AttrTypeCreate,
			"priority": diff.AttrTypeCreate,
			"port":     diff.AttrTypeCreate,
			"weight":   diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"value",
			"priority",
			"weight",
			"port",
		},
	}

	return b.Diff(s, c)
}

func resource_digitalocean_record_update_state(
	s *terraform.ResourceState,
	rec *digitalocean.Record) (*terraform.ResourceState, error) {

	s.Attributes["name"] = rec.Name
	s.Attributes["type"] = rec.Type
	s.Attributes["value"] = rec.Data
	s.Attributes["weight"] = rec.StringWeight()
	s.Attributes["priority"] = rec.StringPriority()
	s.Attributes["port"] = rec.StringPort()

	// We belong to a Domain
	s.Dependencies = []terraform.ResourceDependency{
		terraform.ResourceDependency{ID: s.Attributes["domain"]},
	}

	return s, nil
}

func resource_digitalocean_record_retrieve(domain string, id string, client *digitalocean.Client) (*digitalocean.Record, error) {
	record, err := client.RetrieveRecord(domain, id)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func resource_digitalocean_record_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"type",
			"domain",
		},
		Optional: []string{
			"value",
			"name",
			"weight",
			"port",
			"priority",
		},
	}
}
