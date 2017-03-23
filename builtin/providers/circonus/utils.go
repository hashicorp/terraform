package circonus

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

// convertToHelperSchema converts the schema and injects the necessary
// parameters, notably the descriptions, in order to be valid input to
// Terraform's helper schema.
func convertToHelperSchema(descrs attrDescrs, in map[schemaAttr]*schema.Schema) map[string]*schema.Schema {
	out := make(map[string]*schema.Schema, len(in))
	for k, v := range in {
		if descr, ok := descrs[k]; ok {
			// NOTE(sean@): At some point this check needs to be uncommented and all
			// empty descriptions need to be populated.
			//
			// if len(descr) == 0 {
			// 	log.Printf("[WARN] PROVIDER BUG: Description of attribute %s empty", k)
			// }

			v.Description = string(descr)
		} else {
			log.Printf("[WARN] PROVIDER BUG: Unable to find description for attr %q", k)
		}

		out[string(k)] = v
	}

	return out
}

func failoverGroupIDToCID(groupID int) string {
	if groupID == 0 {
		return ""
	}

	return fmt.Sprintf("%s/%d", config.ContactGroupPrefix, groupID)
}

func failoverGroupCIDToID(cid api.CIDType) (int, error) {
	re := regexp.MustCompile("^" + config.ContactGroupPrefix + "/(" + config.DefaultCIDRegex + ")$")
	matches := re.FindStringSubmatch(string(*cid))
	if matches == nil || len(matches) < 2 {
		return -1, fmt.Errorf("Did not find a valid contact_group ID in the CID %q", string(*cid))
	}

	contactGroupID, err := strconv.Atoi(matches[1])
	if err != nil {
		return -1, errwrap.Wrapf(fmt.Sprintf("invalid contact_group ID: unable to find an ID in %q: {{error}}", string(*cid)), err)
	}

	return contactGroupID, nil
}

// flattenList returns a list of all string values to a []*string.
func flattenList(l []interface{}) []*string {
	vals := make([]*string, 0, len(l))
	for _, v := range l {
		val, ok := v.(string)
		if ok && val != "" {
			vals = append(vals, &val)
		}
	}
	return vals
}

// flattenSet flattens the values in a schema.Set and returns a []*string
func flattenSet(s *schema.Set) []*string {
	return flattenList(s.List())
}

func derefStringList(lp []*string) []string {
	l := make([]string, 0, len(lp))
	for _, sp := range lp {
		if sp != nil {
			l = append(l, *sp)
		}
	}
	return l
}

// listToSet returns a TypeSet from the given list.
func stringListToSet(stringList []string, keyName schemaAttr) []interface{} {
	m := make([]interface{}, 0, len(stringList))
	for _, v := range stringList {
		s := make(map[string]interface{}, 1)
		s[string(keyName)] = v
		m = append(m, s)
	}

	return m
}

func normalizeTimeDurationStringToSeconds(v interface{}) string {
	switch v.(type) {
	case string:
		d, err := time.ParseDuration(v.(string))
		if err != nil {
			return fmt.Sprintf("<unable to normalize time duration %s: %v>", v.(string), err)
		}

		return fmt.Sprintf("%ds", int(d.Seconds()))
	default:
		return fmt.Sprintf("<unable to normalize duration on %#v>", v)
	}
}

func indirect(v interface{}) interface{} {
	switch v.(type) {
	case string:
		return v
	case *string:
		p := v.(*string)
		if p == nil {
			return nil
		}
		return *p
	default:
		return v
	}
}

func suppressEquivalentTimeDurations(k, old, new string, d *schema.ResourceData) bool {
	d1, err := time.ParseDuration(old)
	if err != nil {
		return false
	}

	d2, err := time.ParseDuration(new)
	if err != nil {
		return false
	}

	return d1 == d2
}

func suppressWhitespace(v interface{}) string {
	return strings.TrimSpace(v.(string))
}
