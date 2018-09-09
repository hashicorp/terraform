package remote

import (
	"fmt"
	"os"
	"strings"
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

	testArtifactoryVar(t, "username")
	testArtifactoryVar(t, "password")
	testArtifactoryVar(t, "url")
	testArtifactoryVar(t, "repo")
	testArtifactoryVar(t, "subpath")

	os.Setenv("ARTIFACTORY_USERNAME", "test")
	os.Setenv("ARTIFACTORY_PASSWORD", "testpass")
	os.Setenv("ARTIFACTORY_URL", "http://artifactory.local:8081/artifactory")
	os.Setenv("ARTIFACTORY_REPO", "terraform-repo")
	os.Setenv("ARTIFACTORY_SUBPATH", "myproject")

	// Clean up so that information about the test isn't leaked to other tests
	// through the environment.
	defer func() {
		os.Unsetenv("ARTIFACTORY_USERNAME")
		os.Unsetenv("ARTIFACTORY_PASSWORD")
		os.Unsetenv("ARTIFACTORY_URL")
		os.Unsetenv("ARTIFACTORY_REPO")
		os.Unsetenv("ARTIFACTORY_SUBPATH")
	}()

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

func testArtifactoryVar(t *testing.T, vr string) {
	envvr := fmt.Sprintf("ARTIFACTORY_%v", strings.ToUpper(vr))

	config := make(map[string]string)

	// This config only needs to be valid as far as actually having values.
	config["url"] = "foo"
	config["repo"] = "foo"
	config["subpath"] = "foo"
	config["username"] = "foo"
	config["password"] = "foo"
	delete(config, vr)

	errmsg := fmt.Sprintf(
		"missing '%v' configuration or %v environment variable",
		vr,
		envvr,
	)

	_, err := artifactoryFactory(config)
	if err.Error() != errmsg {
		t.Fatalf("missing %v should be error", envvr)
	}
}
