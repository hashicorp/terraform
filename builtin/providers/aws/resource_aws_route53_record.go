package aws

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/route53"
)

func resource_aws_r53_record_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"zone_id",
			"name",
			"type",
			"ttl",
			"records.*",
		},
	}
}

func resource_aws_r53_record_create(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	conn := p.route53

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	// Get the record
	rec, err := resource_aws_r53_build_record_set(rs)
	if err != nil {
		return rs, err
	}

	// Create the new records. We abuse StateChangeConf for this to
	// retry for us since Route53 sometimes returns errors about another
	// operation happening at the same time.
	req := &route53.ChangeResourceRecordSetsRequest{
		Comment: "Managed by Terraform",
		Changes: []route53.Change{
			route53.Change{
				Action: "UPSERT",
				Record: *rec,
			},
		},
	}
	zone := rs.Attributes["zone_id"]
	log.Printf("[DEBUG] Creating resource records for zone: %s, name: %s",
		zone, rs.Attributes["name"])
	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     "accepted",
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.ChangeResourceRecordSets(zone, req)
			if err != nil {
				if strings.Contains(err.Error(), "PriorRequestNotComplete") {
					// There is some pending operation, so just retry
					// in a bit.
					return nil, "rejected", nil
				}

				return nil, "failure", err
			}

			return resp.ChangeInfo, "accepted", nil
		},
	}
	respRaw, err := wait.WaitForState()
	if err != nil {
		return rs, err
	}
	changeInfo := respRaw.(route53.ChangeInfo)

	// Generate an ID
	rs.ID = fmt.Sprintf("%s_%s_%s", zone, rs.Attributes["name"], rs.Attributes["type"])

	// Wait until we are done
	wait = resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{"PENDING"},
		Target:     "INSYNC",
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			return resource_aws_r53_wait(conn, changeInfo.ID)
		},
	}
	_, err = wait.WaitForState()
	if err != nil {
		return rs, err
	}
	return rs, nil
}

func resource_aws_r53_build_record_set(s *terraform.InstanceState) (*route53.ResourceRecordSet, error) {
	// Parse the TTL
	ttl, err := strconv.ParseInt(s.Attributes["ttl"], 10, 32)
	if err != nil {
		return nil, err
	}

	// Expand the records
	recRaw := flatmap.Expand(s.Attributes, "records")
	var records []string
	for _, raw := range recRaw.([]interface{}) {
		records = append(records, raw.(string))
	}

	rec := &route53.ResourceRecordSet{
		Name:    s.Attributes["name"],
		Type:    s.Attributes["type"],
		TTL:     int(ttl),
		Records: records,
	}
	return rec, nil
}

func resource_aws_r53_record_destroy(
	s *terraform.InstanceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	conn := p.route53

	// Get the record
	rec, err := resource_aws_r53_build_record_set(s)
	if err != nil {
		return err
	}

	// Create the new records
	req := &route53.ChangeResourceRecordSetsRequest{
		Comment: "Deleted by Terraform",
		Changes: []route53.Change{
			route53.Change{
				Action: "DELETE",
				Record: *rec,
			},
		},
	}
	zone := s.Attributes["zone_id"]
	log.Printf("[DEBUG] Deleting resource records for zone: %s, name: %s",
		zone, s.Attributes["name"])
	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     "accepted",
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			_, err := conn.ChangeResourceRecordSets(zone, req)
			if err != nil {
				if strings.Contains(err.Error(), "PriorRequestNotComplete") {
					// There is some pending operation, so just retry
					// in a bit.
					return 42, "rejected", nil
				}

				if strings.Contains(err.Error(), "InvalidChangeBatch") {
					// This means that the record is already gone.
					return 42, "accepted", nil
				}

				return 42, "failure", err
			}

			return 42, "accepted", nil
		},
	}
	if _, err := wait.WaitForState(); err != nil {
		return err
	}

	return nil
}

func resource_aws_r53_record_refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	conn := p.route53

	zone := s.Attributes["zone_id"]
	lopts := &route53.ListOpts{
		Name: s.Attributes["name"],
		Type: s.Attributes["type"],
	}
	resp, err := conn.ListResourceRecordSets(zone, lopts)
	if err != nil {
		return s, err
	}

	// Scan for a matching record
	found := false
	for _, record := range resp.Records {
		if route53.FQDN(record.Name) != route53.FQDN(lopts.Name) {
			continue
		}
		if strings.ToUpper(record.Type) != strings.ToUpper(lopts.Type) {
			continue
		}

		found = true
		resource_aws_r53_record_update_state(s, &record)
		break
	}
	if !found {
		s.ID = ""
	}
	return s, nil
}

func resource_aws_r53_record_update_state(
	s *terraform.InstanceState,
	rec *route53.ResourceRecordSet) {

	flatRec := flatmap.Flatten(map[string]interface{}{
		"records": rec.Records,
	})
	for k, v := range flatRec {
		s.Attributes[k] = v
	}

	s.Attributes["ttl"] = strconv.FormatInt(int64(rec.TTL), 10)
}

func resource_aws_r53_record_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"zone_id": diff.AttrTypeCreate,
			"name":    diff.AttrTypeCreate,
			"type":    diff.AttrTypeCreate,
			"ttl":     diff.AttrTypeUpdate,
			"records": diff.AttrTypeUpdate,
		},
	}
	return b.Diff(s, c)
}
