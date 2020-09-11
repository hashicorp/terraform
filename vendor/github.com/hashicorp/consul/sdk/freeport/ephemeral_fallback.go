//+build !linux,!darwin

package freeport

func getEphemeralPortRange() (int, int, error) {
	return 0, 0, nil
}
