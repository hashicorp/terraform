package circonus

import (
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

type circonusTag string
type circonusTags []circonusTag

// tagMakeConfigSchema returns a schema pointer to the necessary tag structure.
func tagMakeConfigSchema(tagAttrName schemaAttr) *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Schema{
			Type:             schema.TypeString,
			DiffSuppressFunc: suppressAutoTag(tagAttrName),
			ValidateFunc:     validateTag,
		},
	}
}

func (t circonusTag) Category() string {
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

func (t circonusTag) Value() string {
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

func injectTagPtr(ctxt *providerContext, tagPtrs []*string) circonusTags {
	tags := make(circonusTags, 0, len(tagPtrs))
	for i := range tagPtrs {
		tag := circonusTag(*tagPtrs[i])
		tags = append(tags, tag)
	}
	if ctxt == nil {
		return tags
	}

	return injectTag(ctxt, tags)
}

// injectTag adds a default tag.  If enabled, add a missing preconfigured tag to
// circonusTags.
func injectTag(ctxt *providerContext, tags circonusTags) circonusTags {
	if !globalAutoTag || !ctxt.autoTag {
		return tags
	}

	autoTag := ctxt.defaultTag
	if len(tags) == 0 {
		return circonusTags{autoTag}
	}

	for _, tag := range tags {
		if tag == autoTag {
			return tags
		}
	}

	tags = append(tags, autoTag)

	return tags
}

func configGetTags(ctxt *providerContext, d *schema.ResourceData, attrName schemaAttr) circonusTags {
	if tagsRaw, ok := d.GetOk(string(attrName)); ok {
		tagPtrs := flattenSet(tagsRaw.(*schema.Set))
		return injectTagPtr(ctxt, tagPtrs)
	}

	return injectTag(ctxt, circonusTags{})
}

func suppressAutoTag(tagAttrName schemaAttr) func(k, old, new string, d *schema.ResourceData) bool {
	return func(k, old, new string, d *schema.ResourceData) bool {
		if !globalAutoTag {
			return false
		}

		switch {
		case strings.HasPrefix(k, string(tagAttrName)+".") && old == string(defaultCirconusTag):
			return true
		case k == string(tagAttrName)+".#":
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
}

func tagsToAPI(tags circonusTags) []string {
	apiTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		apiTags = append(apiTags, string(tag))
	}
	return apiTags
}

func tagsToState(tags circonusTags) *schema.Set {
	tagSet := schema.NewSet(schema.HashString, nil)
	for i := range tags {
		tagSet.Add(string(tags[i]))
	}
	return tagSet
}

func apiToTags(apiTags []string) circonusTags {
	tags := make(circonusTags, 0, len(apiTags))
	for _, v := range apiTags {
		tags = append(tags, circonusTag(v))
	}
	return tags
}
