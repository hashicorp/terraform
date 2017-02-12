package aws

import (
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/awspolicyequivalence"
)

func suppressEquivalentAwsPolicyDiffs(k, old, new string, d *schema.ResourceData) bool {
	equivalent, err := awspolicy.PoliciesAreEquivalent(old, new)
	if err != nil {
		return false
	}

	return equivalent
}

// Suppresses minor version changes to the db_instance engine_version attribute
func suppressAwsDbEngineVersionDiffs(k, old, new string, d *schema.ResourceData) bool {
	if d.Get("auto_minor_version_upgrade").(bool) {
		// If we're set to auto upgrade minor versions
		// ignore a minor version diff between versions
		if strings.HasPrefix(old, new) {
			log.Printf("[DEBUG] Ignoring minor version diff")
			return true
		}
	}

	// Throw a diff by default
	return false
}
