package config

import (
	"testing"
)

func TestImportTreeHCL2Experiment(t *testing.T) {
	// Can only run this test if we're built with the experiment enabled.
	// Enable this test by passing the following option to "go test":
	//    -ldflags="-X github.com/hashicorp/terraform/config.enableHCL2Experiment=true"
	// See the comment associated with this flag variable for more information.
	if enableHCL2Experiment == "" {
		t.Skip("HCL2 experiment is not enabled")
	}

	t.Run("HCL not opted in", func(t *testing.T) {
		// .tf file without opt-in should use the old HCL parser
		imp, err := loadTree("testdata/hcl2-experiment-switch/not-opted-in.tf")
		if err != nil {
			t.Fatal(err)
		}

		tree, err := imp.ConfigTree()
		if err != nil {
			t.Fatalf("unexpected error loading not-opted-in.tf: %s", err)
		}

		cfg := tree.Config
		if got, want := len(cfg.Locals), 1; got != want {
			t.Fatalf("wrong number of locals %#v; want %#v", got, want)
		}
		if cfg.Locals[0].RawConfig.Raw == nil {
			// Having RawConfig.Raw indicates the old loader
			t.Fatal("local has no RawConfig.Raw")
		}
	})

	t.Run("HCL opted in", func(t *testing.T) {
		// .tf file with opt-in should use the new HCL2 parser
		imp, err := loadTree("testdata/hcl2-experiment-switch/opted-in.tf")
		if err != nil {
			t.Fatal(err)
		}

		tree, err := imp.ConfigTree()
		if err != nil {
			t.Fatalf("unexpected error loading opted-in.tf: %s", err)
		}

		cfg := tree.Config
		if got, want := len(cfg.Locals), 1; got != want {
			t.Fatalf("wrong number of locals %#v; want %#v", got, want)
		}
		if cfg.Locals[0].RawConfig.Body == nil {
			// Having RawConfig.Body indicates the new loader
			t.Fatal("local has no RawConfig.Body")
		}
	})

	t.Run("JSON ineligible", func(t *testing.T) {
		// .tf.json file should always use the old HCL parser
		imp, err := loadTree("testdata/hcl2-experiment-switch/not-eligible.tf.json")
		if err != nil {
			t.Fatal(err)
		}

		tree, err := imp.ConfigTree()
		if err != nil {
			t.Fatalf("unexpected error loading not-eligible.tf.json: %s", err)
		}

		cfg := tree.Config
		if got, want := len(cfg.Locals), 1; got != want {
			t.Fatalf("wrong number of locals %#v; want %#v", got, want)
		}
		if cfg.Locals[0].RawConfig.Raw == nil {
			// Having RawConfig.Raw indicates the old loader
			t.Fatal("local has no RawConfig.Raw")
		}
	})
}
