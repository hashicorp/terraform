package kubernetes

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/backend"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	secretSuffix = "test-state"
)

var namespace string

// verify that we are doing ACC tests or the k8s tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_K8S_TEST") == ""
	if skip {
		t.Log("k8s backend tests require setting TF_ACC or TF_K8S_TEST")
		t.Skip()
	}

	ns := os.Getenv("KUBE_NAMESPACE")

	if ns != "" {
		namespace = ns
	} else {
		namespace = "default"
	}

	cleanupK8sSecrets(t)
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackend(t *testing.T) {
	testACC(t)
	defer cleanupK8sSecrets(t)

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	// Test
	backend.TestBackendStates(t, b1)
}

func TestBackendLocks(t *testing.T) {
	testACC(t)
	defer cleanupK8sSecrets(t)

	// Get the backend. We need two to test locking.
	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	// Test
	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)
}

func cleanupK8sSecrets(t *testing.T) {
	// Get a backend to use the k8s client
	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	b := b1.(*Backend)

	sClient, err := b.KubernetesSecretClient()
	if err != nil {
		t.Fatal(err)
	}

	// Get state secrets based off the tfstateKey label
	opts := metav1.ListOptions{LabelSelector: tfstateKey + "=true"}
	secrets, err := sClient.List(opts)
	if err != nil {
		t.Fatal(err)
	}

	delProp := metav1.DeletePropagationBackground
	delOps := &metav1.DeleteOptions{PropagationPolicy: &delProp}
	var errs []error

	for _, secret := range secrets.Items {
		labels := secret.GetLabels()
		key, ok := labels[tfstateSecretSuffixKey]
		if !ok {
			continue
		}

		if key == secretSuffix {
			err = sClient.Delete(secret.GetName(), delOps)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		t.Fatal(errs)
	}
}
