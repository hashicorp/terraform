package circonus

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func castSchemaToTF(in map[_SchemaAttr]*schema.Schema, descrs _AttrDescrs) map[string]*schema.Schema {
	out := make(map[string]*schema.Schema, len(in))
	for k, v := range in {
		if descr, ok := descrs[k]; ok {
			v.Description = string(descr)
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

// injectTag adds the context's
func injectTag(ctxt *providerContext, tags _Tags, overrideTag _Tag) _Tags {
	if !globalAutoTag || !ctxt.autoTag {
		return tags
	}

	tag := ctxt.defaultTag
	if overrideTag.Category != "" {
		tag = overrideTag
	}

	if len(tags) == 0 {
		return _Tags{
			tag.Category: tag.Value,
		}
	}

	if val, found := tags[ctxt.defaultTag.Category]; found && val == ctxt.defaultTag.Value {
		return tags
	}

	if val, found := tags[tag.Category]; found && val == tag.Value {
		return tags
	}

	tags[tag.Category] = tag.Value

	return tags
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

// schemaGetString returns an attribute as a string.  If the attribute is not
// found, return an empty string.
func schemaGetString(d *schema.ResourceData, attrName _SchemaAttr) string {
	if v, ok := d.GetOk(string(attrName)); ok {
		return v.(string)
	}

	return ""
}

func schemaGetTags(ctxt *providerContext, d *schema.ResourceData, attrName _SchemaAttr, defaultTag _Tag) _Tags {
	var tags _Tags
	if tagsRaw, ok := d.GetOk(string(attrName)); ok {
		tagsMap := tagsRaw.(map[string]interface{})

		tags = make(_Tags, len(tagsMap))
		for k, v := range tagsMap {
			tags[_TagCategory(k)] = _TagValue(v.(string))
		}
	}

	return injectTag(ctxt, tags, defaultTag)
}

// stateSet sets an attribute based on an attrName and panic()'s if the Set
// failed.
func stateSet(d *schema.ResourceData, attrName _SchemaAttr, v interface{}) {
	if err := d.Set(string(attrName), v); err != nil {
		panic(fmt.Sprintf("Provider Bug: failed set schema attribute %s to value %#v", attrName, v))
	}
}

func suppressAutoTag(k, old, new string, d *schema.ResourceData) bool {
	if !globalAutoTag {
		return false
	}

	switch {
	case k == string(_MetricTagsAttr)+"."+string(defaultCirconusTagCategory) && old == string(defaultCirconusTagValue):
		return true
	case k == string(_MetricTagsAttr)+".%":
		oldNum, err := strconv.ParseInt(old, 10, 32)
		if err != nil {
			return false
		}

		newNum, err := strconv.ParseInt(new, 10, 32)
		if err != nil {
			return false
		}

		return (oldNum - 1) == newNum
	default:
		return false
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

func tagsToAPI(tags _Tags) []string {
	apiTags := make([]string, 0, len(tags))
	for k, v := range tags {
		apiTags = append(apiTags, string(k)+":"+string(v))
	}
	sort.Strings(apiTags)
	return apiTags
}
