package kubernetes

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/states/statemgr"
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

	clientCount := 1000
	lockCount := 0

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
		go func(locker statemgr.Locker, i int) {
			r := rand.Intn(10)
			time.Sleep(time.Duration(r) * time.Microsecond)
			li := state.NewLockInfo()
			li.Operation = "test"
			li.Who = fmt.Sprintf("client-%v", i)
			_, err := locker.Lock(li)
			if err == nil {
				t.Logf("[INFO] Client %v got the lock\r\n", i)
				lockCount++
			}
			wg.Done()
		}(l, i)
	}

	wg.Wait()

	if lockCount > 1 {
		t.Fatalf("multiple backend clients were able to acquire a lock, count: %v", lockCount)
	}

	if lockCount == 0 {
		t.Fatal("no clients were able to acquire a lock")
	}
}

func cleanupK8sResources(t *testing.T) {
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

	leaseClient, err := b.KubernetesLeaseClient()
	if err != nil {
		t.Fatal(err)
	}

	// Delete leases
	leases, err := leaseClient.List(opts)
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
			err = leaseClient.Delete(lease.GetName(), delOps)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		t.Fatal(errs)
	}
}
