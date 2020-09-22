package windowsbase

// FolderID is a representation of a known folder id UUID
type FolderID struct {
	a uint32
	b uint16
	c uint16
	d [8]byte
}

var (
	// RoamingAppDataID is the FolderID for the roaming application data folder
	RoamingAppDataID = &FolderID{0x3EB685DB, 0x65F9, 0x4CF6, [...]byte{0xA0, 0x3A, 0xE3, 0xEF, 0x65, 0x72, 0x9F, 0x3D}}

	// LocalAppDataID is the FolderID for the local application data folder
	LocalAppDataID = &FolderID{0xF1B32785, 0x6FBA, 0x4FCF, [...]byte{0x9D, 0x55, 0x7B, 0x8E, 0x7F, 0x15, 0x70, 0x91}}
)

// KnownFolderDir returns the absolute path for the given known folder id, or
// returns an error if that is not possible.
func KnownFolderDir(id *FolderID) (string, error) {
	return knownFolderDir(id)
}

// RoamingAppDataDir returns the absolute path for the current user's roaming
// application data directory.
func RoamingAppDataDir() (string, error) {
	return KnownFolderDir(RoamingAppDataID)
}

// LocalAppDataDir returns the absolute path for the current user's local
// application data directory.
func LocalAppDataDir() (string, error) {
	return KnownFolderDir(LocalAppDataID)
}
