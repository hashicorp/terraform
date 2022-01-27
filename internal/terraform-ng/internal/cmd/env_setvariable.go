package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/spf13/cobra"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/terraform-ng/internal/localenv"
)

var envSetvariableCmd = &cobra.Command{
	Use:   "set-variable ENVIRONMENT NAME <VALUE | --remove>",
	Short: "Set an input variable value for an environment.",
	Long: `Set an input variable value for an environment.
`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("not enough arguments")
		}
		if envSetvariableOpts.Remove {
			if len(args) != 2 {
				return fmt.Errorf("can't set value while using --remove")
			}
		} else {
			if len(args) != 3 {
				return fmt.Errorf("missing required variable value")
			}
		}
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		// TEMP: We only support local environments for the moment, while
		// we're just stubbing.
		env := args[0]
		if !localenv.ValidEnvironmentFilename(env) {
			cmd.PrintErrln("This stub command currently supports only local environment locations, given as a filename with a .tfenv.hcl suffix.")
			os.Exit(1)
		}

		def, err := localenv.OpenDefinitionFile(env)
		if err != nil {
			cmd.PrintErrf("Cannot open local environment %q: %s\n\n", env, err)
			os.Exit(1)
		}

		varName := args[1]
		if !hclsyntax.ValidIdentifier(varName) {
			cmd.PrintErrf("Invalid variable name\n\n", env, err)
			os.Exit(1)
		}

		var val cty.Value
		if envSetvariableOpts.JSON {
			src := []byte(args[2])
			ty, err := ctyjson.ImpliedType(src)
			if err != nil {
				cmd.PrintErrf("Invalid JSON syntax for value: %s\n\n", err)
				os.Exit(1)
			}
			val, err = ctyjson.Unmarshal(src, ty)
			if err != nil {
				cmd.PrintErrf("Invalid JSON syntax for value: %s\n\n", err)
				os.Exit(1)
			}
		} else {
			// Maybe in a real version of this we'd try to be a bit smarter
			// here by asking the environment what variables it's declared
			// and what type constraints they have, and then we could catch
			// if the given value isn't valid for the given type.
			val = cty.StringVal(args[2])
		}

		def.SetVariable(varName, val)
		err = def.Save()
		if err != nil {
			cmd.PrintErrf("Failed to update %q: %s\n\n", def.Filename(), err)
			os.Exit(1)
		}
	},
}

var envSetvariableOpts = struct {
	Remove bool
	JSON   bool
}{}

func init() {
	envSetvariableCmd.Flags().BoolVar(&envSetvariableOpts.Remove, "remove", false, "remove an existing definition")
	envSetvariableCmd.Flags().BoolVar(&envSetvariableOpts.JSON, "json", false, "specify value as JSON, rather than raw string")
	envCmd.AddCommand(envSetvariableCmd)
}
