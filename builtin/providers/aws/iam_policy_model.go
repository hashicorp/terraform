package aws

import (
	"encoding/json"
)

// A string array that serializes to JSON as a single string if the length is 1,
// otherwise as a JSON array of strings.
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
	return json.Marshal([]string(arr))
}

func (ps IAMPolicyStatementPrincipalSet) MarshalJSON() ([]byte, error) {
	raw := map[string]NormalizeStringArray{}

	for _, p := range ps {
		if _, ok := raw[p.Type]; !ok {
			raw[p.Type] = make([]string, 0, len(p.Identifiers))
		}
		raw[p.Type] = append(raw[p.Type], p.Identifiers...)
	}

	return json.Marshal(&raw)
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

func iamPolicyDecodeConfigStringList(lI []interface{}) []string {
	ret := make([]string, len(lI))
	for i, vI := range lI {
		ret[i] = vI.(string)
	}
	return ret
}
