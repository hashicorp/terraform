package models

func NewEnvironment() *Environment {
	return &Environment{
		System:      make(map[string]interface{}),
		Application: make(map[string]interface{}),
		Environment: make(map[string]interface{}),
		Running:     make(map[string]interface{}),
		Staging:     make(map[string]interface{}),
	}
}

type Environment struct {
	System      map[string]interface{} `json:"system_env_json,omitempty"`
	Environment map[string]interface{} `json:"environment_json,omitempty"`
	Running     map[string]interface{} `json:"running_env_json,omitempty"`
	Staging     map[string]interface{} `json:"staging_env_json,omitempty"`
	Application map[string]interface{} `json:"application_env_json,omitempty"`
}
