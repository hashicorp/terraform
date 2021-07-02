package addrs

import (
	"fmt"
	"testing"
)

func TestConfigResourceIncludedInMoveable(t *testing.T) {
	tests := []struct {
		Receiver ConfigResource
		Moveable ConfigMoveable
		Want     bool
	}{
		{
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			true,
		},
		{
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: DataResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			false, // mode doesn't match
		},
		{
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "c",
					Name: "b",
				},
			},
			false, // type doesn't match
		},
		{
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "c",
				},
			},
			false, // name doesn't match
		},
		{
			ConfigResource{
				Module: RootModule.Child("foo"),
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			false, // reciever is in a different module
		},

		{
			ConfigResource{
				Module: RootModule.Child("foo"),
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			RootModule.Child("foo"),
			true, // receiver is in the given module
		},
		{
			ConfigResource{
				Module: RootModule.Child("foo").Child("bar"),
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			RootModule.Child("foo"),
			true, // receiver is in a descendent of the given module
		},
		{
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			RootModule.Child("foo"),
			false, // receiver is in parent of the given module
		},
		{
			ConfigResource{
				Module: RootModule.Child("bar"),
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			RootModule.Child("foo"),
			false, // receiver is in sibling of the given module
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s in %s", test.Receiver, test.Moveable), func(t *testing.T) {
			got := test.Receiver.IncludedInMoveable(test.Moveable)
			if got != test.Want {
				t.Errorf(
					"wrong result\nreciever: %s\nmoveable: %s\ngot:  %t\nwant: %t",
					test.Receiver, test.Moveable,
					got, test.Want,
				)
			}
		})
	}
}

func TestModuleIncludedInMoveable(t *testing.T) {
	tests := []struct {
		Receiver Module
		Moveable ConfigMoveable
		Want     bool
	}{
		{
			RootModule.Child("a"),
			RootModule.Child("a"),
			true, // a module is included in itself
		},
		{
			RootModule.Child("a").Child("b"),
			RootModule.Child("a"),
			true, // a module is included in its parent
		},
		{
			RootModule.Child("a").Child("b").Child("c"),
			RootModule.Child("a"),
			true, // a module is included in its ancestors
		},
		{
			RootModule.Child("a"),
			RootModule.Child("a").Child("b"),
			false, // a module is not included in its child
		},
		{
			RootModule.Child("a"),
			RootModule.Child("a").Child("b").Child("c"),
			false, // a module is not included in its descendents
		},
		{
			RootModule.Child("a"),
			ConfigResource{
				Module: RootModule,
				Resource: Resource{
					Mode: ManagedResourceMode,
					Type: "a",
					Name: "b",
				},
			},
			false, // a module can never be inside a resource
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s in %s", test.Receiver, test.Moveable), func(t *testing.T) {
			got := test.Receiver.IncludedInMoveable(test.Moveable)
			if got != test.Want {
				t.Errorf(
					"wrong result\nreciever: %s\nmoveable: %s\ngot:  %t\nwant: %t",
					test.Receiver, test.Moveable,
					got, test.Want,
				)
			}
		})
	}
}
