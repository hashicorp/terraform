package acl

import "fmt"

// Language models the privileges of a language aclitem
type Language struct {
	ACL
}

// NewLanguage parses an ACL object and returns a Language object.
func NewLanguage(acl ACL) (Language, error) {
	if !validRights(acl, validLanguagePrivs) {
		return Language{}, fmt.Errorf("invalid flags set for language (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validLanguagePrivs)
	}

	return Language{ACL: acl}, nil
}
