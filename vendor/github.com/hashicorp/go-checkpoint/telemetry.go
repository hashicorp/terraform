package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	uuid "github.com/hashicorp/go-uuid"
)

// ReportParams are the parameters for configuring a telemetry report.
type ReportParams struct {
	// Signature is some random signature that should be stored and used
	// as a cookie-like value. This ensures that alerts aren't repeated.
	// If the signature is changed, repeat alerts may be sent down. The
	// signature should NOT be anything identifiable to a user (such as
	// a MAC address). It should be random.
	//
	// If SignatureFile is given, then the signature will be read from this
	// file. If the file doesn't exist, then a random signature will
	// automatically be generated and stored here. SignatureFile will be
	// ignored if Signature is given.
	Signature     string `json:"signature"`
	SignatureFile string `json:"-"`

	StartTime     time.Time   `json:"start_time"`
	EndTime       time.Time   `json:"end_time"`
	Arch          string      `json:"arch"`
	OS            string      `json:"os"`
	Payload       interface{} `json:"payload,omitempty"`
	Product       string      `json:"product"`
	RunID         string      `json:"run_id"`
	SchemaVersion string      `json:"schema_version"`
	Version       string      `json:"version"`
}

func (i *ReportParams) signature() string {
	signature := i.Signature
	if i.Signature == "" && i.SignatureFile != "" {
		var err error
		signature, err = checkSignature(i.SignatureFile)
		if err != nil {
			return ""
		}
	}
	return signature
}

// Report sends telemetry information to checkpoint
func Report(ctx context.Context, r *ReportParams) error {
	if disabled := os.Getenv("CHECKPOINT_DISABLE"); disabled != "" {
		return nil
	}

	req, err := ReportRequest(r)
	if err != nil {
		return err
	}

	client := cleanhttp.DefaultClient()
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return fmt.Errorf("Unknown status: %d", resp.StatusCode)
	}

	return nil
}

// ReportRequest creates a request object for making a report
func ReportRequest(r *ReportParams) (*http.Request, error) {
	// Populate some fields automatically if we can
	if r.RunID == "" {
		uuid, err := uuid.GenerateUUID()
		if err != nil {
			return nil, err
		}
		r.RunID = uuid
	}
	if r.Arch == "" {
		r.Arch = runtime.GOARCH
	}
	if r.OS == "" {
		r.OS = runtime.GOOS
	}
	if r.Signature == "" {
		r.Signature = r.signature()
	}

	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	u := &url.URL{
		Scheme: "https",
		Host:   "checkpoint-api.hashicorp.com",
		Path:   fmt.Sprintf("/v1/telemetry/%s", r.Product),
	}

	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "HashiCorp/go-checkpoint")

	return req, nil
}
