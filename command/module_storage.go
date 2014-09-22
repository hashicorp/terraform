package command

import (
	"fmt"

	"github.com/hashicorp/terraform/config/module"
	"github.com/mitchellh/cli"
)

// uiModuleStorage implements module.Storage and is just a proxy to output
// to the UI any Get operations.
type uiModuleStorage struct {
	Storage module.Storage
	Ui      cli.Ui
}

func (s *uiModuleStorage) Dir(source string) (string, bool, error) {
	return s.Storage.Dir(source)
}

func (s *uiModuleStorage) Get(source string, update bool) error {
	updateStr := ""
	if update {
		updateStr = " (update)"
	}

	s.Ui.Output(fmt.Sprintf("Get: %s%s", source, updateStr))
	return s.Storage.Get(source, update)
}
