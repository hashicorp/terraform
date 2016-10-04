package akamai

import (
    "encoding/json"
    "fmt"
)

// AkamaiError represents a non-successful HTTP response from the Akamai API.
type AkamaiError struct {
    Type   string `json:"type"`
    Title  string `json:"title"`
    Detail string `json:"detail"`
    Status int    `json:"status"`
}

func NewAkamaiError(body []byte) (*AkamaiError, error) {
    akamaiError := &AkamaiError{}

    if err := json.Unmarshal(body, &akamaiError); err != nil {
        return nil, err
    }

    return akamaiError, nil
}

func (err AkamaiError) Error() string {
    return fmt.Sprintf("%d %s\n%s", err.Status, err.Title, err.Detail)
}

