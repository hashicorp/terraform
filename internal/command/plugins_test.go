package command

import (
	"os"
	"reflect"
	"testing"
)

func TestPluginPath(t *testing.T) {
	td := testTempDir(t)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	pluginPath := []string{"a", "b", "c"}

	m := Meta{}
	if err := m.storePluginPath(pluginPath); err != nil {
		t.Fatal(err)
	}

	restoredPath, err := m.loadPluginPath()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(pluginPath, restoredPath) {
		t.Fatalf("expected plugin path %#v, got %#v", pluginPath, restoredPath)
	}
}
