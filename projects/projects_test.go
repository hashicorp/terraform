package projects

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/addrs"
)

func TestFindProjectAllWorkspaceAddrs(t *testing.T) {
	project, diags := FindProject("testdata/simpleproject")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err().Error())
	}

	got := project.AllWorkspaceAddrs()
	want := []addrs.ProjectWorkspace{
		addrs.ProjectWorkspace{
			Rel: addrs.ProjectWorkspaceCurrent, Name: "admin",
		},
		addrs.ProjectWorkspace{
			Rel: addrs.ProjectWorkspaceCurrent, Name: "monitoring", Key: addrs.StringKey("PROD"),
		},
		addrs.ProjectWorkspace{
			Rel: addrs.ProjectWorkspaceCurrent, Name: "monitoring", Key: addrs.StringKey("STAGE"),
		},
		addrs.ProjectWorkspace{
			Rel: addrs.ProjectWorkspaceCurrent, Name: "network", Key: addrs.StringKey("PROD"),
		},
		addrs.ProjectWorkspace{
			Rel: addrs.ProjectWorkspaceCurrent, Name: "network", Key: addrs.StringKey("STAGE"),
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestProjectWorkspaceConfigDependencies(t *testing.T) {
	project, diags := FindProject("testdata/withdependencies")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err().Error())
	}

	addrsCmp := cmp.Comparer(func(a, b addrs.ProjectWorkspaceConfig) bool {
		return a == b
	})

	t.Run("admin", func(t *testing.T) {
		// Upstream workspaces do not have dependencies that are visible
		// from this perspective. (We'd need to consult the upstream project's
		// configuration instead.)
		got := project.WorkspaceConfigDependencies(addrs.ProjectWorkspaceConfig{
			Rel:  addrs.ProjectWorkspaceUpstream,
			Name: "admin",
		})
		var want []addrs.ProjectWorkspaceConfig
		if diff := cmp.Diff(want, got, addrsCmp); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("network", func(t *testing.T) {
		got := project.WorkspaceConfigDependencies(addrs.ProjectWorkspaceConfig{
			Rel:  addrs.ProjectWorkspaceCurrent,
			Name: "network",
		})
		want := []addrs.ProjectWorkspaceConfig{
			{Rel: addrs.ProjectWorkspaceUpstream, Name: "admin"},
		}
		if diff := cmp.Diff(want, got, addrsCmp); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("monitoring", func(t *testing.T) {
		got := project.WorkspaceConfigDependencies(addrs.ProjectWorkspaceConfig{
			Rel:  addrs.ProjectWorkspaceCurrent,
			Name: "monitoring",
		})
		want := []addrs.ProjectWorkspaceConfig{
			{Rel: addrs.ProjectWorkspaceCurrent, Name: "network"},
			{Rel: addrs.ProjectWorkspaceUpstream, Name: "admin"},
		}
		if diff := cmp.Diff(want, got, addrsCmp); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("dns", func(t *testing.T) {
		got := project.WorkspaceConfigDependencies(addrs.ProjectWorkspaceConfig{
			Rel:  addrs.ProjectWorkspaceCurrent,
			Name: "dns",
		})
		want := []addrs.ProjectWorkspaceConfig{
			{Rel: addrs.ProjectWorkspaceCurrent, Name: "monitoring"},
			{Rel: addrs.ProjectWorkspaceCurrent, Name: "network"},
		}
		if diff := cmp.Diff(want, got, addrsCmp); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
}

func TestProjectWorkspaceDependencies(t *testing.T) {
	project, diags := FindProject("testdata/withdependencies")
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err().Error())
	}

	addrsCmp := cmp.Comparer(func(a, b addrs.ProjectWorkspaceConfig) bool {
		return a == b
	})

	t.Run("admin", func(t *testing.T) {
		// Upstream workspaces do not have dependencies that are visible
		// from this perspective. (We'd need to consult the upstream project's
		// configuration instead.)
		got := project.WorkspaceDependencies(addrs.ProjectWorkspace{
			Rel:  addrs.ProjectWorkspaceUpstream,
			Name: "admin",
		})
		var want []addrs.ProjectWorkspace
		if diff := cmp.Diff(want, got, addrsCmp); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("network", func(t *testing.T) {
		t.Skip("FIXME: upstream workspaces not currently included")
		got := project.WorkspaceDependencies(addrs.ProjectWorkspace{
			Rel:  addrs.ProjectWorkspaceCurrent,
			Name: "network",
			Key:  addrs.StringKey("PROD"),
		})
		want := []addrs.ProjectWorkspace{
			{Rel: addrs.ProjectWorkspaceUpstream, Name: "admin"},
		}
		if diff := cmp.Diff(want, got, addrsCmp); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("monitoring", func(t *testing.T) {
		t.Skip("FIXME: upstream workspaces not currently included")
		got := project.WorkspaceDependencies(addrs.ProjectWorkspace{
			Rel:  addrs.ProjectWorkspaceCurrent,
			Name: "monitoring",
			Key:  addrs.StringKey("PROD"),
		})
		want := []addrs.ProjectWorkspace{
			{Rel: addrs.ProjectWorkspaceCurrent, Name: "network", Key: addrs.StringKey("PROD")},
			{Rel: addrs.ProjectWorkspaceCurrent, Name: "network", Key: addrs.StringKey("STAGE")},
			{Rel: addrs.ProjectWorkspaceUpstream, Name: "admin"},
		}
		if diff := cmp.Diff(want, got, addrsCmp); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("dns", func(t *testing.T) {
		got := project.WorkspaceDependencies(addrs.ProjectWorkspace{
			Rel:  addrs.ProjectWorkspaceCurrent,
			Name: "dns",
			Key:  addrs.StringKey("PROD"),
		})
		want := []addrs.ProjectWorkspace{
			{Rel: addrs.ProjectWorkspaceCurrent, Name: "monitoring", Key: addrs.StringKey("PROD")},
			{Rel: addrs.ProjectWorkspaceCurrent, Name: "monitoring", Key: addrs.StringKey("STAGE")},
			{Rel: addrs.ProjectWorkspaceCurrent, Name: "network", Key: addrs.StringKey("PROD")},
			{Rel: addrs.ProjectWorkspaceCurrent, Name: "network", Key: addrs.StringKey("STAGE")},
		}
		if diff := cmp.Diff(want, got, addrsCmp); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
}
