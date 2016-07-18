package aws

import (
	"encoding/json"
)

type IAMPolicyDoc struct {
	Version    string                `json:",omitempty"`
	Id         string                `json:",omitempty"`
	Statements []*IAMPolicyStatement `json:"Statement"`
}

type IAMPolicyStatement struct {
	Sid           string                         `json:",omitempty"`
	Effect        string                         `json:",omitempty"`
	Actions       []string                       `json:"Action,omitempty"`
	NotActions    []string                       `json:"NotAction,omitempty"`
	Resources     []string                       `json:"Resource,omitempty"`
	NotResources  []string                       `json:"NotResource,omitempty"`
	Principals    IAMPolicyStatementPrincipalSet `json:"Principal,omitempty"`
	NotPrincipals IAMPolicyStatementPrincipalSet `json:"NotPrincipal,omitempty"`
	Conditions    IAMPolicyStatementConditionSet `json:"Condition,omitempty"`
}

type IAMPolicyStatementPrincipal struct {
	Type        string
	Identifiers []string
}

type IAMPolicyStatementCondition struct {
	Test     string
	Variable string
	Values   []string
}

type IAMPolicyStatementPrincipalSet []IAMPolicyStatementPrincipal
type IAMPolicyStatementConditionSet []IAMPolicyStatementCondition

func (ps IAMPolicyStatementPrincipalSet) MarshalJSON() ([]byte, error) {
	raw := map[string][]string{}

	for _, p := range ps {
		if _, ok := raw[p.Type]; !ok {
			raw[p.Type] = make([]string, 0, len(p.Identifiers))
		}
		raw[p.Type] = append(raw[p.Type], p.Identifiers...)
	}

	return json.Marshal(&raw)
}

func (cs IAMPolicyStatementConditionSet) MarshalJSON() ([]byte, error) {
	raw := map[string]map[string][]string{}

	for _, c := range cs {
		if _, ok := raw[c.Test]; !ok {
			raw[c.Test] = map[string][]string{}
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
