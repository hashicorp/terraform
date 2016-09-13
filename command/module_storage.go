package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/mitchellh/cli"
)

// map to store all already downloaded URLs
var storedModules = make(map[string]string)

// uiModuleStorage implements module.Storage and is just a proxy to output
// to the UI any Get operations.
type uiModuleStorage struct {
	Storage getter.Storage
	Ui      cli.Ui
}

func (s *uiModuleStorage) Dir(key string) (string, bool, error) {
	return s.Storage.Dir(key)
}

func (s *uiModuleStorage) Get(key string, source string, update bool) error {
	updateStr := ""
	if update {
		updateStr = " (update)"
	}

	if strings.Hasprefix(source, "file") {
		s.Ui.Output(fmt.Sprintf("Get: %s%s", source, updateStr))
		return s.Storage.Get(key, source, update)
	} else if storedModules[source] == "" {
		storedModules[source] = key
		s.Ui.Output(fmt.Sprintf("Get: %s%s storemap %+v", source, updateStr, storedModules))
		return s.Storage.Get(key, source, update)
	} else {
		downloadedPath := storedModules[source]
		downloadedPathURL := fmtfileURL(downloadedPath)
		s.Ui.Output(fmt.Sprintf("Get: %s%s storemap %+v", downloadedPathURL, updateStr, storedModules))
		return s.Storage.Get(key, downloadedPathURL, update)
	}

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
