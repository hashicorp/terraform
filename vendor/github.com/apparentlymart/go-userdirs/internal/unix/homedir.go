package unix

import (
	"bytes"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Home returns the home directory for the current process, with the following
// preference order:
//
//     - The value of the HOME environment variable, if it is set and contains
//       an absolute path.
//     - The home directory indicated in the return value of the "Current"
//       function in the os/user standard library package, which has
//       platform-specific behavior, if it contains an absolute path.
//     - If neither of the above yields an absolute path, the string "/".
//
// In practice, POSIX requires the HOME environment variable to be set, so on
// any reasonable system it is that which will be selected. The other
// permutations are fallback behavior for less reasonable systems.
//
// XDG does not permit applications to write directly into the home directory.
// Instead, the paths returned by other functions in this package are
// potentially derived from the home path, if their explicit environment
// variables are not set.
func Home() string {
	if homeDir := os.Getenv("HOME"); homeDir != "" {
		if filepath.IsAbs(homeDir) {
			return homeDir
		}
	}

	user, err := user.Current()
	if err == nil {
		if homeDir := user.HomeDir; homeDir != "" {
			if filepath.IsAbs(homeDir) {
				return homeDir
			}
		}
	}

	if maybe := desperateFallback(); maybe != "" {
		return maybe
	}

	// Fallback behavior mimics a common choice in other software.
	return "/"
}

func desperateFallback() string {
	// This function implements some rather-nasty fallback behavior via some
	// platform-specific shell commands. This should always be a last resort,
	// but particulary when we are working not in CGo mode this path can help
	// us on platforms where the pure Go user.Current() stub's behavior isn't
	// appropriate for some more unusual Unix platforms, like Mac OS X.
	//
	// The existence and behavior of these commands is not an OS API contract,
	// so we run them in a best-effort way and just move on and try something
	// else if they fail.

	switch runtime.GOOS {
	case "darwin":
		var stdout bytes.Buffer
		cmd := exec.Command("sh", "-c", `dscl -q . -read /Users/"$(whoami)" NFSHomeDirectory | sed 's/^[^ ]*: //'`)
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			if result := strings.TrimSpace(stdout.String()); filepath.IsAbs(result) {
				return result
			}
		}
		return ""
	case "linux":
		var stdout bytes.Buffer
		cmd := exec.Command("getent", "passwd", strconv.Itoa(os.Getuid()))
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			if passwd := strings.TrimSpace(stdout.String()); passwd != "" {
				// username:password:uid:gid:gecos:home:shell
				passwdParts := strings.SplitN(passwd, ":", 7)
				if len(passwdParts) > 5 {
					if result := passwdParts[5]; filepath.IsAbs(result) {
						return result
					}
				}
			}
		}
		return ""
	default:
		return ""
	}
}
