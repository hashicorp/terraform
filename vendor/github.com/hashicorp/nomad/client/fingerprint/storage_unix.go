// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package fingerprint

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// diskFree inspects the filesystem for path and returns the volume name and
// the total and free bytes available on the file system.
func (f *StorageFingerprint) diskFree(path string) (volume string, total, free uint64, err error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to determine absolute path for %s", path)
	}

	// Use -k to standardize the output values between darwin and linux
	var dfArgs string
	if runtime.GOOS == "linux" {
		// df on linux needs the -P option to prevent linebreaks on long filesystem paths
		dfArgs = "-kP"
	} else {
		dfArgs = "-k"
	}

	mountOutput, err := exec.Command("df", dfArgs, absPath).Output()
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to determine mount point for %s", absPath)
	}
	// Output looks something like:
	//	Filesystem 1024-blocks      Used Available Capacity   iused    ifree %iused  Mounted on
	//	/dev/disk1   487385240 423722532  63406708    87% 105994631 15851677   87%   /
	//	[0] volume [1] capacity [2] SKIP  [3] free
	lines := strings.Split(string(mountOutput), "\n")
	if len(lines) < 2 {
		return "", 0, 0, fmt.Errorf("failed to parse `df` output; expected at least 2 lines")
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return "", 0, 0, fmt.Errorf("failed to parse `df` output; expected at least 4 columns")
	}
	volume = fields[0]

	total, err = strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to parse storage.bytestotal size in kilobytes")
	}
	// convert to bytes
	total *= 1024

	free, err = strconv.ParseUint(fields[3], 10, 64)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to parse storage.bytesfree size in kilobytes")
	}
	// convert to bytes
	free *= 1024

	return volume, total, free, nil
}
