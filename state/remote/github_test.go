// This test assume that you will provide a valid token in the
// GITHUB_TOKEN environment variable.
package remote

import "testing"

func TestGithubClient(t *testing.T) {

	client, err := githubFactory(map[string]string{
		"repository": "blort",
		"owner":      "foo",
		"state_path": "foo/state",
		"lock_path":  "path/to/lockfile/lock",
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	testClient(t, client)
}
