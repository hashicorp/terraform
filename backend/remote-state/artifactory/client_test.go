package artifactory

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/zclconf/go-cty/cty"
)

func TestArtifactoryClient_impl(t *testing.T) {
	var _ remote.Client = new(ArtifactoryClient)
}

func TestArtifactoryFactory(t *testing.T) {
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	config := make(map[string]cty.Value)
	config["url"] = cty.StringVal("http://artifactory.local:8081/artifactory")
	config["repo"] = cty.StringVal("terraform-repo")
	config["subpath"] = cty.StringVal("myproject")

	// For this test we'll provide the credentials as config. The
	// acceptance tests implicitly test passing credentials as
	// environment variables.
	config["username"] = cty.StringVal("test")
	config["password"] = cty.StringVal("testpass")

	config["lock_url"] = cty.StringVal("http://lock-artifactory.local:8081/artifactory")
	config["lock_repo"] = cty.StringVal("terraform-lock-repo")
	config["lock_subpath"] = cty.StringVal("lock-myproject")
	config["lock_username"] = cty.StringVal("lock-test")
	config["lock_password"] = cty.StringVal("lock-testpass")
	config["unlock_username"] = cty.StringVal("unlock-test")
	config["unlock_password"] = cty.StringVal("unlock-testpass")

	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", config))

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("Error for valid config: %s", err)
	}

	artifactoryClient := state.(*remote.State).Client.(*ArtifactoryClient)

	if artifactoryClient.nativeClient.Config.BaseURL != "http://artifactory.local:8081/artifactory" {
		t.Fatalf("Incorrect url was populated")
	}
	if artifactoryClient.nativeClient.Config.Username != "test" {
		t.Fatalf("Incorrect username was populated")
	}
	if artifactoryClient.nativeClient.Config.Password != "testpass" {
		t.Fatalf("Incorrect password was populated")
	}
	if artifactoryClient.repo != "terraform-repo" {
		t.Fatalf("Incorrect repo was populated")
	}
	if artifactoryClient.subpath != "myproject" {
		t.Fatalf("Incorrect subpath was populated")
	}
	if artifactoryClient.lockNativeClient.Config.BaseURL != "http://lock-artifactory.local:8081/artifactory" {
		t.Fatalf("Incorrect lock_url was populated")
	}
	if artifactoryClient.lockNativeClient.Config.Username != "lock-test" {
		t.Fatalf("Incorrect lock_username was populated")
	}
	if artifactoryClient.lockNativeClient.Config.Password != "lock-testpass" {
		t.Fatalf("Incorrect lock_password was populated")
	}
	if artifactoryClient.unlockNativeClient.Config.BaseURL != "http://lock-artifactory.local:8081/artifactory" {
		t.Fatalf("Incorrect lock_url was populated")
	}
	if artifactoryClient.unlockNativeClient.Config.Username != "unlock-test" {
		t.Fatalf("Incorrect unlock_username was populated")
	}
	if artifactoryClient.unlockNativeClient.Config.Password != "unlock-testpass" {
		t.Fatalf("Incorrect unlock_password was populated")
	}
	if artifactoryClient.lockRepo != "terraform-lock-repo" {
		t.Fatalf("Incorrect lock_repo was populated")
	}
	if artifactoryClient.lockSubpath != "lock-myproject" {
		t.Fatalf("Incorrect lock_subpath was populated")
	}
}
