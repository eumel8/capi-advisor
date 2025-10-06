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

type OrphanedResults struct {
	Namespace        string
	Metal3DataClaims []string
	Metal3Data       []string
	Secrets          []string
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
		Namespace:        f.namespace,
		Metal3DataClaims: []string{},
		Metal3Data:       []string{},
		Secrets:          []string{},
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

		// Check ownerRef
		spec, ok := claim.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		ownerRef, ok := spec["ownerRef"].(map[string]interface{})
		if !ok || ownerRef == nil {
			orphaned = append(orphaned, claimName)
			continue
		}

		refName, ok := ownerRef["name"].(string)
		if !ok || refName == "" {
			orphaned = append(orphaned, claimName)
			continue
		}

		// Check if referenced Metal3Machine exists
		_, err := f.client.Clientset.Discovery().RESTClient().
			Get().
			AbsPath(fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s",
				m3mGVR.Group, m3mGVR.Version, f.namespace, m3mGVR.Resource, refName)).
			Do(ctx).
			Raw()
		if err != nil {
			// Metal3Machine doesn't exist, mark as orphaned
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

		// Check claimRef
		spec, ok := data.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		claimRef, ok := spec["claimRef"].(map[string]interface{})
		if !ok || claimRef == nil {
			orphaned = append(orphaned, dataName)
			continue
		}

		refName, ok := claimRef["name"].(string)
		if !ok || refName == "" {
			orphaned = append(orphaned, dataName)
			continue
		}

		// Check if referenced Metal3DataClaim exists
		_, err := f.client.Clientset.Discovery().RESTClient().
			Get().
			AbsPath(fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/metal3dataclaims/%s",
				f.namespace, refName)).
			Do(ctx).
			Raw()
		if err != nil {
			// Metal3DataClaim doesn't exist, mark as orphaned
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

		// Check ownerReferences
		hasValidOwner := false
		for _, owner := range secret.OwnerReferences {
			if strings.Contains(owner.Name, "metal3") {
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
		err := f.client.Clientset.Discovery().RESTClient().
			Delete().
			AbsPath(fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/metal3dataclaims/%s",
				f.namespace, claim)).
			Do(ctx).
			Error()
		if err != nil {
			return fmt.Errorf("failed to delete Metal3DataClaim %s: %w", claim, err)
		}
	}

	// Delete orphaned Metal3Data
	for _, data := range results.Metal3Data {
		err := f.client.Clientset.Discovery().RESTClient().
			Delete().
			AbsPath(fmt.Sprintf("/apis/infrastructure.cluster.x-k8s.io/v1beta1/namespaces/%s/metal3datas/%s",
				f.namespace, data)).
			Do(ctx).
			Error()
		if err != nil {
			return fmt.Errorf("failed to delete Metal3Data %s: %w", data, err)
		}
	}

	// Delete orphaned Secrets
	for _, secret := range results.Secrets {
		err := f.client.Clientset.CoreV1().Secrets(f.namespace).Delete(ctx, secret, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete Secret %s: %w", secret, err)
		}
	}

	return nil
}
