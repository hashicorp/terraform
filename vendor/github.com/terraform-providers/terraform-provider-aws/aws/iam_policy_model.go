package aws

import (
	"encoding/json"
	"fmt"
	"sort"
)

type IAMPolicyDoc struct {
	Version    string                `json:",omitempty"`
	Id         string                `json:",omitempty"`
	Statements []*IAMPolicyStatement `json:"Statement"`
}

type IAMPolicyStatement struct {
	Sid           string
	Effect        string                         `json:",omitempty"`
	Actions       interface{}                    `json:"Action,omitempty"`
	NotActions    interface{}                    `json:"NotAction,omitempty"`
	Resources     interface{}                    `json:"Resource,omitempty"`
	NotResources  interface{}                    `json:"NotResource,omitempty"`
	Principals    IAMPolicyStatementPrincipalSet `json:"Principal,omitempty"`
	NotPrincipals IAMPolicyStatementPrincipalSet `json:"NotPrincipal,omitempty"`
	Conditions    IAMPolicyStatementConditionSet `json:"Condition,omitempty"`
}

type IAMPolicyStatementPrincipal struct {
	Type        string
	Identifiers interface{}
}

type IAMPolicyStatementCondition struct {
	Test     string
	Variable string
	Values   interface{}
}

type IAMPolicyStatementPrincipalSet []IAMPolicyStatementPrincipal
type IAMPolicyStatementConditionSet []IAMPolicyStatementCondition

func (self *IAMPolicyDoc) Merge(newDoc *IAMPolicyDoc) {
	// adopt newDoc's Id
	if len(newDoc.Id) > 0 {
		self.Id = newDoc.Id
	}

	// let newDoc upgrade our Version
	if newDoc.Version > self.Version {
		self.Version = newDoc.Version
	}

	// merge in newDoc's statements, overwriting any existing Sids
	var seen bool
	for _, newStatement := range newDoc.Statements {
		if len(newStatement.Sid) == 0 {
			self.Statements = append(self.Statements, newStatement)
			continue
		}
		seen = false
		for i, existingStatement := range self.Statements {
			if existingStatement.Sid == newStatement.Sid {
				self.Statements[i] = newStatement
				seen = true
				break
			}
		}
		if !seen {
			self.Statements = append(self.Statements, newStatement)
		}
	}
}

func (ps IAMPolicyStatementPrincipalSet) MarshalJSON() ([]byte, error) {
	raw := map[string]interface{}{}

	// As a special case, IAM considers the string value "*" to be
	// equivalent to "AWS": "*", and normalizes policies as such.
	// We'll follow their lead and do the same normalization here.
	// IAM also considers {"*": "*"} to be equivalent to this.
	if len(ps) == 1 {
		p := ps[0]
		if p.Type == "AWS" || p.Type == "*" {
			if sv, ok := p.Identifiers.(string); ok && sv == "*" {
				return []byte(`"*"`), nil
			}

			if av, ok := p.Identifiers.([]string); ok && len(av) == 1 && av[0] == "*" {
				return []byte(`"*"`), nil
			}
		}
	}

	for _, p := range ps {
		switch i := p.Identifiers.(type) {
		case []string:
			if _, ok := raw[p.Type]; !ok {
				raw[p.Type] = make([]string, 0, len(i))
			}
			sort.Sort(sort.Reverse(sort.StringSlice(i)))
			raw[p.Type] = append(raw[p.Type].([]string), i...)
		case string:
			raw[p.Type] = i
		default:
			panic("Unsupported data type for IAMPolicyStatementPrincipalSet")
		}
	}

	return json.Marshal(&raw)
}

func (ps *IAMPolicyStatementPrincipalSet) UnmarshalJSON(b []byte) error {
	var out IAMPolicyStatementPrincipalSet

	var data interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	switch t := data.(type) {
	case string:
		out = append(out, IAMPolicyStatementPrincipal{Type: "*", Identifiers: []string{"*"}})
	case map[string]interface{}:
		for key, value := range data.(map[string]interface{}) {
			out = append(out, IAMPolicyStatementPrincipal{Type: key, Identifiers: value})
		}
	default:
		return fmt.Errorf("Unsupported data type %s for IAMPolicyStatementPrincipalSet", t)
	}

	*ps = out
	return nil
}

func (cs IAMPolicyStatementConditionSet) MarshalJSON() ([]byte, error) {
	raw := map[string]map[string]interface{}{}

	for _, c := range cs {
		if _, ok := raw[c.Test]; !ok {
			raw[c.Test] = map[string]interface{}{}
		}
		switch i := c.Values.(type) {
		case []string:
			if _, ok := raw[c.Test][c.Variable]; !ok {
				raw[c.Test][c.Variable] = make([]string, 0, len(i))
			}
			sort.Sort(sort.Reverse(sort.StringSlice(i)))
			raw[c.Test][c.Variable] = append(raw[c.Test][c.Variable].([]string), i...)
		case string:
			raw[c.Test][c.Variable] = i
		default:
			panic("Unsupported data type for IAMPolicyStatementConditionSet")
		}
	}

	return json.Marshal(&raw)
}

func (cs *IAMPolicyStatementConditionSet) UnmarshalJSON(b []byte) error {
	var out IAMPolicyStatementConditionSet

	var data map[string]map[string]interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	for test_key, test_value := range data {
		for var_key, var_values := range test_value {
			switch var_values.(type) {
			case string:
				out = append(out, IAMPolicyStatementCondition{Test: test_key, Variable: var_key, Values: []string{var_values.(string)}})
			case []interface{}:
				values := []string{}
				for _, v := range var_values.([]interface{}) {
					values = append(values, v.(string))
				}
				out = append(out, IAMPolicyStatementCondition{Test: test_key, Variable: var_key, Values: values})
			}
		}
	}

	*cs = out
	return nil
}

func iamPolicyDecodeConfigStringList(lI []interface{}) interface{} {
	if len(lI) == 1 {
		return lI[0].(string)
	}
	ret := make([]string, len(lI))
	for i, vI := range lI {
		ret[i] = vI.(string)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ret)))
	return ret
}
