package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/s3"
)

func resource_aws_s3_bucket_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"bucket",
		},
		Optional: []string{
			"acl",
		},
	}
}

func resource_aws_s3_bucket_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	s3conn := p.s3conn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	// Get the bucket and optional acl
	bucket := rs.Attributes["bucket"]
	acl := "private"
	if other, ok := rs.Attributes["acl"]; ok {
		acl = other
	}

	log.Printf("[DEBUG] S3 bucket create: %s, ACL: %s", bucket, acl)
	s3Bucket := s3conn.Bucket(bucket)
	err := s3Bucket.PutBucket(s3.ACL(acl))
	if err != nil {
		return nil, fmt.Errorf("Error creating S3 bucket: %s", err)
	}

	// Assign the bucket name as the resource ID
	rs.ID = bucket
	return rs, nil
}

func resource_aws_s3_bucket_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	s3conn := p.s3conn

	name := s.Attributes["bucket"]
	bucket := s3conn.Bucket(name)

	log.Printf("[DEBUG] S3 Delete Bucket: %s", name)
	return bucket.DelBucket()
}

func resource_aws_s3_bucket_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	s3conn := p.s3conn

	bucket := s3conn.Bucket(s.Attributes["bucket"])
	resp, err := bucket.Head("/")
	if err != nil {
		return s, err
	}
	defer resp.Body.Close()
	return s, nil
}

func resource_aws_s3_bucket_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"bucket": diff.AttrTypeCreate,
		},
	}
	return b.Diff(s, c)
}
