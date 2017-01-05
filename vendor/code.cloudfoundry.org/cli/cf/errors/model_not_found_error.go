package errors

import (
	. "code.cloudfoundry.org/cli/cf/i18n"
)

type ModelNotFoundError struct {
	ModelType string
	ModelName string
}

func NewModelNotFoundError(modelType, name string) error {
	return &ModelNotFoundError{
		ModelType: modelType,
		ModelName: name,
	}
}

func (err *ModelNotFoundError) Error() string {
	return err.ModelType + " " + err.ModelName + T(" not found")
}
