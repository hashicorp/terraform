package fingerprint

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// linkSpeed returns link speed in Mb/s, or 0 when unable to determine it.
func (f *NetworkFingerprint) linkSpeed(device string) int {
	command := fmt.Sprintf("Get-NetAdapter -IncludeHidden | Where name -eq '%s' | Select -ExpandProperty LinkSpeed", device)
	path := "powershell.exe"
	outBytes, err := exec.Command(path, command).Output()

	if err != nil {
		f.logger.Printf("[WARN] fingerprint.network: Error calling %s (%s): %v", path, command, err)
		return 0
	}

	output := strings.TrimSpace(string(outBytes))

	return f.parseLinkSpeed(output)
}

func (f *NetworkFingerprint) parseLinkSpeed(commandOutput string) int {
	args := strings.Split(commandOutput, " ")
	if len(args) != 2 {
		f.logger.Printf("[WARN] fingerprint.network: Couldn't split LinkSpeed (%s)", commandOutput)
		return 0
	}

	unit := strings.Replace(args[1], "\r\n", "", -1)
	value, err := strconv.Atoi(args[0])
	if err != nil {
		f.logger.Printf("[WARN] fingerprint.network: Unable to parse LinkSpeed value (%s)", commandOutput)
		return 0
	}

	switch unit {
	case "Mbps":
		return value
	case "Kbps":
		return value / 1000
	case "Gbps":
		return value * 1000
	case "bps":
		return value / 1000000
	}

	return 0
}
