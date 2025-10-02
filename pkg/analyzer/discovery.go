package analyzer

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var SupportedGVKs = map[ComponentType]schema.GroupVersionKind{
	ClusterType: {
		Group:   "cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "Cluster",
	},
	MachineType: {
		Group:   "cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "Machine",
	},
	MachineSetType: {
		Group:   "cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "MachineSet",
	},
	MachineDeploymentType: {
		Group:   "cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "MachineDeployment",
	},
	Metal3MachineType: {
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "Metal3Machine",
	},
	Metal3ClusterType: {
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "Metal3Cluster",
	},
	BareMetalHostType: {
		Group:   "metal3.io",
		Version: "v1alpha1",
		Kind:    "BareMetalHost",
	},
	KubeadmControlPlaneType: {
		Group:   "controlplane.cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "KubeadmControlPlane",
	},
	KubeadmConfigType: {
		Group:   "bootstrap.cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "KubeadmConfig",
	},
}

type ComponentDiscovery struct {
	client client.Client
}

func NewComponentDiscovery(c client.Client) *ComponentDiscovery {
	return &ComponentDiscovery{client: c}
}

func (d *ComponentDiscovery) DiscoverComponents(ctx context.Context, namespace string, clusterName string) ([]*Component, error) {
	var allComponents []*Component

	for componentType, gvk := range SupportedGVKs {
		components, err := d.discoverComponentType(ctx, namespace, componentType, gvk)
		if err != nil {
			// Log the error but continue with other component types
			fmt.Printf("Warning: failed to discover %s components: %v\n", componentType, err)
			continue
		}
		allComponents = append(allComponents, components...)
	}

	// Filter by cluster name if specified
	if clusterName != "" {
		allComponents = d.filterByCluster(allComponents, clusterName)
	}

	return allComponents, nil
}

func (d *ComponentDiscovery) discoverComponentType(ctx context.Context, namespace string, compType ComponentType, gvk schema.GroupVersionKind) ([]*Component, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	})

	var opts []client.ListOption
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	err := d.client.List(ctx, list, opts...)
	if err != nil {
		// Check if this is a "not found" error for CRDs that might not be installed
		if meta.IsNoMatchError(err) {
			return nil, nil // Return empty list, not an error
		}
		return nil, fmt.Errorf("failed to list %s: %v", compType, err)
	}

	var components []*Component
	for _, item := range list.Items {
		component := d.convertUnstructuredToComponent(&item, compType, gvk)
		components = append(components, component)
	}

	return components, nil
}

func (d *ComponentDiscovery) convertUnstructuredToComponent(obj *unstructured.Unstructured, compType ComponentType, gvk schema.GroupVersionKind) *Component {
	component := &Component{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Type:      compType,
		GVK:       gvk,
		Metadata:  make(map[string]interface{}),
	}

	// Extract labels
	component.Metadata["labels"] = obj.GetLabels()

	// Extract conditions from status
	if status, found, err := unstructured.NestedMap(obj.Object, "status"); found && err == nil {
		if conditions, found, err := unstructured.NestedSlice(status, "conditions"); found && err == nil {
			component.Conditions = extractConditions(conditions)
		}

		// Store additional status information
		component.Metadata["status"] = status
	}

	// Extract spec for reference relationships
	if spec, found, err := unstructured.NestedMap(obj.Object, "spec"); found && err == nil {
		component.Metadata["spec"] = spec
	}

	// Determine component status based on conditions
	component.Status = d.determineComponentStatus(component.Conditions)

	return component
}

func extractConditions(conditions []interface{}) []metav1.Condition {
	var result []metav1.Condition

	for _, cond := range conditions {
		if condMap, ok := cond.(map[string]interface{}); ok {
			condition := metav1.Condition{}

			if t, ok := condMap["type"].(string); ok {
				condition.Type = t
			}
			if s, ok := condMap["status"].(string); ok {
				condition.Status = metav1.ConditionStatus(s)
			}
			if r, ok := condMap["reason"].(string); ok {
				condition.Reason = r
			}
			if m, ok := condMap["message"].(string); ok {
				condition.Message = m
			}
			if lt, ok := condMap["lastTransitionTime"].(string); ok {
				if parsedTime, err := time.Parse(time.RFC3339, lt); err == nil {
					condition.LastTransitionTime = metav1.NewTime(parsedTime)
				}
			}

			result = append(result, condition)
		}
	}

	return result
}

func (d *ComponentDiscovery) determineComponentStatus(conditions []metav1.Condition) ComponentStatus {
	if len(conditions) == 0 {
		return StatusUnknown
	}

	// Check for critical conditions first
	for _, condition := range conditions {
		switch condition.Type {
		case "Ready", "Available":
			if condition.Status == metav1.ConditionFalse {
				return StatusFailed
			} else if condition.Status == metav1.ConditionTrue {
				return StatusHealthy
			}
		case "InfrastructureReady", "BootstrapReady", "ControlPlaneReady":
			if condition.Status == metav1.ConditionFalse {
				return StatusDegraded
			}
		}
	}

	// Look for any False conditions as warnings
	for _, condition := range conditions {
		if condition.Status == metav1.ConditionFalse {
			return StatusDegraded
		}
	}

	// Check if there are any Unknown conditions
	for _, condition := range conditions {
		if condition.Status == metav1.ConditionUnknown {
			return StatusPending
		}
	}

	return StatusHealthy
}

func (d *ComponentDiscovery) filterByCluster(components []*Component, clusterName string) []*Component {
	var filtered []*Component
	clusterMap := make(map[string]bool)

	// First pass: find all Cluster resources with matching name
	for _, comp := range components {
		if comp.Type == ClusterType && comp.Name == clusterName {
			filtered = append(filtered, comp)
			clusterMap[comp.Namespace+"/"+comp.Name] = true
		}
	}

	// Second pass: filter components that belong to the specified cluster
	for _, comp := range components {
		if comp.Type == ClusterType {
			continue // Already added
		}

		// Check if component has cluster owner reference in labels
		if spec, ok := comp.Metadata["spec"].(map[string]interface{}); ok {
			// Check for clusterName in spec
			if cn, ok := spec["clusterName"].(string); ok && cn == clusterName {
				filtered = append(filtered, comp)
				continue
			}
		}

		// Check labels for cluster association
		if labels, ok := comp.Metadata["labels"].(map[string]string); ok {
			if cn, ok := labels["cluster.x-k8s.io/cluster-name"]; ok && cn == clusterName {
				filtered = append(filtered, comp)
				continue
			}
		}
	}

	return filtered
}