package remote

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
)

func TestConsulClient_impl(t *testing.T) {
	var _ Client = new(ConsulClient)
}

func TestConsulClient(t *testing.T) {
	acctest.RemoteTestPrecheck(t)

	client, err := consulFactory(map[string]string{
		"address": "demo.consul.io:80",
		"path":    fmt.Sprintf("tf-unit/%s", time.Now().String()),
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	testClient(t, client)
}
