package winrm

import (
	"encoding/base64"
	"fmt"
)

// Wraps a PowerShell script and prepares it for execution by the winrm client
func Powershell(psCmd string) string {
	// 2 byte chars to make PowerShell happy
	wideCmd := ""
	for _, b := range []byte(psCmd) {
		wideCmd += string(b) + "\x00"
	}

	// Base64 encode the command
	input := []uint8(wideCmd)
	encodedCmd := base64.StdEncoding.EncodeToString(input)

	// Create the powershell.exe command line to execute the script
	return fmt.Sprintf("powershell.exe -EncodedCommand %s", encodedCmd)
}
