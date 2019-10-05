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
