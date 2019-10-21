package openstack

import (
	"fmt"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
)

var userOptions = map[users.Option]string{
	users.IgnoreChangePasswordUponFirstUse: "ignore_change_password_upon_first_use",
	users.IgnorePasswordExpiry:             "ignore_password_expiry",
	users.IgnoreLockoutFailureAttempts:     "ignore_lockout_failure_attempts",
	users.MultiFactorAuthEnabled:           "multi_factor_auth_enabled",
}

func expandIdentityUserV3MFARules(rules []interface{}) []interface{} {
	var mfaRules []interface{}

	for _, rule := range rules {
		ruleMap := rule.(map[string]interface{})
		ruleList := ruleMap["rule"].([]interface{})
		mfaRules = append(mfaRules, ruleList)
	}

	return mfaRules
}

func flattenIdentityUserV3MFARules(v []interface{}) []map[string]interface{} {
	mfaRules := []map[string]interface{}{}
	for _, rawRule := range v {
		mfaRule := map[string]interface{}{
			"rule": rawRule,
		}
		mfaRules = append(mfaRules, mfaRule)
	}

	return mfaRules
}

// Ensure that password_expires_at query matches format explained in
// https://developer.openstack.org/api-ref/identity/v3/#list-users
func validatePasswordExpiresAtQuery(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	values := strings.SplitN(value, ":", 2)
	if len(values) != 2 {
		err := fmt.Errorf("%s '%s' does not match expected format: {operator}:{timestamp}", k, value)
		errors = append(errors, err)
	}
	operator, timestamp := values[0], values[1]

	validOperators := map[string]bool{
		"lt":  true,
		"lte": true,
		"gt":  true,
		"gte": true,
		"eq":  true,
		"neq": true,
	}
	if !validOperators[operator] {
		err := fmt.Errorf("'%s' is not a valid operator for %s. Choose one of 'lt', 'lte', 'gt', 'gte', 'eq', 'neq'", operator, k)
		errors = append(errors, err)
	}

	_, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		err = fmt.Errorf("'%s' is not a valid timestamp for %s. It should be in the form 'YYYY-MM-DDTHH:mm:ssZ'", timestamp, k)
		errors = append(errors, err)
	}

	return
}
