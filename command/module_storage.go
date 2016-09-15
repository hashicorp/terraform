package command

import (
	"fmt"
	"path/filepath"
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

	isFile := strings.HasPrefix(source, "file")
	if isFile {
		s.Ui.Output(fmt.Sprintf("Get: %s%s", source, updateStr))
		return s.Storage.Get(key, source, update)
	} else if storedModules[source] == "" {
		s.Ui.Output(fmt.Sprintf("Get: %s%s", source, updateStr))
		getResult := s.Storage.Get(key, source, update)
		getDir, _, err := s.Storage.Dir(key)
		if err == nil {
			storedModules[source] = getDir
		}
		return getResult
	} else {
		downloadedPath := storedModules[source]
		downloadedPathURL, _ := filepath.Abs(downloadedPath)
		s.Ui.Output(fmt.Sprintf("Get: %s%s", downloadedPathURL, updateStr))
		return s.Storage.Get(key, downloadedPathURL, update)
	}

}
