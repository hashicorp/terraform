package module

import (
	"fmt"
	"path/filepath"
)

// FileDetector implements Detector to detect file paths.
type FileDetector struct{}

func (d *FileDetector) Detect(src, pwd string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	// Make sure we're using "/" even on Windows. URLs are "/"-based.
	src = filepath.ToSlash(src)
	if !filepath.IsAbs(src) {
		if pwd == "" {
			return "", true, fmt.Errorf(
				"relative paths require a module with a pwd")
		}

		src = filepath.Join(pwd, src)
	}

	// Make sure that we don't start with "/" since we add that below
	if src[0] == '/' {
		src = src[1:]
	}

	return fmt.Sprintf("file:///%s", src), true, nil
}
