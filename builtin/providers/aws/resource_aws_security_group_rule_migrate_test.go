package aws

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAWSSecurityGroupRuleMigrateState(t *testing.T) {
	//   "id":"sg-4235098228", "from_port":"0", "source_security_group_id":"sg-11877275"}

	// 2015/06/16 16:04:21 terraform-provider-aws: 2015/06/16 16:04:21 [DEBUG] Attributes after migration:

	// map[string]string{"from_port":"0", "source_security_group_id":"sg-11877275", "id":"sg-3766347571", "security_group_id":"sg-13877277", "cidr_blocks.#":"0", "type":"ingress", "protocol":"-1", "self":"false", "to_port":"0"}, new id: sg-3766347571
	cases := map[string]struct {
		StateVersion int
		Attributes   map[string]string
		Expected     string
		Meta         interface{}
	}{
		"v0_1": {
			StateVersion: 0,
			Attributes: map[string]string{
				"self":                     "false",
				"to_port":                  "0",
				"security_group_id":        "sg-13877277",
				"cidr_blocks.#":            "0",
				"type":                     "ingress",
				"protocol":                 "-1",
				"id":                       "sg-4235098228",
				"from_port":                "0",
				"source_security_group_id": "sg-11877275",
			},
			Expected: "sg-3766347571",
		},
		// "v0_2": {
		// 	StateVersion: 0,
		// 	Attributes: map[string]string{
		// 		// EBS
		// 		"self": "false",
		// 	},
		// 	Expected: "sg-1235",
		// },
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "sg-4235098228",
			Attributes: tc.Attributes,
		}
		is, err := resourceAwsSecurityGroupRuleMigrateState(
			tc.StateVersion, is, tc.Meta)

		if err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		if is.ID != tc.Expected {
			t.Fatalf("bad sg rule id: %s\n\n expected: %s", is.ID, tc.Expected)
		}
	}
}
