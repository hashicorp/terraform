package addrs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDebuggableDebugAncestorFrames(t *testing.T) {
	tests := []struct {
		Input Debuggable
		Want  []Debuggable
	}{
		{
			RootModuleInstance,
			nil,
		},
		{
			AbsModuleCall{
				Caller: RootModuleInstance,
				Call: ModuleCall{
					Name: "boop",
				},
			},
			[]Debuggable{
				RootModuleInstance,
			},
		},
		{
			RootModuleInstance.Child("boop", NoKey),
			[]Debuggable{
				RootModuleInstance,
				AbsModuleCall{
					Caller: RootModuleInstance,
					Call: ModuleCall{
						Name: "boop",
					},
				},
			},
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "bleep",
				Name: "bloop",
			}.Absolute(RootModuleInstance),
			[]Debuggable{
				RootModuleInstance,
			},
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "bleep",
				Name: "bloop",
			}.Instance(NoKey).Absolute(RootModuleInstance),
			[]Debuggable{
				RootModuleInstance,
				Resource{
					Mode: ManagedResourceMode,
					Type: "bleep",
					Name: "bloop",
				}.Absolute(RootModuleInstance),
			},
		},
		{
			Resource{
				Mode: ManagedResourceMode,
				Type: "bleep",
				Name: "bloop",
			}.Instance(NoKey).Absolute(RootModuleInstance.Child("beep", NoKey)),
			[]Debuggable{
				RootModuleInstance,
				RootModuleInstance.Child("beep", NoKey).Call(),
				RootModuleInstance.Child("beep", NoKey),
				Resource{
					Mode: ManagedResourceMode,
					Type: "bleep",
					Name: "bloop",
				}.Absolute(RootModuleInstance.Child("beep", NoKey)),
			},
		},
		{
			InputVariable{
				Name: "bloop",
			}.Absolute(RootModuleInstance),
			[]Debuggable{
				RootModuleInstance,
			},
		},
		{
			LocalValue{
				Name: "bloop",
			}.Absolute(RootModuleInstance),
			[]Debuggable{
				RootModuleInstance,
			},
		},
		{
			OutputValue{
				Name: "bloop",
			}.Absolute(RootModuleInstance),
			[]Debuggable{
				RootModuleInstance,
			},
		},
		{
			AbsProviderConfig{
				Module:   RootModule,
				Provider: NewDefaultProvider("bop"),
			},
			[]Debuggable{
				RootModuleInstance,
			},
		},
		{
			AbsProviderConfig{
				Module:   RootModule.Child("a"),
				Provider: NewDefaultProvider("bop"),
			},
			[]Debuggable{
				RootModuleInstance,
				RootModuleInstance.Child("a", NoKey).Call(),
				RootModuleInstance.Child("a", NoKey),
			},
		},
	}

	co := cmp.Options{
		cmpopts.IgnoreUnexported(
			AbsModuleCall{},
			ModuleCall{},
		),
	}

	for _, test := range tests {
		t.Run(test.Input.String(), func(t *testing.T) {
			got := test.Input.DebugAncestorFrames()
			if diff := cmp.Diff(test.Want, got, co...); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
