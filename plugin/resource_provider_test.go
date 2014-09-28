package plugin

import (
	"testing"
)

func TestResourceProvider(t *testing.T) {
	c := NewClient(&ClientConfig{Cmd: helperProcess("resource-provider")})
	defer c.Kill()

	_, err := c.Client()
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}
