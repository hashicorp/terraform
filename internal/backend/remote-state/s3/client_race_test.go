// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package s3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestRemoteClientGet_objectShrinksBetweenHeadAndDownload reproduces the race
// where the state object is overwritten by a smaller one between the HeadObject
// call that sizes the buffer and the download that fills it. The returned
// payload must contain exactly the downloaded bytes, with no trailing padding.
func TestRemoteClientGet_objectShrinksBetweenHeadAndDownload(t *testing.T) {
	const oldSize = manager_DefaultPartSize + 58
	newBody := makeStateJSON(t, newSizeTarget)

	var mu sync.Mutex
	current := makeStateJSON(t, oldSize)
	replaceAfterHead := true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		body := current
		mu.Unlock()

		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.Header().Set("ETag", strconv.Quote(strconv.Itoa(len(body))))
			w.WriteHeader(http.StatusOK)
			if replaceAfterHead {
				mu.Lock()
				current = newBody
				mu.Unlock()
			}
		case http.MethodGet:
			mu.Lock()
			body = current
			mu.Unlock()
			status := http.StatusOK
			if rng := r.Header.Get("Range"); rng != "" {
				start, end := parseRange(t, rng, len(body))
				w.Header().Set("Content-Range", "bytes "+strconv.Itoa(start)+"-"+strconv.Itoa(end)+"/"+strconv.Itoa(len(body)))
				body = body[start : end+1]
				status = http.StatusPartialContent
			}
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.WriteHeader(status)
			w.Write(body)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	s3Client := s3.New(s3.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(srv.URL),
		UsePathStyle: true,
		Credentials:  credentials.NewStaticCredentialsProvider("x", "x", ""),
	})

	c := &RemoteClient{
		s3Client:   s3Client,
		bucketName: "b",
		path:       "k",
	}

	payload, err := c.get(context.Background())
	if err != nil {
		t.Fatalf("get: %s", err)
	}
	if payload == nil {
		t.Fatal("expected a payload, got nil")
	}

	var doc map[string]any
	if err := json.Unmarshal(payload.Data, &doc); err != nil {
		t.Fatalf("payload is not valid JSON (trailing padding not trimmed): %s\ndata ends with: %q",
			err, tail(payload.Data))
	}
	if len(payload.Data) != len(newBody) {
		t.Fatalf("payload length = %d, want %d (downloaded object size)", len(payload.Data), len(newBody))
	}
}

const (
	manager_DefaultPartSize = 5 * 1024 * 1024
	newSizeTarget           = manager_DefaultPartSize
)

func makeStateJSON(t *testing.T, size int) []byte {
	t.Helper()
	doc := map[string]any{
		"version":           4,
		"terraform_version": "1.14.7",
		"serial":            1,
		"lineage":           "repro",
		"outputs":           map[string]any{"o": map[string]any{"value": "", "type": "string"}},
		"resources":         []any{},
	}
	base, _ := json.Marshal(doc)
	pad := size - len(base)
	if pad < 0 {
		t.Fatalf("requested size %d smaller than base document %d", size, len(base))
	}
	doc["outputs"].(map[string]any)["o"].(map[string]any)["value"] = strings.Repeat("x", pad)
	body, _ := json.Marshal(doc)
	if len(body) != size {
		t.Fatalf("built document of %d bytes, want %d", len(body), size)
	}
	return body
}

func parseRange(t *testing.T, rng string, n int) (int, int) {
	t.Helper()
	spec := strings.TrimPrefix(rng, "bytes=")
	parts := strings.SplitN(spec, "-", 2)
	start, _ := strconv.Atoi(parts[0])
	end := n - 1
	if parts[1] != "" {
		end, _ = strconv.Atoi(parts[1])
	}
	if end > n-1 {
		end = n - 1
	}
	return start, end
}

func tail(b []byte) []byte {
	if len(b) > 16 {
		return b[len(b)-16:]
	}
	return b
}
