package dnsimple

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/rubyist/go-dnsimple"
)

func resource_dnsimple_record_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	var err error

	newRecord := dnsimple.Record{
		Name:       rs.Attributes["name"],
		Content:    rs.Attributes["value"],
		RecordType: rs.Attributes["type"],
	}

	if attr, ok := rs.Attributes["ttl"]; ok {
		newRecord.TTL, err = strconv.Atoi(attr)
		if err != nil {
			return nil, err
		}
	}

	log.Printf("[DEBUG] record create configuration: %#v", newRecord)

	rec, err := client.CreateRecord(rs.Attributes["domain"], newRecord)

	if err != nil {
		return nil, fmt.Errorf("Failed to create record: %s", err)
	}

	rs.ID = strconv.Itoa(rec.Id)

	log.Printf("[INFO] record ID: %s", rs.ID)

	return resource_dnsimple_record_update_state(rs, &rec)
}

func resource_dnsimple_record_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	panic("Cannot update record")

	return nil, nil
}

func resource_dnsimple_record_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting record: %s", s.ID)

	rec, err := resource_dnsimple_record_retrieve(s.Attributes["domain"], s.ID, client)
	if err != nil {
		return err
	}

	err = rec.Delete(client)
	if err != nil {
		return fmt.Errorf("Error deleting record: %s", err)
	}

	return nil
}

func resource_dnsimple_record_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	rec, err := resource_dnsimple_record_retrieve(s.Attributes["app"], s.ID, client)
	if err != nil {
		return nil, err
	}

	return resource_dnsimple_record_update_state(s, rec)
}

func resource_dnsimple_record_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"domain": diff.AttrTypeCreate,
			"name":   diff.AttrTypeCreate,
			"value":  diff.AttrTypeUpdate,
			"ttl":    diff.AttrTypeCreate,
			"type":   diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"priority",
			"domain_id",
		},
	}

	return b.Diff(s, c)
}

func resource_dnsimple_record_update_state(
	s *terraform.ResourceState,
	rec *dnsimple.Record) (*terraform.ResourceState, error) {

	s.Attributes["name"] = rec.Name
	s.Attributes["value"] = rec.Content
	s.Attributes["type"] = rec.RecordType
	s.Attributes["ttl"] = strconv.Itoa(rec.TTL)
	s.Attributes["priority"] = strconv.Itoa(rec.Priority)
	s.Attributes["domain_id"] = strconv.Itoa(rec.DomainId)

	return s, nil
}

func resource_dnsimple_record_retrieve(domain string, id string, client *dnsimple.DNSimpleClient) (*dnsimple.Record, error) {
	intId, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	record, err := client.RetrieveRecord(domain, intId)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving record: %s", err)
	}

	return &record, nil
}

func resource_dnsimple_record_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"domain",
			"name",
			"value",
			"type",
		},
		Optional: []string{
			"ttl",
		},
	}
}
