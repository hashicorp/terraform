package aws

import (
	"bytes"
	"encoding/json"
	"log"
	"net/url"
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
	// First check if the old/new values are nil.
	// If both are nil, we have no state to compare the values with, so register a diff.
	// This populates the attribute field during a plan/apply with fresh state, allowing
	// the attribute to still be used in future resources.
	// See https://github.com/hashicorp/terraform/issues/11881
	if old == "" && new == "" {
		return false
	}

	if v, ok := d.GetOk("auto_minor_version_upgrade"); ok {
		if v.(bool) {
			// If we're set to auto upgrade minor versions
			// ignore a minor version diff between versions
			if strings.HasPrefix(old, new) {
				log.Printf("[DEBUG] Ignoring minor version diff")
				return true
			}
		}
	}

	// Throw a diff by default
	return false
}

func suppressEquivalentJsonDiffs(k, old, new string, d *schema.ResourceData) bool {
	ob := bytes.NewBufferString("")
	if err := json.Compact(ob, []byte(old)); err != nil {
		return false
	}

	nb := bytes.NewBufferString("")
	if err := json.Compact(nb, []byte(new)); err != nil {
		return false
	}

	return jsonBytesEqual(ob.Bytes(), nb.Bytes())
}

func suppressOpenIdURL(k, old, new string, d *schema.ResourceData) bool {
	oldUrl, err := url.Parse(old)
	if err != nil {
		return false
	}

	newUrl, err := url.Parse(new)
	if err != nil {
		return false
	}

	oldUrl.Scheme = "https"

	return oldUrl.String() == newUrl.String()
}
