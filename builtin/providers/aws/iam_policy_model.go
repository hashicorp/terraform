package aws

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/helper/schema"
)

// IAMPolicyDoc is an in-memory representation of an IAM policy document
// with annotations to marshal to and unmarshal from JSON policy syntax.
//
// Round-tripping through this struct will normalize a policy, but it
// is only guaranteed to work with policy versions 2012-10-17 or earlier.
// Newer policies may be silently corrupted by round-tripping through
// this structure. (At the time of writing there is no newer policy version.)
type IAMPolicyDoc struct {
	Version    string                `json:",omitempty"`
	Id         string                `json:",omitempty"`
	Statements []*IAMPolicyStatement `json:"Statement,omitempty"`
}

type IAMPolicyStatement struct {
	Sid           string                         `json:",omitempty"`
	Effect        string                         `json:",omitempty"`
	Actions       IAMPolicyStringSet             `json:"Action,omitempty"`
	NotActions    IAMPolicyStringSet             `json:"NotAction,omitempty"`
	Resources     IAMPolicyStringSet             `json:"Resource,omitempty"`
	NotResources  IAMPolicyStringSet             `json:"NotResource,omitempty"`
	Principals    IAMPolicyStatementPrincipalSet `json:"Principal,omitempty"`
	NotPrincipals IAMPolicyStatementPrincipalSet `json:"NotPrincipal,omitempty"`
	Conditions    IAMPolicyStatementConditionSet `json:"Condition,omitempty"`
}

type IAMPolicyStatementPrincipal struct {
	Type        string
	Identifiers IAMPolicyStringSet
}

type IAMPolicyStatementCondition struct {
	Test     string
	Variable string
	Values   IAMPolicyStringSet
}

// IAMPolicyStringSet is a specialization of []string that has special normalization
// rules for unmarshalling from JSON. Specifically, a single-item list is considered
// equivalent to a plain string, and a multi-item list is sorted into lexographical
// order to reflect that the ordering is not meaningful.
//
// When marshalling, the set is again ordered lexographically (in case it has been
// modified by code that didn't preserve the order) and single-item lists are serialized
// as primitive strings, since IAM considers this to be the normalized form.
type IAMPolicyStringSet []string

type IAMPolicyStatementPrincipalSet []IAMPolicyStatementPrincipal
type IAMPolicyStatementConditionSet []IAMPolicyStatementCondition

func (ss *IAMPolicyStringSet) UnmarshalJSON(data []byte) error {
	var single string
	err := json.Unmarshal(data, &single)
	if err == nil {
		*ss = IAMPolicyStringSet{single}
		return nil
	}

	var set []string
	err = json.Unmarshal(data, &set)
	if err == nil {
		*ss = IAMPolicyStringSet(set)
		sort.Strings(*ss)
		return nil
	}

	return fmt.Errorf("must be string or array of strings")
}

func (ss IAMPolicyStringSet) MarshalJSON() ([]byte, error) {
	if len(ss) == 1 {
		return json.Marshal(ss[0])
	}

	sort.Strings([]string(ss))
	return json.Marshal([]string(ss))
}

func (ps IAMPolicyStatementPrincipalSet) MarshalJSON() ([]byte, error) {
	raw := map[string]IAMPolicyStringSet{}

	for _, p := range ps {
		if _, ok := raw[p.Type]; !ok {
			raw[p.Type] = make(IAMPolicyStringSet, 0, len(p.Identifiers))
		}
		raw[p.Type] = append(raw[p.Type], p.Identifiers...)
	}

	return json.Marshal(raw)
}

func (ps *IAMPolicyStatementPrincipalSet) UnmarshalJSON(data []byte) error {
	var wildcard string
	err := json.Unmarshal(data, &wildcard)
	if err == nil {
		if wildcard != "*" {
			return fmt.Errorf(
				"Principal and NotPrincipal must be either object or the string \"*\"",
			)
		}

		// This wildcard is an alias for all/anonymous AWS principals.
		// After round-tripping, this will normalize as an explicit wildcard under
		// the "AWS" provider.
		*ps = IAMPolicyStatementPrincipalSet{
			IAMPolicyStatementPrincipal{
				Type:        "AWS",
				Identifiers: IAMPolicyStringSet{wildcard},
			},
		}
		return nil
	}

	var raw map[string]IAMPolicyStringSet
	err = json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	principalTypes := make([]string, 0, len(raw))
	for k := range raw {
		principalTypes = append(principalTypes, k)
	}
	sort.Strings(principalTypes)

	*ps = make(IAMPolicyStatementPrincipalSet, 0, len(raw))
	for _, principalType := range principalTypes {
		*ps = append(*ps, IAMPolicyStatementPrincipal{
			Type:        principalType,
			Identifiers: raw[principalType],
		})
	}
	return nil
}

func (cs IAMPolicyStatementConditionSet) MarshalJSON() ([]byte, error) {
	raw := map[string]map[string]IAMPolicyStringSet{}

	for _, c := range cs {
		if _, ok := raw[c.Test]; !ok {
			raw[c.Test] = map[string]IAMPolicyStringSet{}
		}
		if _, ok := raw[c.Test][c.Variable]; !ok {
			raw[c.Test][c.Variable] = make(IAMPolicyStringSet, 0, len(c.Values))
		}
		raw[c.Test][c.Variable] = append(raw[c.Test][c.Variable], c.Values...)
	}

	return json.Marshal(&raw)
}

func (cs *IAMPolicyStatementConditionSet) UnmarshalJSON(data []byte) error {
	var raw map[string]map[string]IAMPolicyStringSet
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	tests := make([]string, 0, len(raw))
	count := 0
	for k, v := range raw {
		tests = append(tests, k)
		count += len(v)
	}
	sort.Strings(tests)

	*cs = make(IAMPolicyStatementConditionSet, 0, count)
	for _, test := range tests {
		variables := make([]string, 0, len(raw[test]))
		for k := range raw[test] {
			variables = append(variables, k)
		}
		sort.Strings(variables)

		for _, variable := range variables {
			*cs = append(*cs, IAMPolicyStatementCondition{
				Test:     test,
				Variable: variable,
				Values:   raw[test][variable],
			})
		}
	}
	return nil
}

// NormalizeIAMPolicyJSON takes an IAM policy in JSON format and produces
// an equivalent JSON document with normalizations applied. In particular,
// single-element string lists are serialized as standalone strings,
// multi-element string lists are sorted lexographically, and the
// policy element attributes are written in a predictable order.
//
// In the event of an error, the result is the input buffer, verbatim.
func NormalizeIAMPolicyJSON(in []byte, cb IAMPolicyStatementNormalizer) ([]byte, error) {
	doc := &IAMPolicyDoc{}
	err := json.Unmarshal([]byte(in), doc)
	if err != nil {
		return in, err
	}

	if cb != nil && doc.Statements != nil {
		// Caller wants to do some additional normalization
		for i, stmt := range doc.Statements {
			// Callback modifies statement data in-place
			err := cb(stmt)
			if err != nil {
				return in, fmt.Errorf("statement #%d: %s", i+1, err)
			}
		}
	}

	resultBytes, err := json.Marshal(doc)
	if err != nil {
		return in, err
	}

	return resultBytes, nil
}

type IAMPolicyStatementNormalizer func(*IAMPolicyStatement) error

// iamPolicyJSONStateFunc can be used as a StateFunc for attributes that
// take IAM policies in JSON format.
// Should usually be used in conjunction with iamPolicyJSONValidateFunc.
func iamPolicyJSONStateFunc(jsonSrcI interface{}) string {
	// Safe to ignore the error because NormalizeIAMPolicyJSON will pass through
	// the given string verbatim if it's not valid.
	result, _ := NormalizeIAMPolicyJSON([]byte(jsonSrcI.(string)), nil)
	return string(result)
}

// iamPolicyJSONCustomStateFunc produces a function that can be used as a StateFunc
// for attributes that take IAM policies in JSON format and that need further
// resource-specific normalization via a normalization callback.
func iamPolicyJSONCustomStateFunc(cb IAMPolicyStatementNormalizer) schema.SchemaStateFunc {
	return func(jsonSrcI interface{}) string {
		result, _ := NormalizeIAMPolicyJSON([]byte(jsonSrcI.(string)), cb)
		return string(result)
	}
}

// Can be used as a ValidateFunc for attributes that take IAM policies in JSON format.
// Does simple syntactic validation.
func iamPolicyJSONValidateFunc(jsonSrcI interface{}, _ string) ([]string, []error) {
	_, err := NormalizeIAMPolicyJSON([]byte(jsonSrcI.(string)), nil)
	if err != nil {
		return nil, []error{err}
	}

	return nil, nil
}

func iamPolicyDecodeConfigStringList(configList []interface{}) IAMPolicyStringSet {
	ret := make([]string, len(configList))
	for i, valueI := range configList {
		ret[i] = valueI.(string)
	}
	sort.Strings(ret)
	return ret
}
