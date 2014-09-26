package module

// Storage is an interface that knows how to lookup downloaded modules
// as well as download and update modules from their sources into the
// proper location.
type Storage interface {
	// Dir returns the directory on local disk where the modulue source
	// can be loaded from.
	Dir(string) (string, bool, error)

	// Get will download and optionally update the given module.
	Get(string, bool) error
}

func getStorage(s Storage, src string, mode GetMode) (string, bool, error) {
	// Get the module with the level specified if we were told to.
	if mode > GetModeNone {
		if err := s.Get(src, mode == GetModeUpdate); err != nil {
			return "", false, err
		}
	}

	// Get the directory where the module is.
	return s.Dir(src)
}
