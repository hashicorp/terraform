package kubernetes

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/dynamic"
)

const (
	tfstateKey             = "tfstate"
	tfstateSecretSuffixKey = "tfstateSecretSuffix"
	tfstateWorkspaceKey    = "tfstateWorkspace"
	tfstateLockInfoKey     = "tfstateLockInfo"
	managedByKey           = "app.kubernetes.io/managed-by"
)

type RemoteClient struct {
	kubernetesSecretClient dynamic.ResourceInterface
	namespace              string
	labels                 map[string]string
	nameSuffix             string
	workspace              string
}

func (c *RemoteClient) Get() (payload *remote.Payload, err error) {
	secretName, err := c.createSecretName()
	if err != nil {
		return nil, err
	}
	secret, err := c.kubernetesSecretClient.Get(secretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	secretData := getSecretData(secret)

	stateRaw, ok := secretData[tfstateKey]
	if !ok {
		// The secret exists but there is no state in it
		return nil, nil
	}

	stateRawString := stateRaw.(string)

	state, err := uncompressState(stateRawString)
	if err != nil {
		return nil, err
	}

	md5 := md5.Sum(state)

	p := &remote.Payload{
		Data: state,
		MD5:  md5[:],
	}
	return p, nil
}

func (c *RemoteClient) Put(data []byte) error {
	secretName, err := c.createSecretName()
	if err != nil {
		return err
	}

	payload, err := compressState(data)
	if err != nil {
		return err
	}

	secret, err := c.getSecret(secretName)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("secret does not exist so lock is not held, secret name: %v err: %v", secretName, err)
	}

	lockInfo, err := c.getLockInfo(secret)
	if err != nil {
		return err
	}

	setState(secret, payload)

	secret, err = c.kubernetesSecretClient.Update(secret, metav1.UpdateOptions{})
	if err != nil {
		lockErr := &state.LockError{
			Info: lockInfo,
			Err:  fmt.Errorf("error updating the state: %v", err),
		}
		return lockErr
	}

	return err
}

// Delete the state secret
func (c *RemoteClient) Delete() error {
	secretName, err := c.createSecretName()
	if err != nil {
		return err
	}

	err = c.deleteSecret(secretName)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	secretName, err := c.createSecretName()
	if err != nil {
		return "", err
	}

	lockInfo := info.Marshal()

	secret, err := c.getSecret(secretName)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return "", err
		}

		// The secret doesn't exist yet, create it with the lock
		secretData := make(map[string][]byte)
		secretData[tfstateLockInfoKey] = lockInfo

		labels := map[string]string{
			tfstateKey:             "true",
			tfstateSecretSuffixKey: c.nameSuffix,
			tfstateWorkspaceKey:    c.workspace,
			managedByKey:           "terraform",
		}

		if len(c.labels) != 0 {
			for k, v := range c.labels {
				labels[k] = v
			}
		}

		secret = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": metav1.ObjectMeta{
					Name:        secretName,
					Namespace:   c.namespace,
					Labels:      labels,
					Annotations: map[string]string{"encoding": "gzip"},
				},
				"data": secretData,
			},
		}

		_, err = c.kubernetesSecretClient.Create(secret, metav1.CreateOptions{})
		if err != nil {
			if !k8serrors.IsAlreadyExists(err) {
				return "", err
			}
			// The secret was created between the get and create, grab it and keep going
			secret, err = c.getSecret(secretName)
			if err != nil {
				return "", err
			}
		} else {
			// No error on the create so the lock has been created successfully
			return info.ID, nil
		}
	}

	li, err := c.getLockInfo(secret)
	if err != nil {
		return "", err
	}

	if li != nil {
		// The lock already exists
		lockErr := &state.LockError{
			Info: li,
			Err:  errors.New("lock already exists"),
		}
		return "", lockErr
	}

	setLockInfo(secret, lockInfo)

	_, err = c.kubernetesSecretClient.Update(secret, metav1.UpdateOptions{})
	if err != nil {
		return "", err
	}

	return info.ID, err
}

func (c *RemoteClient) Unlock(id string) error {
	secretName, err := c.createSecretName()
	if err != nil {
		return err
	}

	secret, err := c.getSecret(secretName)
	if err != nil {
		// If the secret doesn't exist, there is nothing to unlock
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	lockInfo, err := c.getLockInfo(secret)
	if err != nil {
		return err
	}

	if lockInfo == nil {
		// The lock doesn't exist
		return nil
	}

	lockErr := &state.LockError{
		Info: lockInfo,
	}

	if lockInfo.ID != id {
		lockErr.Err = fmt.Errorf("lock id %q does not match existing lock", id)
		return lockErr
	}

	setLockInfo(secret, []byte{})

	_, err = c.kubernetesSecretClient.Update(secret, metav1.UpdateOptions{})
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	return nil
}

//getLockInfo takes a secret and attempts to read the lockInfo field.
func (c *RemoteClient) getLockInfo(secret *unstructured.Unstructured) (*state.LockInfo, error) {
	lockData, ok := getLockInfo(secret)
	if len(lockData) == 0 || !ok {
		return nil, nil
	}

	lockInfo := &state.LockInfo{}
	err := json.Unmarshal(lockData, lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

func (c *RemoteClient) getSecret(name string) (*unstructured.Unstructured, error) {
	return c.kubernetesSecretClient.Get(name, metav1.GetOptions{})
}

func (c *RemoteClient) deleteSecret(name string) error {
	secret, err := c.getSecret(name)
	if err != nil {
		return err
	}

	labels := secret.GetLabels()
	v, ok := labels[tfstateKey]
	if !ok || v != "true" {
		return fmt.Errorf("Secret does does not have %q label", tfstateKey)
	}

	delProp := metav1.DeletePropagationBackground
	delOps := &metav1.DeleteOptions{PropagationPolicy: &delProp}
	return c.kubernetesSecretClient.Delete(name, delOps)
}

func (c *RemoteClient) createSecretName() (string, error) {
	secretName := strings.Join([]string{tfstateKey, c.workspace, c.nameSuffix}, "-")

	errs := validation.IsDNS1123Subdomain(secretName)
	if len(errs) > 0 {
		k8sInfo := `
This is a requirement for Kubernetes secret names. 
The workspace name and key must adhere to Kubernetes naming conventions.`
		msg := fmt.Sprintf("the secret name %v is invalid, ", secretName)
		return "", errors.New(msg + strings.Join(errs, ",") + k8sInfo)
	}

	return secretName, nil
}

func compressState(data []byte) ([]byte, error) {
	b := new(bytes.Buffer)
	gz := gzip.NewWriter(b)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func uncompressState(data string) ([]byte, error) {
	decode, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	b := new(bytes.Buffer)
	gz, err := gzip.NewReader(bytes.NewReader(decode))
	if err != nil {
		return nil, err
	}
	b.ReadFrom(gz)
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func getSecretData(secret *unstructured.Unstructured) map[string]interface{} {
	return secret.Object["data"].(map[string]interface{})
}

func getLockInfo(secret *unstructured.Unstructured) ([]byte, bool) {
	secretData := getSecretData(secret)

	info := secretData[tfstateLockInfoKey].(string)
	decode, err := base64.StdEncoding.DecodeString(info)
	if err != nil {
		return nil, false
	}

	return decode, true
}

func setLockInfo(secret *unstructured.Unstructured, l []byte) {
	secretData := getSecretData(secret)
	secretData[tfstateLockInfoKey] = l
}

func setState(secret *unstructured.Unstructured, t []byte) {
	secretData := getSecretData(secret)
	secretData[tfstateKey] = t
}
