package getter

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sync"

	gg "github.com/hashicorp/go-getter"
	"github.com/hashicorp/nomad/client/driver/env"
	"github.com/hashicorp/nomad/nomad/structs"
)

var (
	// getters is the map of getters suitable for Nomad. It is initialized once
	// and the lock is used to guard access to it.
	getters map[string]gg.Getter
	lock    sync.Mutex

	// supported is the set of download schemes supported by Nomad
	supported = []string{"http", "https", "s3"}
)

// getClient returns a client that is suitable for Nomad downloading artifacts.
func getClient(src, dst string) *gg.Client {
	lock.Lock()
	defer lock.Unlock()

	// Return the pre-initialized client
	if getters == nil {
		getters = make(map[string]gg.Getter, len(supported))
		for _, getter := range supported {
			if impl, ok := gg.Getters[getter]; ok {
				getters[getter] = impl
			}
		}
	}

	return &gg.Client{
		Src:     src,
		Dst:     dst,
		Mode:    gg.ClientModeAny,
		Getters: getters,
	}
}

// getGetterUrl returns the go-getter URL to download the artifact.
func getGetterUrl(taskEnv *env.TaskEnvironment, artifact *structs.TaskArtifact) (string, error) {
	taskEnv.Build()
	u, err := url.Parse(taskEnv.ReplaceEnv(artifact.GetterSource))
	if err != nil {
		return "", fmt.Errorf("failed to parse source URL %q: %v", artifact.GetterSource, err)
	}

	// Build the url
	q := u.Query()
	for k, v := range artifact.GetterOptions {
		q.Add(k, taskEnv.ReplaceEnv(v))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// GetArtifact downloads an artifact into the specified task directory.
func GetArtifact(taskEnv *env.TaskEnvironment, artifact *structs.TaskArtifact, taskDir string) error {
	url, err := getGetterUrl(taskEnv, artifact)
	if err != nil {
		return err
	}

	// Download the artifact
	dest := filepath.Join(taskDir, artifact.RelativeDest)
	if err := getClient(url, dest).Get(); err != nil {
		return fmt.Errorf("GET error: %v", err)
	}

	return nil
}
