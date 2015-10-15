package command

import (
	"fmt"

	"github.com/hashicorp/go-getter"
	"github.com/mitchellh/cli"
)

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

	s.Ui.Output(fmt.Sprintf("Get: %s%s", source, updateStr))
	return s.Storage.Get(key, source, update)
}
