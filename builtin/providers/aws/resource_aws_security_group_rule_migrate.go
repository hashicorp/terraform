package aws

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/terraform"
)

func resourceAwsSecurityGroupRuleMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found AWS Security Group State v0; migrating to v1")
		return migrateSGRuleStateV0toV1(is)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}

	return is, nil
}

func migrateSGRuleStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	perm, err := migrateExpandIPPerm(is.Attributes)

	if err != nil {
		return nil, fmt.Errorf("[WARN] Error making new IP Permission in Security Group migration")
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)
	newID := ipPermissionIDHash(is.Attributes["type"], perm)
	is.Attributes["id"] = newID
	is.ID = newID
	log.Printf("[DEBUG] Attributes after migration: %#v, new id: %s", is.Attributes, newID)
	return is, nil
}

func migrateExpandIPPerm(attrs map[string]string) (*ec2.IPPermission, error) {
	var perm ec2.IPPermission
	tp, err := strconv.Atoi(attrs["to_port"])
	if err != nil {
		return nil, fmt.Errorf("Error converting to_port in Security Group migration")
	}

	fp, err := strconv.Atoi(attrs["from_port"])
	if err != nil {
		return nil, fmt.Errorf("Error converting from_port in Security Group migration")
	}

	perm.ToPort = aws.Long(int64(tp))
	perm.FromPort = aws.Long(int64(fp))
	perm.IPProtocol = aws.String(attrs["protocol"])

	groups := make(map[string]bool)
	if attrs["self"] == "true" {
		groups[attrs["security_group_id"]] = true
	}

	if attrs["source_security_group_id"] != "" {
		groups[attrs["source_security_group_id"]] = true
	}

	if len(groups) > 0 {
		perm.UserIDGroupPairs = make([]*ec2.UserIDGroupPair, len(groups))
		// build string list of group name/ids
		var gl []string
		for k, _ := range groups {
			gl = append(gl, k)
		}

		for i, name := range gl {
			perm.UserIDGroupPairs[i] = &ec2.UserIDGroupPair{
				GroupID: aws.String(name),
			}
		}
	}

	var cb []string
	for k, v := range attrs {
		if k != "cidr_blocks.#" && strings.HasPrefix(k, "cidr_blocks") {
			cb = append(cb, v)
		}
	}
	if len(cb) > 0 {
		perm.IPRanges = make([]*ec2.IPRange, len(cb))
		for i, v := range cb {
			perm.IPRanges[i] = &ec2.IPRange{CIDRIP: aws.String(v)}
		}
	}

	return &perm, nil
}
