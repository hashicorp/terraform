package remote

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestEtcdClient_impl(t *testing.T) {
	var _ Client = new(EtcdClient)
}

func TestEtcdClient(t *testing.T) {
	endpoint := os.Getenv("ETCD_ENDPOINT")
	if endpoint == "" {
		t.Skipf("skipping; ETCD_ENDPOINT must be set")
	}

	config := map[string]string{
		"endpoints": endpoint,
		"path":      fmt.Sprintf("tf-unit/%s", time.Now().String()),
	}

	if username := os.Getenv("ETCD_USERNAME"); username != "" {
		config["username"] = username
	}
	if password := os.Getenv("ETCD_PASSWORD"); password != "" {
		config["password"] = password
	}

	client, err := etcdFactory(config)
	if err != nil {
		t.Fatalf("Error for valid config: %s", err)
	}

	testClient(t, client)
}
