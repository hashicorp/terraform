package maas

import (
	"testing"

	"net/url"

	"launchpad.net/gomaasapi"
)

func TestMaasListAllNodes(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if _, err := maasListAllNodes(gomaasapi.NewMAAS(*authClient)); err == nil {
		t.Fail()
	}
}

func TestMaasGetSingleNode(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if _, err := maasGetSingleNode(gomaasapi.NewMAAS(*authClient), "system_id"); err == nil {
		t.Fail()
	}
}

func TestMaasAllocateNodes(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if _, err := maasAllocateNodes(gomaasapi.NewMAAS(*authClient), url.Values{}); err == nil {
		t.Fail()
	}
}

func TestMaasReleaseNode(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if err := maasReleaseNode(gomaasapi.NewMAAS(*authClient), "system_id"); err == nil {
		t.Fail()
	}
}

func TestToNodeInfo(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if _, err := toNodeInfo(gomaasapi.NewMAAS(*authClient)); err == nil {
		t.Fail()
	}
}

func TestGetNodeStatus(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if err := getNodeStatus(gomaasapi.NewMAAS(*authClient), "system_id"); err == nil {
		t.Fail()
	}
}

func TestGetSingleNode(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if _, err := getSingleNode(gomaasapi.NewMAAS(*authClient), "system_id"); err == nil {
		t.Fail()
	}
}

func TestGetAllNodes(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if _, err := getAllNodes(gomaasapi.NewMAAS(*authClient)); err == nil {
		t.Fail()
	}
}

func TestNodeDo(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if err := nodeDo(gomaasapi.NewMAAS(*authClient), "system_id", "node_action", url.Values{}); err == nil {
		t.Fail()
	}
}

func TestNodesAllocate(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if _, err := nodesAllocate(gomaasapi.NewMAAS(*authClient), url.Values{}); err == nil {
		t.Fail()
	}
}

func TestNodesRelease(t *testing.T) {
	authClient, err := gomaasapi.NewAuthenticatedClient("http://example.com/", "a:b:c", "1.0")
	if err != nil {
		t.Fail()
	}
	if err := nodeRelease(gomaasapi.NewMAAS(*authClient), "system_id"); err == nil {
		t.Fail()
	}
}
