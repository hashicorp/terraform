package kubernetes

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/statemgr"
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

	cleanupK8sResources(t)
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackend(t *testing.T) {
	testACC(t)
	defer cleanupK8sResources(t)

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	// Test
	backend.TestBackendStates(t, b1)
}

func TestBackendLocks(t *testing.T) {
	testACC(t)
	defer cleanupK8sResources(t)

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

func TestBackendLocksSoak(t *testing.T) {
	testACC(t)
	defer cleanupK8sResources(t)

	clientCount := 100
	lockAttempts := 100

	lockers := []statemgr.Locker{}
	for i := 0; i < clientCount; i++ {
		b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
			"secret_suffix": secretSuffix,
		}))

		s, err := b.StateMgr(backend.DefaultStateName)
		if err != nil {
			t.Fatalf("Error creating state manager: %v", err)
		}

		lockers = append(lockers, s.(statemgr.Locker))
	}

	wg := sync.WaitGroup{}
	for i, l := range lockers {
		wg.Add(1)
		go func(locker statemgr.Locker, n int) {
			defer wg.Done()

			li := statemgr.NewLockInfo()
			li.Operation = "test"
			li.Who = fmt.Sprintf("client-%v", n)

			for i := 0; i < lockAttempts; i++ {
				id, err := locker.Lock(li)
				if err != nil {
					continue
				}

				// hold onto the lock for a little bit
				time.Sleep(time.Duration(rand.Intn(10)) * time.Microsecond)

				err = locker.Unlock(id)
				if err != nil {
					t.Errorf("failed to unlock: %v", err)
				}
			}
		}(l, i)
	}

	wg.Wait()
}

func cleanupK8sResources(t *testing.T) {
	ctx := context.Background()
	// Get a backend to use the k8s client
	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"secret_suffix": secretSuffix,
	}))

	b := b1.(*Backend)

	sClient, err := b.KubernetesSecretClient()
	if err != nil {
		t.Fatal(err)
	}

	// Delete secrets
	opts := metav1.ListOptions{LabelSelector: tfstateKey + "=true"}
	secrets, err := sClient.List(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}

	delProp := metav1.DeletePropagationBackground
	delOps := metav1.DeleteOptions{PropagationPolicy: &delProp}
	var errs []error

	for _, secret := range secrets.Items {
		labels := secret.GetLabels()
		key, ok := labels[tfstateSecretSuffixKey]
		if !ok {
			continue
		}

		if key == secretSuffix {
			err = sClient.Delete(ctx, secret.GetName(), delOps)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	leaseClient, err := b.KubernetesLeaseClient()
	if err != nil {
		t.Fatal(err)
	}

	// Delete leases
	leases, err := leaseClient.List(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, lease := range leases.Items {
		labels := lease.GetLabels()
		key, ok := labels[tfstateSecretSuffixKey]
		if !ok {
			continue
		}

		if key == secretSuffix {
			err = leaseClient.Delete(ctx, lease.GetName(), delOps)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		t.Fatal(errs)
	}
}
