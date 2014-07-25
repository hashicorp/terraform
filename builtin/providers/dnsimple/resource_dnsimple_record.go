package dnsimple

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/dnsimple"
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

	newRecord := dnsimple.ChangeRecord{
		Name:  rs.Attributes["name"],
		Value: rs.Attributes["value"],
		Type:  rs.Attributes["type"],
	}

	if attr, ok := rs.Attributes["ttl"]; ok {
		newRecord.Ttl = attr
	}

	log.Printf("[DEBUG] record create configuration: %#v", newRecord)

	recId, err := client.CreateRecord(rs.Attributes["domain"], &newRecord)

	if err != nil {
		return nil, fmt.Errorf("Failed to create record: %s", err)
	}

	rs.ID = recId
	log.Printf("[INFO] record ID: %s", rs.ID)

	record, err := resource_dnsimple_record_retrieve(rs.Attributes["domain"], rs.ID, client)
	if err != nil {
		return nil, fmt.Errorf("Couldn't find record: %s", err)
	}

	return resource_dnsimple_record_update_state(rs, record)
}

func resource_dnsimple_record_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client
	rs := s.MergeDiff(d)

	updateRecord := dnsimple.ChangeRecord{}

	if attr, ok := d.Attributes["name"]; ok {
		updateRecord.Name = attr.New
	}

	if attr, ok := d.Attributes["value"]; ok {
		updateRecord.Value = attr.New
	}

	if attr, ok := d.Attributes["type"]; ok {
		updateRecord.Type = attr.New
	}

	if attr, ok := d.Attributes["ttl"]; ok {
		updateRecord.Ttl = attr.New
	}

	log.Printf("[DEBUG] record update configuration: %#v", updateRecord)

	_, err := client.UpdateRecord(rs.Attributes["domain"], rs.ID, &updateRecord)
	if err != nil {
		return rs, fmt.Errorf("Failed to update record: %s", err)
	}

	record, err := resource_dnsimple_record_retrieve(rs.Attributes["domain"], rs.ID, client)
	if err != nil {
		return rs, fmt.Errorf("Couldn't find record: %s", err)
	}

	return resource_dnsimple_record_update_state(rs, record)
}

func resource_dnsimple_record_destroy(
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

func resource_dnsimple_record_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	rec, err := resource_dnsimple_record_retrieve(s.Attributes["domain"], s.ID, client)
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
			"name":   diff.AttrTypeUpdate,
			"value":  diff.AttrTypeUpdate,
			"ttl":    diff.AttrTypeUpdate,
			"type":   diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"priority",
			"domain_id",
			"ttl",
		},

		ComputedAttrsUpdate: []string{
			"hostname",
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
	s.Attributes["ttl"] = rec.StringTtl()
	s.Attributes["priority"] = rec.StringPrio()
	s.Attributes["domain_id"] = rec.StringDomainId()

	if rec.Name == "" {
		s.Attributes["hostname"] = s.Attributes["domain"]
	} else {
		s.Attributes["hostname"] = fmt.Sprintf("%s.%s", rec.Name, s.Attributes["domain"])
	}

	return s, nil
}

func resource_dnsimple_record_retrieve(domain string, id string, client *dnsimple.Client) (*dnsimple.Record, error) {
	record, err := client.RetrieveRecord(domain, id)
	if err != nil {
		return nil, err
	}

	return record, nil
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
