// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/hashicorp/terraform/internal/states/statemgr"
)

// fakeLockBucket is an in-memory S3 bucket with the semantics S3 native locking relies on: a write
// with If-None-Match: * is rejected with 412 when the object exists, and a read of a missing object
// fails with 404 NoSuchKey.
type fakeLockBucket struct {
	mu      sync.Mutex
	objects map[string][]byte
	puts    int

	// putError, when set, answers a write instead of the conditional-write check.
	putError func(put int) (status int, code string)
	// beforeGet runs before each read is answered, to interleave another client's operations.
	beforeGet func(b *fakeLockBucket)
}

func (b *fakeLockBucket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/bucket/")
	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		b.mu.Lock()
		defer b.mu.Unlock()
		b.puts++
		if b.putError != nil {
			if status, code := b.putError(b.puts); status != 0 {
				writeS3Error(w, status, code)
				return
			}
		}
		if _, exists := b.objects[key]; exists && r.Header.Get("If-None-Match") == "*" {
			writeS3Error(w, http.StatusPreconditionFailed, "PreconditionFailed")
			return
		}
		b.objects[key] = body
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		if b.beforeGet != nil {
			b.beforeGet(b)
		}
		b.mu.Lock()
		body, exists := b.objects[key]
		b.mu.Unlock()
		if !exists {
			writeS3Error(w, http.StatusNotFound, "NoSuchKey")
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		w.Write(body)
	case http.MethodDelete:
		b.release(key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (b *fakeLockBucket) release(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.objects, key)
}

func writeS3Error(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>%s</Code><Message>test</Message></Error>`, code)
}

const fakeLockFile = "state.tflock"

// newFakeLockClient returns a client using S3 native locking against bucket, with the lock already
// held by another client.
func newFakeLockClient(t *testing.T, bucket *fakeLockBucket) (*RemoteClient, *statemgr.LockInfo) {
	t.Helper()
	holder := statemgr.NewLockInfo()
	holder.Path = fakeLockFile
	held, err := json.Marshal(holder)
	if err != nil {
		t.Fatal(err)
	}
	bucket.objects = map[string][]byte{fakeLockFile: held}

	server := httptest.NewServer(bucket)
	t.Cleanup(server.Close)
	return &RemoteClient{
		s3Client: s3.New(s3.Options{
			BaseEndpoint:     aws.String(server.URL),
			Region:           "us-east-1",
			UsePathStyle:     true,
			Credentials:      credentials.NewStaticCredentialsProvider("id", "secret", ""),
			RetryMaxAttempts: 1,
		}),
		bucketName:   "bucket",
		lockFilePath: fakeLockFile,
		useLockFile:  true,
	}, holder
}

func lockWithTimeout(client *RemoteClient, timeout time.Duration) (*statemgr.LockInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	info := statemgr.NewLockInfo()
	_, err := statemgr.LockWithContext(ctx, client, info)
	return info, err
}

// The holder releases the lock after our write is rejected but before we read who holds it.
func TestLockWithFile_lockReleasedWhileAcquiring(t *testing.T) {
	cases := map[string]func(put int) (int, string){
		"write rejected because the lock file exists": nil,
		"write rejected by a concurrent conditional write": func(put int) (int, string) {
			if put == 1 {
				return http.StatusConflict, "ConditionalRequestConflict"
			}
			return 0, ""
		},
	}
	for name, putError := range cases {
		t.Run(name, func(t *testing.T) {
			bucket := &fakeLockBucket{putError: putError}
			var once sync.Once
			bucket.beforeGet = func(b *fakeLockBucket) { once.Do(func() { b.release(fakeLockFile) }) }
			client, _ := newFakeLockClient(t, bucket)

			info, err := lockWithTimeout(client, 10*time.Second)
			if err != nil {
				t.Fatalf("lock not acquired after the holder released it: %v", err)
			}

			var stored statemgr.LockInfo
			if err := json.Unmarshal(bucket.objects[fakeLockFile], &stored); err != nil {
				t.Fatal(err)
			}
			if stored.ID != info.ID {
				t.Fatalf("lock file holds %q, want ours %q", stored.ID, info.ID)
			}
		})
	}
}

// A lock that stays held is waited on until the timeout, and reports its holder.
func TestLockWithFile_lockHeld(t *testing.T) {
	client, holder := newFakeLockClient(t, &fakeLockBucket{})

	_, err := lockWithTimeout(client, 2*time.Second)
	le, ok := err.(*statemgr.LockError)
	if !ok {
		t.Fatalf("expected a LockError, got %T: %v", err, err)
	}
	if le.Info == nil || le.Info.ID != holder.ID {
		t.Fatalf("expected the holder's lock info, got %+v", le.Info)
	}
}

// Failures other than a rejected conditional write fail at once rather than wait out the timeout.
func TestLockWithFile_accessDenied(t *testing.T) {
	bucket := &fakeLockBucket{putError: func(int) (int, string) { return http.StatusForbidden, "AccessDenied" }}
	bucket.beforeGet = func(b *fakeLockBucket) { b.release(fakeLockFile) }
	client, _ := newFakeLockClient(t, bucket)

	start := time.Now()
	_, err := lockWithTimeout(client, 30*time.Second)
	if _, ok := err.(*statemgr.LockError); !ok {
		t.Fatalf("expected a LockError, got %T: %v", err, err)
	}
	if bucket.puts != 1 || time.Since(start) > 5*time.Second {
		t.Fatalf("expected one write and an immediate failure, got %d writes in %s", bucket.puts, time.Since(start))
	}
}
