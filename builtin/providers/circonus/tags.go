package circonus

import (
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

type _Tag string
type _Tags []_Tag

// _TagMakeConfigSchema returns a schema pointer to the necessary tag structure.
func _TagMakeConfigSchema(attrName _SchemaAttr) *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Set:      schema.HashString,
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validateTag,
		},
	}
}

func (t _Tag) Category() string {
	tagInfo := strings.SplitN(string(t), ":", 2)
	switch len(tagInfo) {
	case 1:
		return string(t)
	case 2:
		return tagInfo[0]
	default:
		panic("bad")
	}
}

func (t _Tag) Value() string {
	tagInfo := strings.SplitN(string(t), ":", 2)
	switch len(tagInfo) {
	case 1:
		return ""
	case 2:
		return tagInfo[1]
	default:
		panic("bad")
	}
}

func injectTagPtr(ctxt *_ProviderContext, tagPtrs []*string) _Tags {
	tags := make(_Tags, 0, len(tagPtrs))
	for _, t := range tagPtrs {
		tags = append(tags, _Tag(*t))
	}

	return injectTag(ctxt, tags)
}

// injectTag adds a default tag.  If enabled, add a missing preconfigured tag to
// _Tags.
func injectTag(ctxt *_ProviderContext, tags _Tags) _Tags {
	if !globalAutoTag || !ctxt.autoTag {
		return tags
	}

	autoTag := ctxt.defaultTag
	if len(tags) == 0 {
		return _Tags{autoTag}
	}

	for _, tag := range tags {
		if tag == autoTag {
			return tags
		}
	}

	tags = append(tags, autoTag)

	return tags
}

func _ConfigGetTags(ctxt *_ProviderContext, d *schema.ResourceData, attrName _SchemaAttr) _Tags {
	if tagsRaw, ok := d.GetOk(string(attrName)); ok {
		tagPtrs := flattenSet(tagsRaw.(*schema.Set))
		return injectTagPtr(ctxt, tagPtrs)
	}

	return injectTag(ctxt, _Tags{})
}

func suppressAutoTag(k, old, new string, d *schema.ResourceData) bool {
	if !globalAutoTag {
		return false
	}

	switch {
	case strings.HasPrefix(k, string(_MetricTagsAttr)+".") && old == string(defaultCirconusTag):
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

func tagsToAPI(tags _Tags) []string {
	apiTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		apiTags = append(apiTags, string(tag))
	}
	return apiTags
}

func tagsToState(tags _Tags) []interface{} {
	stateTags := make([]interface{}, 0, len(tags))
	for i := range tags {
		stateTags = append(stateTags, string(tags[i]))
	}
	return stateTags
}

func apiToTags(apiTags []string) _Tags {
	tags := make(_Tags, 0, len(apiTags))
	for _, v := range apiTags {
		tags = append(tags, _Tag(v))
	}
	return tags
}
