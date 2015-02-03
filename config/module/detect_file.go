package module

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// FileDetector implements Detector to detect file paths.
type FileDetector struct{}

func (d *FileDetector) Detect(src, pwd string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	if !filepath.IsAbs(src) {
		if pwd == "" {
			return "", true, fmt.Errorf(
				"relative paths require a module with a pwd")
		}

		src = filepath.Join(pwd, src)
	}
	return fmtFileURL(src), true, nil
}

func fmtFileURL(path string) string {
	if runtime.GOOS == "windows" {
		// Make sure we're using "/" on Windows. URLs are "/"-based.
		path = filepath.ToSlash(path)
		return fmt.Sprintf("file://%s", path)
	}

	// Make sure that we don't start with "/" since we add that below.
	if path[0] == '/' {
		path = path[1:]
	}
	return fmt.Sprintf("file:///%s", path)
}
