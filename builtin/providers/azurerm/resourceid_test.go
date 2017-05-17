package azurerm

import (
	"reflect"
	"testing"
)

func TestParseAzureResourceID(t *testing.T) {
	testCases := []struct {
		id                 string
		expectedResourceID *ResourceID
		expectError        bool
	}{
		{
			// Missing "resourceGroups".
			"/subscriptions/00000000-0000-0000-0000-000000000000//myResourceGroup/",
			nil,
			true,
		},
		{
			// Empty resource group ID.
			"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups//",
			nil,
			true,
		},
		{
			"random",
			nil,
			true,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
			nil,
			true,
		},
		{
			"subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
			nil,
			true,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "",
				Path:           map[string]string{},
			},
			false,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.Network",
				Path:           map[string]string{},
			},
			false,
		},
		{
			// Missing leading /
			"subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network/virtualNetworks/virtualNetwork1/",
			nil,
			true,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network/virtualNetworks/virtualNetwork1",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.Network",
				Path: map[string]string{
					"virtualNetworks": "virtualNetwork1",
				},
			},
			false,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network/virtualNetworks/virtualNetwork1?api-version=2006-01-02-preview",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.Network",
				Path: map[string]string{
					"virtualNetworks": "virtualNetwork1",
				},
			},
			false,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network/virtualNetworks/virtualNetwork1/subnets/publicInstances1?api-version=2006-01-02-preview",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.Network",
				Path: map[string]string{
					"virtualNetworks": "virtualNetwork1",
					"subnets":         "publicInstances1",
				},
			},
			false,
		},
		{
			"/subscriptions/34ca515c-4629-458e-bf7c-738d77e0d0ea/resourcegroups/acceptanceTestResourceGroup1/providers/Microsoft.Cdn/profiles/acceptanceTestCdnProfile1",
			&ResourceID{
				SubscriptionID: "34ca515c-4629-458e-bf7c-738d77e0d0ea",
				ResourceGroup:  "acceptanceTestResourceGroup1",
				Provider:       "Microsoft.Cdn",
				Path: map[string]string{
					"profiles": "acceptanceTestCdnProfile1",
				},
			},
			false,
		},
		{
			"/subscriptions/34ca515c-4629-458e-bf7c-738d77e0d0ea/resourceGroups/testGroup1/providers/Microsoft.ServiceBus/namespaces/testNamespace1/topics/testTopic1/subscriptions/testSubscription1",
			&ResourceID{
				SubscriptionID: "34ca515c-4629-458e-bf7c-738d77e0d0ea",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.ServiceBus",
				Path: map[string]string{
					"namespaces":    "testNamespace1",
					"topics":        "testTopic1",
					"subscriptions": "testSubscription1",
				},
			},
			false,
		},
	}

	for _, test := range testCases {
		parsed, err := parseAzureResourceID(test.id)
		if test.expectError && err != nil {
			continue
		}
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		if !reflect.DeepEqual(test.expectedResourceID, parsed) {
			t.Fatalf("Unexpected resource ID:\nExpected: %+v\nGot:      %+v\n", test.expectedResourceID, parsed)
		}
	}
}
