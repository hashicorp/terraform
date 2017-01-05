package errors

import (
	"fmt"

	. "code.cloudfoundry.org/cli/cf/i18n"
)

type ModelAlreadyExistsError struct {
	ModelType string
	ModelName string
}

func NewModelAlreadyExistsError(modelType, name string) *ModelAlreadyExistsError {
	return &ModelAlreadyExistsError{
		ModelType: modelType,
		ModelName: name,
	}
}

func (err *ModelAlreadyExistsError) Error() string {
	return fmt.Sprintf(T("{{.ModelType}} {{.ModelName}} already exists",
		map[string]interface{}{"ModelType": err.ModelType, "ModelName": err.ModelName}))
}
