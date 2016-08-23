package aws

import (
	"encoding/json"
	"sort"
)

// A string array that serializes to JSON as a single string if the length is 1,
// otherwise as a sorted JSON array of strings.
type NormalizeStringArray []string

type IAMPolicyDoc struct {
	Version    string                `json:",omitempty"`
	Id         string                `json:",omitempty"`
	Statements []*IAMPolicyStatement `json:"Statement"`
}

type IAMPolicyStatement struct {
	Sid           string                         `json:",omitempty"`
	Effect        string                         `json:",omitempty"`
	Actions       NormalizeStringArray           `json:"Action,omitempty"`
	NotActions    NormalizeStringArray           `json:"NotAction,omitempty"`
	Resources     NormalizeStringArray           `json:"Resource,omitempty"`
	NotResources  NormalizeStringArray           `json:"NotResource,omitempty"`
	Principals    IAMPolicyStatementPrincipalSet `json:"Principal,omitempty"`
	NotPrincipals IAMPolicyStatementPrincipalSet `json:"NotPrincipal,omitempty"`
	Conditions    IAMPolicyStatementConditionSet `json:"Condition,omitempty"`
}

type IAMPolicyStatementPrincipal struct {
	Type        string
	Identifiers NormalizeStringArray
}

type IAMPolicyStatementCondition struct {
	Test     string
	Variable string
	Values   NormalizeStringArray
}

type IAMPolicyStatementPrincipalSet []IAMPolicyStatementPrincipal
type IAMPolicyStatementConditionSet []IAMPolicyStatementCondition

func (arr NormalizeStringArray) MarshalJSON() ([]byte, error) {
	if len(arr) == 1 {
		return json.Marshal(arr[0])
	}
	sort.Strings([]string(arr))
	return json.Marshal([]string(arr))
}

func (arr *NormalizeStringArray) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err == nil {
		*arr = NormalizeStringArray{s}
		return nil
	}

	var ss []string
	err = json.Unmarshal(data, &ss)
	if err != nil {
		return err
	}
	sort.Strings(ss)
	*arr = NormalizeStringArray(ss)
	return nil
}

func (ps IAMPolicyStatementPrincipalSet) MarshalJSON() ([]byte, error) {
	raw := map[string]NormalizeStringArray{}

	for _, p := range ps {
		if _, ok := raw[p.Type]; !ok {
			raw[p.Type] = make([]string, 0, len(p.Identifiers))
		}
		raw[p.Type] = append(raw[p.Type], p.Identifiers...)
	}

	return json.Marshal(raw)
}

func (ps *IAMPolicyStatementPrincipalSet) UnmarshalJSON(data []byte) error {
	var raw map[string]NormalizeStringArray
	err := json.Unmarshal(data, &raw)
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
	raw := map[string]map[string]NormalizeStringArray{}

	for _, c := range cs {
		if _, ok := raw[c.Test]; !ok {
			raw[c.Test] = map[string]NormalizeStringArray{}
		}
		if _, ok := raw[c.Test][c.Variable]; !ok {
			raw[c.Test][c.Variable] = make([]string, 0, len(c.Values))
		}
		raw[c.Test][c.Variable] = append(raw[c.Test][c.Variable], c.Values...)
	}

	return json.Marshal(&raw)
}

func (cs *IAMPolicyStatementConditionSet) UnmarshalJSON(data []byte) error {
	var raw map[string]map[string]NormalizeStringArray
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

func iamPolicyDecodeConfigStringList(lI []interface{}) []string {
	ret := make([]string, len(lI))
	for i, vI := range lI {
		ret[i] = vI.(string)
	}
	return ret
}
