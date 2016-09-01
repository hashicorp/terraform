package awspolicy

import (
	"reflect"
	"encoding/json"

	"github.com/hashicorp/errwrap"
)

// PoliciesAreEquivalent tests for the structural equivalence of two
// AWS policies. It does not read into the semantics, other than treating
// single element string arrays as equivalent to a string without an
// array, as the AWS endpoints do.
//
// It will, however, detect reordering and ignore whitespace.
//
// Returns true if the policies are structurally equivalent, false
// otherwise. If either of the input strings are not valid JSON,
// false is returned along with an error.
func PoliciesAreEquivalent(policy1, policy2 string) (bool, error) {
	policy1doc := &awsPolicyDocument{}
	if err := json.Unmarshal([]byte(policy1), policy1doc); err != nil {
		return false, errwrap.Wrapf("Error unmarshaling policy: {{err}}", err)
	}

	policy2doc := &awsPolicyDocument{}
	if err := json.Unmarshal([]byte(policy2), policy2doc); err != nil {
		return false, errwrap.Wrapf("Error unmarshaling policy: {{err}}", err)
	}

	return policy1doc.equals(policy2doc), nil
}

type awsPolicyDocument struct {
	Version    string                `json:",omitempty"`
	Id         string                `json:",omitempty"`
	Statements []*awsPolicyStatement `json:"Statement"`
}

func (doc *awsPolicyDocument) equals(other *awsPolicyDocument) bool {
	// Check the basic fields of the document
	if doc.Version != other.Version {
		return false
	}
	if doc.Id != other.Id {
		return false
	}

	// If we have different number of statements we are very unlikely
	// to have them be equivalent.
	if len(doc.Statements) != len(other.Statements) {
		return false
	}

	// If we have the same number of statements in the policy, does
	// each statement in the doc have a corresponding statement in
	// other which is equal? If no, policies are not equal, if yes,
	// then they may be.
	for _, ours := range doc.Statements {
		found := false
		for _, theirs := range other.Statements {
			if ours.equals(theirs) {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	// Now we need to repeat this process the other way around to
	// ensure we don't have any matching errors.
	for _, theirs := range other.Statements {
		found := false
		for _, ours := range doc.Statements {
			if theirs.equals(ours) {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	return true
}

type awsPolicyStatement struct {
	Sid           string                            `json:",omitempty"`
	Effect        string                            `json:",omitempty"`
	Actions       interface{}                       `json:"Action,omitempty"`
	NotActions    interface{}                       `json:"NotAction,omitempty"`
	Resources     interface{}                       `json:"Resource,omitempty"`
	NotResources  interface{}                       `json:"NotResource,omitempty"`
	Principals    interface{}                       `json:"Principal,omitempty"`
	NotPrincipals interface{}                       `json:"NotPrincipal,omitempty"`
	Conditions    map[string]map[string]interface{} `json:"Condition,omitempty"`
}

func (statement *awsPolicyStatement) equals(other *awsPolicyStatement) bool {
	if statement.Sid != other.Sid {
		return false
	}

	if statement.Effect != other.Effect {
		return false
	}

	ourActions := newAWSStringSet(statement.Actions)
	theirActions := newAWSStringSet(other.Actions)
	if !ourActions.equals(theirActions) {
		return false
	}

	ourNotActions := newAWSStringSet(statement.NotActions)
	theirNotActions := newAWSStringSet(other.NotActions)
	if !ourNotActions.equals(theirNotActions) {
		return false
	}

	ourResources := newAWSStringSet(statement.Resources)
	theirResources := newAWSStringSet(other.Resources)
	if !ourResources.equals(theirResources) {
		return false
	}

	ourNotResources := newAWSStringSet(statement.NotResources)
	theirNotResources := newAWSStringSet(other.NotResources)
	if !ourNotResources.equals(theirNotResources) {
		return false
	}

	ourConditionsBlock := awsConditionsBlock(statement.Conditions)
	theirConditionsBlock := awsConditionsBlock(other.Conditions)
	if !ourConditionsBlock.Equals(theirConditionsBlock) {
		return false
	}

	if statement.Principals != nil || other.Principals != nil {
		stringPrincipalsEqual := stringPrincipalsEqual(statement.Principals, other.Principals)
		mapPrincipalsEqual := mapPrincipalsEqual(statement.Principals, other.Principals)
		if !(stringPrincipalsEqual || mapPrincipalsEqual) {
			return false
		}
	}

	if statement.NotPrincipals != nil || other.NotPrincipals != nil {
		stringNotPrincipalsEqual := stringPrincipalsEqual(statement.NotPrincipals, other.NotPrincipals)
		mapNotPrincipalsEqual := mapPrincipalsEqual(statement.NotPrincipals, other.NotPrincipals)
		if !(stringNotPrincipalsEqual || mapNotPrincipalsEqual) {
			return false
		}
	}

	return true
}

func mapPrincipalsEqual(ours, theirs interface{}) bool {
	ourPrincipalMap, ok := ours.(map[string]interface{})
	if !ok {
		return false
	}

	theirPrincipalMap, ok := theirs.(map[string]interface{})
	if ! ok {
		return false
	}

	oursNormalized := make(map[string]awsStringSet)
	for key, val := range ourPrincipalMap {
		oursNormalized[key] = newAWSStringSet(val)
	}

	theirsNormalized := make(map[string]awsStringSet)
	for key, val := range theirPrincipalMap {
		theirsNormalized[key] = newAWSStringSet(val)
	}

	for key, ours := range oursNormalized {
		theirs, ok := theirsNormalized[key]
		if !ok {
			return false
		}

		if !ours.equals(theirs) {
			return false
		}
	}

	for key, theirs := range theirsNormalized {
		ours, ok := oursNormalized[key]
		if !ok {
			return false
		}

		if !theirs.equals(ours) {
			return false
		}
	}

	return true
}

func stringPrincipalsEqual(ours, theirs interface{}) bool {
	ourPrincipal, oursIsString := ours.(string)
	theirPrincipal, theirsIsString := theirs.(string)

	if !(oursIsString && theirsIsString) {
		return false
	}

	if ourPrincipal == theirPrincipal {
		return true
	}

	return false
}


type awsConditionsBlock map[string]map[string]interface{}

func (conditions awsConditionsBlock) Equals(other awsConditionsBlock) bool {
	if conditions == nil && other != nil || other == nil && conditions != nil {
		return false
	}

	if len(conditions) != len(other) {
		return false
	}

	oursNormalized := make(map[string]map[string]awsStringSet)
	for key, condition := range conditions {
		normalizedCondition := make(map[string]awsStringSet)
		for innerKey, val := range condition {
			normalizedCondition[innerKey] = newAWSStringSet(val)
		}
		oursNormalized[key] = normalizedCondition
	}

	theirsNormalized := make(map[string]map[string]awsStringSet)
	for key, condition := range other {
		normalizedCondition := make(map[string]awsStringSet)
		for innerKey, val := range condition {
			normalizedCondition[innerKey] = newAWSStringSet(val)
		}
		theirsNormalized[key] = normalizedCondition
	}

	for key, ours := range oursNormalized {
		theirs, ok := theirsNormalized[key]
		if !ok {
			return false
		}

		for innerKey, oursInner := range ours {
			theirsInner, ok := theirs[innerKey]
			if ! ok {
				return false
			}

			if !oursInner.equals(theirsInner) {
				return false
			}
		}
	}

	for key, theirs := range theirsNormalized {
		ours, ok := oursNormalized[key]
		if !ok {
			return false
		}

		for innerKey, theirsInner := range theirs {
			oursInner, ok := ours[innerKey]
			if ! ok {
				return false
			}

			if !theirsInner.equals(oursInner) {
				return false
			}
		}
	}

	return true
}


type awsStringSet []string

// newAWSStringSet constructs an awsStringSet from an interface{} - which
// may be nil, a single string, or []interface{} (each of which is a string).
// This corresponds with how structures come off the JSON unmarshaler
// without any custom encoding rules.
func newAWSStringSet(members interface{}) awsStringSet {
	if members == nil {
		return awsStringSet{}
	}

	if single, ok := members.(string); ok {
		return awsStringSet{single}
	}

	if multiple, ok := members.([]interface{}); ok {
		actions := make([]string, len(multiple))
		for i, action := range multiple {
			actions[i] = action.(string)
		}
		return awsStringSet(actions)
	}

	return nil
}

func (actions awsStringSet) equals(other awsStringSet) bool {
	if len(actions) != len(other) {
		return false
	}

	ourMap := map[string]struct{}{}
	theirMap := map[string]struct{}{}

	for _, action := range actions {
		ourMap[action] = struct{}{}
	}

	for _, action := range other {
		theirMap[action] = struct{}{}
	}

	return reflect.DeepEqual(ourMap, theirMap)
}
