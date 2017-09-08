package etcd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/hashicorp/terraform/backend"
)

var (
	etcdv3Endpoints = strings.Split(os.Getenv("TF_ETCDV3_ENDPOINTS"), ",")
)

const (
	keyPrefix = "tf-unit"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func prepareEtcdv3(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_ETCDV3_TEST") == ""
	if skip {
		t.Log("etcd server tests require setting TF_ACC or TF_ETCDV3_TEST")
		t.Skip()
	}

	client, err := etcdv3.New(etcdv3.Config{
		Endpoints: etcdv3Endpoints,
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.KV.Delete(context.TODO(), keyPrefix, etcdv3.WithPrefix())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Cleaned up %d keys.", res.Deleted)
}

func TestBackend(t *testing.T) {
	prepareEtcdv3(t)

	prefix := fmt.Sprintf("%s/%s/", keyPrefix, time.Now().Format(time.RFC3339))

	// Get the backend. We need two to test locking.
	b1 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"endpoints": etcdv3Endpoints,
		"prefix":    prefix,
	})

	b2 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"endpoints": etcdv3Endpoints,
		"prefix":    prefix,
	})

	// Test
	backend.TestBackend(t, b1, b2)
}

func TestBackend_lockDisabled(t *testing.T) {
	prepareEtcdv3(t)

	prefix := fmt.Sprintf("%s/%s/", keyPrefix, time.Now().Format(time.RFC3339))

	// Get the backend. We need two to test locking.
	b1 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"endpoints": etcdv3Endpoints,
		"prefix":    prefix,
		"lock":      false,
	})

	b2 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"endpoints": etcdv3Endpoints,
		"prefix":    prefix + "/" + "different", // Diff so locking test would fail if it was locking
		"lock":      false,
	})

	// Test
	backend.TestBackend(t, b1, b2)
}
