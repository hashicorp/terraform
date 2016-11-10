// +build !linux,!windows

package fingerprint

// linkSpeed returns the default link speed
func (f *NetworkFingerprint) linkSpeed(device string) int {
	return 0
}
