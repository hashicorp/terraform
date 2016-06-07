package cloudflare

import (
	"fmt"
	"log"
	"strconv"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform/terraform"
)

func resourceCloudFlareRecordMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found CloudFlare Record State v0; migrating to v1")
		return migrateCloudFlareRecordStateV0toV1(is, meta)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateCloudFlareRecordStateV0toV1(is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)
	client := meta.(*cloudflare.API)

	// look up new id based on attributes
	domain := is.Attributes["domain"]
	zoneId, err := client.ZoneIDByName(domain)
	if err != nil {
		return is, fmt.Errorf("Error finding zone %q: %s", domain, err)
	}

	// all other information is ignored in the DNSRecords call
	searchRecord := cloudflare.DNSRecord{
		Type:    is.Attributes["type"],
		Name:    is.Attributes["hostname"],
		Content: is.Attributes["value"],
	}

	records, err := client.DNSRecords(zoneId, searchRecord)
	if err != nil {
		return is, err
	}

	for _, r := range records {
		if is.Attributes["ttl"] != "" {
			v, err := strconv.Atoi(is.Attributes["ttl"])
			if err != nil {
				return is, fmt.Errorf("Error converting ttl to int in CloudFlare Record Migration")
			}

			if v != r.TTL {
				continue
			}
		}

		if is.Attributes["proxied"] != "" {
			b, err := strconv.ParseBool(is.Attributes["proxied"])
			if err != nil {
				return is, fmt.Errorf("Error converting proxied to bool in CloudFlare Record Migration")
			}

			if b != r.Proxied {
				continue
			}
		}

		if is.Attributes["priority"] != "" {
			v, err := strconv.Atoi(is.Attributes["priority"])
			if err != nil {
				return is, fmt.Errorf("Error converting priority to int in CloudFlare Record Migration")
			}

			if v != r.Priority {
				continue
			}
		}

		// assume record found
		is.Attributes["id"] = r.ID
		is.ID = r.ID
		log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
		return is, nil
	}

	// assume no record found
	log.Printf("[DEBUG] Attributes after no migration: %#v", is.Attributes)
	return is, fmt.Errorf("No matching Record found")
}
