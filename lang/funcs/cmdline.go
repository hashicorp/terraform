package funcs

import (
	"github.com/apparentlymart/go-shquot/shquot"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// MakeCmdlineFunc constructs a command line building function based around
// the given quoter.
//
// A command line building function is one that takes a sequence of individual
// arguments, in a similar manner to the Unix-style "exec" functions, and
// returns a single string that can be evaluated by a particular shell
// to execute the intended command.
func MakeCmdlineFunc(quoter shquot.Q) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "cmd",
				Type: cty.String,
			},
		},
		VarParam: &function.Parameter{
			Name: "args",
			Type: cty.String,
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			strArgs := make([]string, len(args))
			for i, v := range args {
				strArgs[i] = v.AsString()
			}
			cmdline := quoter(strArgs)
			return cty.StringVal(cmdline), nil
		},
	})
}

// CmdlineUnixFunc is a function that takes an "argv"-style sequence of
// string arguments and produces a single string that can be evaluated by
// a Unix-style shell to execute the intended command line.
//
// The result is quoted to ensure that the shell will not interpret any
// metacharacters in the given strings and that the produced command line
// will run a real program rather than a shell alias.
var CmdlineUnixFunc = MakeCmdlineFunc(shquot.POSIXShell)

// CmdlineWindowsFunc is a function that takes an "argv"-style sequence of
// string arguments and produces a single string that can be evaluated by
// the Windows command line interpreter (cmd.exe) in combination with the
// Microsoft Visual C++ runtime library's command line parser to execute
// the intended command.
//
// Not all Windows programs parse their command line arguments using the
// Microsoft Visual C++ runtime library, so the result of this function may
// not be compatible with all Windows software.
var CmdlineWindowsFunc = MakeCmdlineFunc(shquot.WindowsCmdExe(shquot.WindowsArgv))
