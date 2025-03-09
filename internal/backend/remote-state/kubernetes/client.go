// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Import to initialize client auth plugins.
	"k8s.io/utils/pointer"

	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"

	coordinationv1 "k8s.io/api/coordination/v1"
	coordinationclientv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
)

const (
	tfstateKey                = "tfstate"
	tfstateSecretSuffixKey    = "tfstateSecretSuffix"
	tfstateWorkspaceKey       = "tfstateWorkspace"
	tfstateLockInfoAnnotation = "app.terraform.io/lock-info"

	managedByKey = "app.kubernetes.io/managed-by"

	defaultChunkSize = 1048576
)

type RemoteClient struct {
	kubernetesSecretClient dynamic.ResourceInterface
	kubernetesLeaseClient  coordinationclientv1.LeaseInterface
	namespace              string
	labels                 map[string]string
	nameSuffix             string
	workspace              string
}

func (c *RemoteClient) Get() (payload *remote.Payload, err error) {
	secretList, err := c.getSecrets()
	if err != nil {
		return nil, err
	}

	if len(secretList) == 0 {
		return nil, nil
	}

	var data []string
	for _, secret := range secretList {
		secretData := getSecretData(&secret)
		stateRaw, ok := secretData[tfstateKey]
		if !ok {
			// The secret exists but there is no state in it
			return nil, nil
		}
		data = append(data, stateRaw.(string))
	}

	state, err := uncompressState(data)
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

func (c *RemoteClient) getSecrets() ([]unstructured.Unstructured, error) {
	ls := metav1.SetAsLabelSelector(c.getLabels())
	res, err := c.kubernetesSecretClient.List(context.Background(),
		metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(ls),
		},
	)
	if err != nil {
		return []unstructured.Unstructured{}, err
	}

	// NOTE we need to sort the list as the k8s API will return
	// the list sorted by name which will corrupt the state when
	// the number of secrets is greater than 10
	items := make([]unstructured.Unstructured, len(res.Items))
	for _, item := range res.Items {
		name := item.GetName()
		nameParts := strings.Split(name, "-")
		// Because large Terraform state files are split into multiple secrets,
		// we parse the index from the secret name.
		index, err := strconv.Atoi(nameParts[len(nameParts)-1])
		if err != nil {
			index = 0
		}
		items[index] = item
	}
	return items, nil
}

func (c *RemoteClient) Put(data []byte) error {
	ctx := context.Background()

	payload, err := compressState(data)
	if err != nil {
		return err
	}

	chunks := chunkPayload(payload, defaultChunkSize)
	existingSecrets, err := c.getSecrets()
	if err != nil {
		return err
	}

	for idx, data := range chunks {
		secretName, err := c.createSecretName(idx)
		if err != nil {
			return err
		}

		secret, err := c.getSecret(secretName)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}
			secret = &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": metav1.ObjectMeta{
						Name:        secretName,
						Namespace:   c.namespace,
						Labels:      c.getLabels(),
						Annotations: map[string]string{"encoding": "gzip"},
					},
				},
			}
			secret, err = c.kubernetesSecretClient.Create(ctx, secret, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		}

		setState(secret, data)
		_, err = c.kubernetesSecretClient.Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	// remove old secrets
	existingSecretCount := len(existingSecrets)
	newSecretCount := len(chunks)
	if existingSecretCount == newSecretCount {
		return nil
	}
	for i := newSecretCount; i < existingSecretCount; i++ {
		secretName, err := c.createSecretName(i)
		if err != nil {
			return err
		}
		err = c.deleteSecret(secretName)
		if err != nil {
			return err
		}
	}
	return nil
}

// chunkPayload splits the state payload into byte arrays of the given size
func chunkPayload(buf []byte, size int) [][]byte {
	chunks := make([][]byte, 0, len(buf)/size+1)
	for len(buf) >= size {
		var chunk []byte
		chunk, buf = buf[:size], buf[size:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf)
	}
	return chunks
}

// Delete the state secret
func (c *RemoteClient) Delete() error {
	secretList, err := c.getSecrets()
	if err != nil {
		return err
	}

	for i, _ := range secretList {
		secretName, err := c.createSecretName(i)
		if err != nil {
			return err
		}

		err = c.deleteSecret(secretName)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}
		}
	}

	leaseName, err := c.createLeaseName()
	if err != nil {
		return err
	}

	err = c.deleteLease(leaseName)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	ctx := context.Background()
	leaseName, err := c.createLeaseName()
	if err != nil {
		return "", err
	}

	lease, err := c.getLease(leaseName)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return "", err
		}

		labels := c.getLabels()
		lease = &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:   leaseName,
				Labels: labels,
				Annotations: map[string]string{
					tfstateLockInfoAnnotation: string(info.Marshal()),
				},
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: pointer.StringPtr(info.ID),
			},
		}

		_, err = c.kubernetesLeaseClient.Create(ctx, lease, metav1.CreateOptions{})
		if err != nil {
			return "", err
		} else {
			return info.ID, nil
		}
	}

	if lease.Spec.HolderIdentity != nil {
		if *lease.Spec.HolderIdentity == info.ID {
			return info.ID, nil
		}

		currentLockInfo, err := c.getLockInfo(lease)
		if err != nil {
			return "", err
		}

		lockErr := &statemgr.LockError{
			Info: currentLockInfo,
			Err:  errors.New("the state is already locked by another terraform client"),
		}
		return "", lockErr
	}

	lease.Spec.HolderIdentity = pointer.StringPtr(info.ID)
	setLockInfo(lease, info.Marshal())
	_, err = c.kubernetesLeaseClient.Update(ctx, lease, metav1.UpdateOptions{})
	if err != nil {
		return "", err
	}

	return info.ID, err
}

func (c *RemoteClient) Unlock(id string) error {
	leaseName, err := c.createLeaseName()
	if err != nil {
		return err
	}

	lease, err := c.getLease(leaseName)
	if err != nil {
		return err
	}

	if lease.Spec.HolderIdentity == nil {
		return fmt.Errorf("state is already unlocked")
	}

	lockInfo, err := c.getLockInfo(lease)
	if err != nil {
		return err
	}

	lockErr := &statemgr.LockError{Info: lockInfo}
	if *lease.Spec.HolderIdentity != id {
		lockErr.Err = fmt.Errorf("lock id %q does not match existing lock", id)
		return lockErr
	}

	lease.Spec.HolderIdentity = nil
	removeLockInfo(lease)

	_, err = c.kubernetesLeaseClient.Update(context.Background(), lease, metav1.UpdateOptions{})
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	return nil
}

func (c *RemoteClient) getLockInfo(lease *coordinationv1.Lease) (*statemgr.LockInfo, error) {
	lockData, ok := getLockInfo(lease)
	if len(lockData) == 0 || !ok {
		return nil, nil
	}

	lockInfo := &statemgr.LockInfo{}
	err := json.Unmarshal(lockData, lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

func (c *RemoteClient) getLabels() map[string]string {
	l := map[string]string{
		tfstateKey:             "true",
		tfstateSecretSuffixKey: c.nameSuffix,
		tfstateWorkspaceKey:    c.workspace,
		managedByKey:           "terraform",
	}

	if len(c.labels) != 0 {
		for k, v := range c.labels {
			l[k] = v
		}
	}

	return l
}

func (c *RemoteClient) getSecret(name string) (*unstructured.Unstructured, error) {
	return c.kubernetesSecretClient.Get(context.Background(), name, metav1.GetOptions{})
}

func (c *RemoteClient) getLease(name string) (*coordinationv1.Lease, error) {
	return c.kubernetesLeaseClient.Get(context.Background(), name, metav1.GetOptions{})
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
	delOps := metav1.DeleteOptions{PropagationPolicy: &delProp}
	return c.kubernetesSecretClient.Delete(context.Background(), name, delOps)
}

func (c *RemoteClient) deleteLease(name string) error {
	secret, err := c.getLease(name)
	if err != nil {
		return err
	}

	labels := secret.GetLabels()
	v, ok := labels[tfstateKey]
	if !ok || v != "true" {
		return fmt.Errorf("Lease does does not have %q label", tfstateKey)
	}

	delProp := metav1.DeletePropagationBackground
	delOps := metav1.DeleteOptions{PropagationPolicy: &delProp}
	return c.kubernetesLeaseClient.Delete(context.Background(), name, delOps)
}

func (c *RemoteClient) createSecretName(idx int) (string, error) {
	secretName := strings.Join([]string{tfstateKey, c.workspace, c.nameSuffix}, "-")

	if idx > 0 {
		secretName = fmt.Sprintf("%s-part-%d", secretName, idx)
	}

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

func (c *RemoteClient) createLeaseName() (string, error) {
	n, err := c.createSecretName(0)
	if err != nil {
		return "", err
	}
	return "lock-" + n, nil
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

func uncompressState(data []string) ([]byte, error) {
	var rawData []byte
	for _, chunk := range data {
		decode, err := base64.StdEncoding.DecodeString(chunk)
		if err != nil {
			return nil, err
		}
		rawData = append(rawData, decode...)
	}

	b := new(bytes.Buffer)
	gz, err := gzip.NewReader(bytes.NewReader(rawData))
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
	if m, ok := secret.Object["data"].(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func getLockInfo(lease *coordinationv1.Lease) ([]byte, bool) {
	info, ok := lease.ObjectMeta.GetAnnotations()[tfstateLockInfoAnnotation]
	if !ok {
		return nil, false
	}
	return []byte(info), true
}

func setLockInfo(lease *coordinationv1.Lease, l []byte) {
	annotations := lease.ObjectMeta.GetAnnotations()
	if annotations != nil {
		annotations[tfstateLockInfoAnnotation] = string(l)
	} else {
		annotations = map[string]string{
			tfstateLockInfoAnnotation: string(l),
		}
	}
	lease.ObjectMeta.SetAnnotations(annotations)
}

func removeLockInfo(lease *coordinationv1.Lease) {
	annotations := lease.ObjectMeta.GetAnnotations()
	delete(annotations, tfstateLockInfoAnnotation)
	lease.ObjectMeta.SetAnnotations(annotations)
}

func setState(secret *unstructured.Unstructured, t []byte) {
	secretData := getSecretData(secret)
	secretData[tfstateKey] = t
	secret.Object["data"] = secretData
}
