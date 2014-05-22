package config

// Config is the configuration that comes from loading a collection
// of Terraform templates.
type Config struct {
	Variables map[string]Variable
	Resources []Resource
}

type Resource struct {
	Name   string
	Type   string
	Config map[string]interface{}
}

type Variable struct {
	Default     string
	Description string
}
