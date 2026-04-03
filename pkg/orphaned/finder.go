package orphaned

import (
	"context"
	"fmt"
	"strings"

	"capi-advisor/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type SecretMismatch struct {
	SecretName    string
	Metal3Machine string
	Reason        string
}

type OrphanedResults struct {
	Namespace             string
	Metal3DataClaims      []string
	Metal3Data            []string
	Secrets               []string
	SecretOwnerMismatches []SecretMismatch
}

type Finder struct {
	client    *client.K8sClient
	namespace string
}

func NewFinder(k8sClient *client.K8sClient, namespace string) *Finder {
	return &Finder{
		client:    k8sClient,
		namespace: namespace,
	}
}

func (f *Finder) FindOrphaned(ctx context.Context) (*OrphanedResults, error) {
	results := &OrphanedResults{
		Namespace:             f.namespace,
		Metal3DataClaims:      []string{},
		Metal3Data:            []string{},
		Secrets:               []string{},
		SecretOwnerMismatches: []SecretMismatch{},
	}

	// Find orphaned Metal3DataClaims
	claims, err := f.findOrphanedMetal3DataClaims(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find orphaned Metal3DataClaims: %w", err)
	}
	results.Metal3DataClaims = claims

	// Find orphaned Metal3Data
	data, err := f.findOrphanedMetal3Data(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find orphaned Metal3Data: %w", err)
	}
	results.Metal3Data = data

	// Find orphaned Secrets
	secrets, err := f.findOrphanedSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find orphaned Secrets: %w", err)
	}
	results.Secrets = secrets

	// Find Metal3Machine secret ownerRef mismatches
	mismatches, err := f.findMetal3MachineSecretMismatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find Metal3Machine secret owner mismatches: %w", err)
	}
	results.SecretOwnerMismatches = mismatches

	return results, nil
}

func (f *Finder) findOrphanedMetal3DataClaims(ctx context.Context) ([]string, error) {
	orphaned := []string{}

	// List all Metal3DataClaims
	claimList, err := f.client.Clientset.Discovery().RESTClient().
		Get().
		AbsPath("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/" + f.namespace + "/metal3dataclaims").
		Do(ctx).
		Raw()
	if err != nil {
		return nil, err
	}

	// Parse with unstructured
	obj := &unstructured.UnstructuredList{}
	if err := obj.UnmarshalJSON(claimList); err != nil {
		return nil, err
	}

	// Define Metal3Machine GVR for lookups
	m3mGVR := schema.GroupVersionResource{
		Group:    "infrastructure.cluster.x-k8s.io",
		Version:  "v1beta1",
		Resource: "metal3machines",
	}

	for _, claim := range obj.Items {
		claimName := claim.GetName()

		// Check metadata.ownerReferences
		ownerRefs := claim.GetOwnerReferences()
		if len(ownerRefs) == 0 {
			orphaned = append(orphaned, claimName)
			continue
		}

		// Verify that at least one owner reference points to an existing Metal3Machine
		hasValidOwner := false
		for _, ownerRef := range ownerRefs {
			if ownerRef.Kind == "Metal3Machine" {
				// Check if referenced Metal3Machine exists
				_, err := f.client.Clientset.Discovery().RESTClient().
					Get().
					AbsPath(fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s",
						m3mGVR.Group, m3mGVR.Version, f.namespace, m3mGVR.Resource, ownerRef.Name)).
					Do(ctx).
					Raw()
				if err == nil {
					// Owner exists
					hasValidOwner = true
					break
				}
			}
		}

		if !hasValidOwner {
			orphaned = append(orphaned, claimName)
		}
	}

	return orphaned, nil
}

func (f *Finder) findOrphanedMetal3Data(ctx context.Context) ([]string, error) {
	orphaned := []string{}

	// List all Metal3Data
	dataList, err := f.client.Clientset.Discovery().RESTClient().
		Get().
		AbsPath("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/" + f.namespace + "/metal3datas").
		Do(ctx).
		Raw()
	if err != nil {
		return nil, err
	}

	// Parse with unstructured
	obj := &unstructured.UnstructuredList{}
	if err := obj.UnmarshalJSON(dataList); err != nil {
		return nil, err
	}

	for _, data := range obj.Items {
		dataName := data.GetName()

		// Check metadata.ownerReferences
		ownerRefs := data.GetOwnerReferences()
		if len(ownerRefs) == 0 {
			orphaned = append(orphaned, dataName)
			continue
		}

		// Verify that at least one owner reference points to an existing Metal3DataClaim
		hasValidOwner := false
		for _, ownerRef := range ownerRefs {
			if ownerRef.Kind == "Metal3DataClaim" {
				// Check if referenced Metal3DataClaim exists
				_, err := f.client.Clientset.Discovery().RESTClient().
					Get().
					AbsPath(fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/metal3dataclaims/%s",
						f.namespace, ownerRef.Name)).
					Do(ctx).
					Raw()
				if err == nil {
					// Owner exists
					hasValidOwner = true
					break
				}
			}
		}

		if !hasValidOwner {
			orphaned = append(orphaned, dataName)
		}
	}

	return orphaned, nil
}

func (f *Finder) findOrphanedSecrets(ctx context.Context) ([]string, error) {
	orphaned := []string{}

	// List all secrets in namespace
	secretList, err := f.client.Clientset.CoreV1().Secrets(f.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, secret := range secretList.Items {
		secretName := secret.Name

		// Skip if not metal3-related or is a token
		if !strings.Contains(secretName, "metal3") || strings.Contains(secretName, "token") {
			continue
		}

		// Check metadata.ownerReferences
		if len(secret.OwnerReferences) == 0 {
			orphaned = append(orphaned, secretName)
			continue
		}

		// Verify that at least one owner reference points to an existing resource
		hasValidOwner := false
		for _, ownerRef := range secret.OwnerReferences {
			// Check if the owner exists based on its Kind
			var exists bool
			switch ownerRef.Kind {
			case "Metal3Machine":
				_, err := f.client.Clientset.Discovery().RESTClient().
					Get().
					AbsPath(fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/metal3machines/%s",
						f.namespace, ownerRef.Name)).
					Do(ctx).
					Raw()
				exists = (err == nil)
			case "Metal3DataClaim":
				_, err := f.client.Clientset.Discovery().RESTClient().
					Get().
					AbsPath(fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/metal3dataclaims/%s",
						f.namespace, ownerRef.Name)).
					Do(ctx).
					Raw()
				exists = (err == nil)
			case "Metal3Data":
				_, err := f.client.Clientset.Discovery().RESTClient().
					Get().
					AbsPath(fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/metal3datas/%s",
						f.namespace, ownerRef.Name)).
					Do(ctx).
					Raw()
				exists = (err == nil)
			default:
				// For other kinds, assume they exist if we can't verify
				exists = true
			}

			if exists {
				hasValidOwner = true
				break
			}
		}

		if !hasValidOwner {
			orphaned = append(orphaned, secretName)
		}
	}

	return orphaned, nil
}

func (f *Finder) CleanupOrphaned(ctx context.Context, results *OrphanedResults) error {
	// Delete orphaned Metal3DataClaims
	for _, claim := range results.Metal3DataClaims {
		if err := f.removeFinalizers(ctx, "metal3dataclaims", claim); err != nil {
			return fmt.Errorf("failed to remove finalizers from Metal3DataClaim %s: %w", claim, err)
		}
		err := f.client.Clientset.Discovery().RESTClient().
			Delete().
			AbsPath(fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/metal3dataclaims/%s",
				f.namespace, claim)).
			Do(ctx).
			Error()
		if err != nil {
			fmt.Printf("warning: failed to delete Metal3DataClaim %s: %v\n", claim, err)
		}
	}

	// Delete orphaned Metal3Data
	for _, data := range results.Metal3Data {
		if err := f.removeFinalizers(ctx, "metal3datas", data); err != nil {
			return fmt.Errorf("failed to remove finalizers from Metal3Data %s: %w", data, err)
		}
		err := f.client.Clientset.Discovery().RESTClient().
			Delete().
			AbsPath(fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/metal3datas/%s",
				f.namespace, data)).
			Do(ctx).
			Error()
		if err != nil {
			fmt.Printf("warning: failed to delete Metal3Data %s: %v\n", data, err)
		}
	}

	// Delete orphaned Secrets
	for _, secret := range results.Secrets {
		if err := f.deleteSecret(ctx, secret); err != nil {
			fmt.Printf("warning: %v\n", err)
		}
	}

	// Delete interfering secrets with ownerRef mismatches
	for _, mismatch := range results.SecretOwnerMismatches {
		if err := f.deleteSecret(ctx, mismatch.SecretName); err != nil {
			fmt.Printf("warning: %v\n", err)
		}
	}

	return nil
}

func (f *Finder) deleteSecret(ctx context.Context, name string) error {
	secretObj, err := f.client.Clientset.CoreV1().Secrets(f.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Secret %s: %w", name, err)
	}
	if len(secretObj.Finalizers) > 0 {
		secretObj.Finalizers = []string{}
		_, err = f.client.Clientset.CoreV1().Secrets(f.namespace).Update(ctx, secretObj, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to remove finalizers from Secret %s: %w", name, err)
		}
	}
	if err := f.client.Clientset.CoreV1().Secrets(f.namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete Secret %s: %w", name, err)
	}
	return nil
}

// findMetal3MachineSecretMismatches checks secrets referenced in Metal3Machine
// spec.metaData and spec.networkData for missing or stale ownerReferences.
func (f *Finder) findMetal3MachineSecretMismatches(ctx context.Context) ([]SecretMismatch, error) {
	mismatches := []SecretMismatch{}

	m3mListRaw, err := f.client.Clientset.Discovery().RESTClient().
		Get().
		AbsPath("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/" + f.namespace + "/metal3machines").
		Do(ctx).
		Raw()
	if err != nil {
		// CRD may not be installed — skip silently
		return mismatches, nil
	}

	m3mList := &unstructured.UnstructuredList{}
	if err := m3mList.UnmarshalJSON(m3mListRaw); err != nil {
		return nil, err
	}

	for _, m3m := range m3mList.Items {
		m3mName := m3m.GetName()
		m3mUID := string(m3m.GetUID())

		for _, field := range []string{"metaData", "networkData"} {
			secretName, found, _ := unstructured.NestedString(m3m.Object, "spec", field, "name")
			if !found || secretName == "" {
				continue
			}

			secret, err := f.client.Clientset.CoreV1().Secrets(f.namespace).Get(ctx, secretName, metav1.GetOptions{})
			if err != nil {
				mismatches = append(mismatches, SecretMismatch{
					SecretName:    secretName,
					Metal3Machine: m3mName,
					Reason:        fmt.Sprintf("spec.%s references non-existent secret", field),
				})
				continue
			}

			hasMatchingOwner := false
			hasStaleOwner := false
			for _, ownerRef := range secret.OwnerReferences {
				switch ownerRef.Kind {
				case "Metal3Machine":
					if ownerRef.Name == m3mName && string(ownerRef.UID) == m3mUID {
						hasMatchingOwner = true
					} else if ownerRef.Name == m3mName {
						// Same name, different UID — stale reference from a previous provisioning cycle
						hasStaleOwner = true
					}
				case "Metal3DataClaim", "Metal3Data":
					// Created through the data pipeline — valid ownership
					hasMatchingOwner = true
				}
				if hasMatchingOwner {
					break
				}
			}

			switch {
			case hasStaleOwner:
				mismatches = append(mismatches, SecretMismatch{
					SecretName:    secretName,
					Metal3Machine: m3mName,
					Reason:        fmt.Sprintf("spec.%s secret has stale ownerRef UID (interference from previous provisioning cycle)", field),
				})
			case !hasMatchingOwner && len(secret.OwnerReferences) == 0:
				mismatches = append(mismatches, SecretMismatch{
					SecretName:    secretName,
					Metal3Machine: m3mName,
					Reason:        fmt.Sprintf("spec.%s secret has no ownerReferences", field),
				})
			case !hasMatchingOwner:
				mismatches = append(mismatches, SecretMismatch{
					SecretName:    secretName,
					Metal3Machine: m3mName,
					Reason:        fmt.Sprintf("spec.%s secret ownerRef does not match this Metal3Machine", field),
				})
			}
		}
	}

	return mismatches, nil
}

// removeFinalizers removes all finalizers from a Metal3 resource
func (f *Finder) removeFinalizers(ctx context.Context, resourceType, resourceName string) error {
	// Get the current resource
	resourcePath := fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/%s/%s",
		f.namespace, resourceType, resourceName)

	rawResource, err := f.client.Clientset.Discovery().RESTClient().
		Get().
		AbsPath(resourcePath).
		Do(ctx).
		Raw()
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Parse with unstructured
	obj := &unstructured.Unstructured{}
	if err := obj.UnmarshalJSON(rawResource); err != nil {
		return fmt.Errorf("failed to unmarshal resource: %w", err)
	}

	// Check if there are finalizers
	finalizers := obj.GetFinalizers()
	if len(finalizers) == 0 {
		return nil // No finalizers to remove
	}

	// Remove all finalizers
	obj.SetFinalizers([]string{})

	// Update the resource
	updatedJSON, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	_, err = f.client.Clientset.Discovery().RESTClient().
		Put().
		AbsPath(resourcePath).
		Body(updatedJSON).
		SetHeader("Content-Type", "application/json").
		Do(ctx).
		Raw()
	if err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}

	return nil
}
