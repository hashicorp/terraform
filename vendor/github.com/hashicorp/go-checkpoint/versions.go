package checkpoint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

// VersionsParams are the parameters for a versions request.
type VersionsParams struct {
	// Service is used to lookup the correct service.
	Service string

	// Product is used to filter the version contraints.
	Product string

	// Force, if true, will force the check even if CHECKPOINT_DISABLE
	// is set. Within HashiCorp products, this is ONLY USED when the user
	// specifically requests it. This is never automatically done without
	// the user's consent.
	Force bool
}

// VersionsResponse is the response for a versions request.
type VersionsResponse struct {
	Service   string   `json:"service"`
	Product   string   `json:"product"`
	Minimum   string   `json:"minimum"`
	Maximum   string   `json:"maximum"`
	Excluding []string `json:"excluding"`
}

// Versions returns the version constrains for a given service and product.
func Versions(p *VersionsParams) (*VersionsResponse, error) {
	if disabled := os.Getenv("CHECKPOINT_DISABLE"); disabled != "" && !p.Force {
		return &VersionsResponse{}, nil
	}

	// Set a default timeout of 1 sec for the versions request (in milliseconds)
	timeout := 1000
	if _, err := strconv.Atoi(os.Getenv("CHECKPOINT_TIMEOUT")); err == nil {
		timeout, _ = strconv.Atoi(os.Getenv("CHECKPOINT_TIMEOUT"))
	}

	v := url.Values{}
	v.Set("product", p.Product)

	u := &url.URL{
		Scheme:   "https",
		Host:     "checkpoint-api.hashicorp.com",
		Path:     fmt.Sprintf("/v1/versions/%s", p.Service),
		RawQuery: v.Encode(),
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "HashiCorp/go-checkpoint")

	client := cleanhttp.DefaultClient()

	// We use a short timeout since checking for new versions is not critical
	// enough to block on if checkpoint is broken/slow.
	client.Timeout = time.Duration(timeout) * time.Millisecond

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Unknown status: %d", resp.StatusCode)
	}

	result := &VersionsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return nil, err
	}

	return result, nil
}
