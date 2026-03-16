// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"testing"
)

func TestActionEqual(t *testing.T) {
	actions := []Action{
		{Type: "foo", Name: "bar"},
		{Type: "the", Name: "bloop"},
	}
	for _, r := range actions {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}

	// not equal
	testCases := []struct {
		right Action
		left  Action
	}{
		{
			Action{Type: "a", Name: "b"},
			Action{Type: "b", Name: "b"},
		},
		{
			Action{Type: "a", Name: "b"},
			Action{Type: "a", Name: "c"},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}

func TestActionInstanceEqual(t *testing.T) {
	actions := []ActionInstance{
		{
			Action: Action{Type: "foo", Name: "bar"},
			Key:    NoKey,
		},
		{
			Action: Action{Type: "the", Name: "bloop"},
			Key:    StringKey("fish"),
		},
	}
	for _, r := range actions {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}

	// not equal
	testCases := []struct {
		right ActionInstance
		left  ActionInstance
	}{
		{
			ActionInstance{
				Action: Action{Type: "foo", Name: "bar"},
				Key:    NoKey,
			},
			ActionInstance{
				Action: Action{Type: "foo", Name: "bar"},
				Key:    IntKey(1),
			},
		},
		{
			ActionInstance{
				Action: Action{Type: "foo", Name: "bar"},
				Key:    NoKey,
			},
			ActionInstance{
				Action: Action{Type: "baz", Name: "bat"},
				Key:    IntKey(1),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}

func TestAbsActionInstanceEqual(t *testing.T) {
	actions := []AbsActionInstance{
		{
			Module: RootModuleInstance,
			Action: ActionInstance{
				Action: Action{Type: "foo", Name: "bar"},
				Key:    NoKey,
			},
		},
		{
			Module: mustParseModuleInstanceStr("module.child"),
			Action: ActionInstance{
				Action: Action{Type: "the", Name: "bloop"},
				Key:    StringKey("fish"),
			},
		},
	}

	for _, r := range actions {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}

	// not equal
	testCases := []struct {
		right AbsActionInstance
		left  AbsActionInstance
	}{
		{ // different keys
			AbsActionInstance{
				Module: RootModuleInstance,
				Action: ActionInstance{
					Action: Action{Type: "foo", Name: "bar"},
					Key:    NoKey,
				},
			},
			AbsActionInstance{
				Module: RootModuleInstance,
				Action: ActionInstance{
					Action: Action{Type: "foo", Name: "bar"},
					Key:    IntKey(1),
				},
			},
		},

		{ // different module
			AbsActionInstance{
				Module: RootModuleInstance,
				Action: ActionInstance{
					Action: Action{Type: "foo", Name: "bar"},
					Key:    NoKey,
				},
			},
			AbsActionInstance{
				Module: mustParseModuleInstanceStr("module.child[1]"),
				Action: ActionInstance{
					Action: Action{Type: "foo", Name: "bar"},
					Key:    NoKey,
				},
			},
		},

		{ // totally different
			AbsActionInstance{
				Module: RootModuleInstance,
				Action: ActionInstance{
					Action: Action{Type: "oof", Name: "rab"},
					Key:    NoKey,
				},
			},
			AbsActionInstance{
				Module: mustParseModuleInstanceStr("module.foo"),
				Action: ActionInstance{
					Action: Action{Type: "foo", Name: "bar"},
					Key:    IntKey(11),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}

// TestConfigActionEqual
func TestConfigActionEqual(t *testing.T) {
	actions := []ConfigAction{
		{
			RootModule,
			Action{Type: "foo", Name: "bar"},
		},
		{
			Module{"child"},
			Action{Type: "the", Name: "bloop"},
		},
	}
	for _, r := range actions {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}

	// not equal
	testCases := []struct {
		right ConfigAction
		left  ConfigAction
	}{
		{ // different name
			ConfigAction{
				RootModule,
				Action{Type: "foo", Name: "bar"},
			},
			ConfigAction{
				RootModule,
				Action{Type: "foo", Name: "baz"},
			},
		},
		// different type
		{
			ConfigAction{
				RootModule,
				Action{Type: "foo", Name: "bar"},
			},
			ConfigAction{
				RootModule,
				Action{Type: "baz", Name: "bar"},
			},
		},
		// different Module
		{
			ConfigAction{
				RootModule,
				Action{Type: "foo", Name: "bar"},
			},
			ConfigAction{
				Module{"mod"},
				Action{Type: "foo", Name: "bar"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}

// TestAbsActionUniqueKey
func TestAbsActionUniqueKey(t *testing.T) {
	actionAddr1 := Action{
		Type: "a",
		Name: "b1",
	}.Absolute(RootModuleInstance)
	actionAddr2 := Action{
		Type: "a",
		Name: "b2",
	}.Absolute(RootModuleInstance)
	actionAddr3 := Action{
		Type: "a",
		Name: "in_module",
	}.Absolute(RootModuleInstance.Child("boop", NoKey))

	tests := []struct {
		Receiver  AbsAction
		Other     UniqueKeyer
		WantEqual bool
	}{
		{
			actionAddr1,
			actionAddr1,
			true,
		},
		{
			actionAddr1,
			actionAddr2,
			false,
		},
		{
			actionAddr1,
			actionAddr3,
			false,
		},
		{
			actionAddr3,
			actionAddr3,
			true,
		},
		{
			actionAddr1,
			actionAddr1.Instance(NoKey),
			false, // no-key instance key is distinct from its resource even though they have the same String result
		},
		{
			actionAddr1,
			actionAddr1.Instance(IntKey(1)),
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s matches %T %s?", test.Receiver, test.Other, test.Other), func(t *testing.T) {
			rKey := test.Receiver.UniqueKey()
			oKey := test.Other.UniqueKey()

			gotEqual := rKey == oKey
			if gotEqual != test.WantEqual {
				t.Errorf(
					"wrong result\nreceiver: %s\nother:    %s (%T)\ngot:  %t\nwant: %t",
					test.Receiver, test.Other, test.Other,
					gotEqual, test.WantEqual,
				)
			}
		})
	}
}

func TestParseAbsActionInstance(t *testing.T) {
	tests := []struct {
		input     string
		want      AbsActionInstance
		expectErr bool
	}{
		{
			"",
			AbsActionInstance{},
			true,
		},
		{
			"action.example.foo",
			AbsActionInstance{
				Action: ActionInstance{
					Action: Action{
						Type: "example",
						Name: "foo",
					},
					Key: NoKey,
				},
				Module: RootModuleInstance,
			},
			false,
		},
		{
			"action.example.foo[0]",
			AbsActionInstance{
				Action: ActionInstance{
					Action: Action{
						Type: "example",
						Name: "foo",
					},
					Key: IntKey(0),
				},
				Module: RootModuleInstance,
			},
			false,
		},
		{
			"action.example.foo[\"bar\"]",
			AbsActionInstance{
				Action: ActionInstance{
					Action: Action{
						Type: "example",
						Name: "foo",
					},
					Key: StringKey("bar"),
				},
				Module: RootModuleInstance,
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("ParseAbsActionStr(%s)", test.input), func(t *testing.T) {
			got, gotDiags := ParseAbsActionInstanceStr(test.input)
			if gotDiags.HasErrors() != test.expectErr {
				if !test.expectErr {
					t.Fatalf("wrong results! Expected success, got error: %s\n", gotDiags.Err())
				} else {
					t.Fatal("wrong results! Expected error(s), got success!")
				}
			}
			if !got.Equal(test.want) {
				t.Fatalf("wrong result! Got %s, wanted %s", got.String(), test.want.String())
			}
		})
	}
}
