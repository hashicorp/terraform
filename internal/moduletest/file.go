package moduletest

import "github.com/hashicorp/terraform/internal/configs"

type File struct {
	Config *configs.TestFile

	Name   string
	Status Status

	Runs []*Run
}
