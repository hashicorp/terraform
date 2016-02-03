package azure

import "fmt"

// Error represents the body of an error response which is often returned
// by the Azure ARM API.
type Error struct {
	StatusCode int
	ErrorCode  string `mapstructure:"code"`
	Message    string `mapstructure:"message"`
}

// Error implements interface error on AzureError structures
func (e Error) Error() string {
	return fmt.Sprintf("%s (%d) - %s", e.ErrorCode, e.StatusCode, e.Message)
}
