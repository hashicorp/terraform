package remote

import (
	"testing"
)

func TestArtifactoryClient_impl(t *testing.T) {
	var _ Client = new(ArtifactoryClient)
}

func TestArtifactoryFactory(t *testing.T) {
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	config := make(map[string]string)

	// Empty config is an error
	_, err := artifactoryFactory(config)
	if err == nil {
		t.Fatalf("Empty config should be error")
	}

	config["url"] = "http://artifactory.local:8081/artifactory"
	config["repo"] = "terraform-repo"
	config["subpath"] = "myproject"

	// For this test we'll provide the credentials as config. The
	// acceptance tests implicitly test passing credentials as
	// environment variables.
	config["username"] = "test"
	config["password"] = "testpass"

	client, err := artifactoryFactory(config)
	if err != nil {
		t.Fatalf("Error for valid config")
	}

	artifactoryClient := client.(*ArtifactoryClient)

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
}
