package planfile

import "github.com/hashicorp/terraform/internal/cloud/cloudplan"

// WrappedPlanFile is a sum type that represents a saved plan, loaded from a
// file path passed on the command line. If the specified file was a thick local
// plan file, the Local field will be populated; if it was a bookmark for a
// remote cloud plan, the Cloud field will be populated. In both cases, the
// other field is expected to be nil. Finally, the outer struct is also expected
// to be used as a pointer, so that a nil value can represent the absence of any
// plan file.
type WrappedPlanFile struct {
	Local *Reader
	Cloud *cloudplan.SavedPlanBookmark
}

func (w *WrappedPlanFile) IsLocal() bool {
	return w != nil && w.Local != nil
}

func (w *WrappedPlanFile) IsCloud() bool {
	return w != nil && w.Cloud != nil
}

// OpenWrapped loads a local or cloud plan file from a specified file path, or
// returns an error if the file doesn't seem to be a plan file of either kind.
// Most consumers should use this and switch behaviors based on the kind of plan
// they expected, rather than directly using Open.
func OpenWrapped(filename string) (*WrappedPlanFile, error) {
	// First, try to load it as a local planfile.
	local, localErr := Open(filename)
	if localErr == nil {
		return &WrappedPlanFile{Local: local}, nil
	}
	// Then, try to load it as a cloud plan.
	cloud, cloudErr := cloudplan.LoadSavedPlanBookmark(filename)
	if cloudErr == nil {
		return &WrappedPlanFile{Cloud: &cloud}, nil
	}
	// If neither worked, return the error from trying to handle it as a local
	// planfile, since that might have more context. Cloud plans are an opaque
	// format, so we don't care to give any advice about how to fix an internal
	// problem in one.
	return nil, localErr
}
