package internal

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

type ruleResult int

const (
	ruleChanged ruleResult = iota
	ruleUnchanged
	ruleIgnore
)

type metricRule struct {
	// 'Ignore' indicates if the entire transaction should be discarded if
	// there is a match.  This field is only used by "url_rules" and
	// "transaction_name_rules", not "metric_name_rules".
	Ignore              bool   `json:"ignore"`
	EachSegment         bool   `json:"each_segment"`
	ReplaceAll          bool   `json:"replace_all"`
	Terminate           bool   `json:"terminate_chain"`
	Order               int    `json:"eval_order"`
	OriginalReplacement string `json:"replacement"`
	RawExpr             string `json:"match_expression"`

	// Go's regexp backreferences use '${1}' instead of the Perlish '\1', so
	// we transform the replacement string into the Go syntax and store it
	// here.
	TransformedReplacement string
	re                     *regexp.Regexp
}

type metricRules []*metricRule

// Go's regexp backreferences use `${1}` instead of the Perlish `\1`, so we must
// transform the replacement string.  This is non-trivial: `\1` is a
// backreference but `\\1` is not.  Rather than count the number of back slashes
// preceding the digit, we simply skip rules with tricky replacements.
var (
	transformReplacementAmbiguous   = regexp.MustCompile(`\\\\([0-9]+)`)
	transformReplacementRegex       = regexp.MustCompile(`\\([0-9]+)`)
	transformReplacementReplacement = "$${${1}}"
)

func (rules *metricRules) UnmarshalJSON(data []byte) (err error) {
	var raw []*metricRule

	if err := json.Unmarshal(data, &raw); nil != err {
		return err
	}

	valid := make(metricRules, 0, len(raw))

	for _, r := range raw {
		re, err := regexp.Compile("(?i)" + r.RawExpr)
		if err != nil {
			// TODO
			// Warn("unable to compile rule", {
			// 	"match_expression": r.RawExpr,
			// 	"error":            err.Error(),
			// })
			continue
		}

		if transformReplacementAmbiguous.MatchString(r.OriginalReplacement) {
			// TODO
			// Warn("unable to transform replacement", {
			// 	"match_expression": r.RawExpr,
			// 	"replacement":      r.OriginalReplacement,
			// })
			continue
		}

		r.re = re
		r.TransformedReplacement = transformReplacementRegex.ReplaceAllString(r.OriginalReplacement,
			transformReplacementReplacement)
		valid = append(valid, r)
	}

	sort.Sort(valid)

	*rules = valid
	return nil
}

func (rules metricRules) Len() int {
	return len(rules)
}

// Rules should be applied in increasing order
func (rules metricRules) Less(i, j int) bool {
	return rules[i].Order < rules[j].Order
}
func (rules metricRules) Swap(i, j int) {
	rules[i], rules[j] = rules[j], rules[i]
}

func replaceFirst(re *regexp.Regexp, s string, replacement string) string {
	// Note that ReplaceAllStringFunc cannot be used here since it does
	// not replace $1 placeholders.
	loc := re.FindStringIndex(s)
	if nil == loc {
		return s
	}
	firstMatch := s[loc[0]:loc[1]]
	firstMatchReplaced := re.ReplaceAllString(firstMatch, replacement)
	return s[0:loc[0]] + firstMatchReplaced + s[loc[1]:]
}

func (r *metricRule) apply(s string) (ruleResult, string) {
	// Rules are strange, and there is no spec.
	// This code attempts to duplicate the logic of the PHP agent.
	// Ambiguity abounds.

	if r.Ignore {
		if r.re.MatchString(s) {
			return ruleIgnore, ""
		}
		return ruleUnchanged, s
	}

	var out string

	if r.ReplaceAll {
		out = r.re.ReplaceAllString(s, r.TransformedReplacement)
	} else if r.EachSegment {
		segments := strings.Split(string(s), "/")
		applied := make([]string, len(segments))
		for i, segment := range segments {
			applied[i] = replaceFirst(r.re, segment, r.TransformedReplacement)
		}
		out = strings.Join(applied, "/")
	} else {
		out = replaceFirst(r.re, s, r.TransformedReplacement)
	}

	if out == s {
		return ruleUnchanged, out
	}
	return ruleChanged, out
}

func (rules metricRules) Apply(input string) string {
	var res ruleResult
	s := input

	for _, rule := range rules {
		res, s = rule.apply(s)

		if ruleIgnore == res {
			return ""
		}
		if (ruleChanged == res) && rule.Terminate {
			break
		}
	}

	return s
}
