package models

type FeatureFlag struct {
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	ErrorMessage string `json:"error_message"`
}
