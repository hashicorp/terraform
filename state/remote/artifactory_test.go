package remote

import (
	"os"
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

func TestArtifactoryFactoryEnvironmentConfig(t *testing.T) {
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	config := make(map[string]string)

	_, err := artifactoryFactory(config)
	if err.Error() != "missing 'username' configuration or ARTIFACTORY_USERNAME environment variable" {
		t.Fatal("missing ARTIFACTORY_USERNAME should be error")
	}
	os.Setenv("ARTIFACTORY_USERNAME", "test")
	err = nil

	_, err = artifactoryFactory(config)
	if err.Error() != "missing 'password' configuration or ARTIFACTORY_PASSWORD environment variable" {
		t.Fatal("missing ARTIFACTORY_PASSWORD should be error")
	}
	os.Setenv("ARTIFACTORY_PASSWORD", "testpass")
	err = nil

	_, err = artifactoryFactory(config)
	if err.Error() != "missing 'url' configuration or ARTIFACTORY_URL environment variable" {
		t.Fatal("missing ARTIFACTORY_URL should be error")
	}
	os.Setenv("ARTIFACTORY_URL", "http://artifactory.local:8081/artifactory")
	err = nil

	_, err = artifactoryFactory(config)
	if err.Error() != "missing 'repo' configuration or ARTIFACTORY_REPO environment variable" {
		t.Fatal("missing ARTIFACTORY_REPO should be error")
	}
	os.Setenv("ARTIFACTORY_REPO", "terraform-repo")
	err = nil

	_, err = artifactoryFactory(config)
	if err.Error() != "missing 'subpath' configuration or ARTIFACTORY_SUBPATH environment variable" {
		t.Fatal("missing ARTIFACTORY_SUBPATH should be error")
	}
	os.Setenv("ARTIFACTORY_SUBPATH", "myproject")
	err = nil

	client, err := artifactoryFactory(config)
	if err != nil {
		t.Fatalf("Error for valid config: %v", err)
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
