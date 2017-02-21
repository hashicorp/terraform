package internal

// https://newrelic.atlassian.net/wiki/display/eng/Language+agent+transaction+segment+terms+rules

import (
	"encoding/json"
	"strings"
)

const (
	placeholder = "*"
	separator   = "/"
)

type segmentRule struct {
	Prefix   string   `json:"prefix"`
	Terms    []string `json:"terms"`
	TermsMap map[string]struct{}
}

// segmentRules is keyed by each segmentRule's Prefix field with any trailing
// slash removed.
type segmentRules map[string]*segmentRule

func buildTermsMap(terms []string) map[string]struct{} {
	m := make(map[string]struct{}, len(terms))
	for _, t := range terms {
		m[t] = struct{}{}
	}
	return m
}

func (rules *segmentRules) UnmarshalJSON(b []byte) error {
	var raw []*segmentRule

	if err := json.Unmarshal(b, &raw); nil != err {
		return err
	}

	rs := make(map[string]*segmentRule)

	for _, rule := range raw {
		prefix := strings.TrimSuffix(rule.Prefix, "/")
		if len(strings.Split(prefix, "/")) != 2 {
			// TODO
			// Warn("invalid segment term rule prefix",
			// 	{"prefix": rule.Prefix})
			continue
		}

		if nil == rule.Terms {
			// TODO
			// Warn("segment term rule has missing terms",
			// 	{"prefix": rule.Prefix})
			continue
		}

		rule.TermsMap = buildTermsMap(rule.Terms)

		rs[prefix] = rule
	}

	*rules = rs
	return nil
}

func (rule *segmentRule) apply(name string) string {
	if !strings.HasPrefix(name, rule.Prefix) {
		return name
	}

	s := strings.TrimPrefix(name, rule.Prefix)

	leadingSlash := ""
	if strings.HasPrefix(s, separator) {
		leadingSlash = separator
		s = strings.TrimPrefix(s, separator)
	}

	if "" != s {
		segments := strings.Split(s, separator)

		for i, segment := range segments {
			_, whitelisted := rule.TermsMap[segment]
			if whitelisted {
				segments[i] = segment
			} else {
				segments[i] = placeholder
			}
		}

		segments = collapsePlaceholders(segments)
		s = strings.Join(segments, separator)
	}

	return rule.Prefix + leadingSlash + s
}

func (rules segmentRules) apply(name string) string {
	if nil == rules {
		return name
	}

	rule, ok := rules[firstTwoSegments(name)]
	if !ok {
		return name
	}

	return rule.apply(name)
}

func firstTwoSegments(name string) string {
	firstSlashIdx := strings.Index(name, separator)
	if firstSlashIdx == -1 {
		return name
	}

	secondSlashIdx := strings.Index(name[firstSlashIdx+1:], separator)
	if secondSlashIdx == -1 {
		return name
	}

	return name[0 : firstSlashIdx+secondSlashIdx+1]
}

func collapsePlaceholders(segments []string) []string {
	j := 0
	prevStar := false
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		if placeholder == segment {
			if prevStar {
				continue
			}
			segments[j] = placeholder
			j++
			prevStar = true
		} else {
			segments[j] = segment
			j++
			prevStar = false
		}
	}
	return segments[0:j]
}
