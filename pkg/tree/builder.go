package tree

import (
	"fmt"
	"strings"

	"capi-advisor/pkg/analyzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type TreeBuilder struct {
	components map[string]*analyzer.Component
}

func NewTreeBuilder() *TreeBuilder {
	return &TreeBuilder{
		components: make(map[string]*analyzer.Component),
	}
}

func (tb *TreeBuilder) BuildDependencyTree(components []*analyzer.Component) []*analyzer.Component {
	// Index all components by name and namespace
	for _, comp := range components {
		key := tb.getComponentKey(comp)
		tb.components[key] = comp
	}

	// Build relationships
	for _, comp := range components {
		tb.buildRelationships(comp)
	}

	// Return root components (those without parents)
	var roots []*analyzer.Component
	for _, comp := range components {
		if comp.Parent == nil {
			roots = append(roots, comp)
		}
	}

	return roots
}

func (tb *TreeBuilder) buildRelationships(comp *analyzer.Component) {
	switch comp.Type {
	case analyzer.MachineType:
		tb.buildMachineRelationships(comp)
	case analyzer.MachineSetType:
		tb.buildMachineSetRelationships(comp)
	case analyzer.MachineDeploymentType:
		tb.buildMachineDeploymentRelationships(comp)
	case analyzer.Metal3MachineType:
		tb.buildMetal3MachineRelationships(comp)
	case analyzer.ClusterType:
		tb.buildClusterRelationships(comp)
	case analyzer.KubeadmControlPlaneType:
		tb.buildKubeadmControlPlaneRelationships(comp)
	}
}

func (tb *TreeBuilder) buildMachineRelationships(machine *analyzer.Component) {
	if spec, ok := machine.Metadata["spec"].(map[string]interface{}); ok {
		// Link to cluster
		if clusterName, found, _ := unstructured.NestedString(spec, "clusterName"); found {
			if cluster := tb.findComponent(clusterName, machine.Namespace, analyzer.ClusterType); cluster != nil {
				tb.setParentChild(cluster, machine)
			}
		}

		// Link to infrastructure (Metal3Machine)
		if infraRef, found, _ := unstructured.NestedMap(spec, "infrastructureRef"); found {
			if name, ok := infraRef["name"].(string); ok {
				if infraMachine := tb.findComponent(name, machine.Namespace, analyzer.Metal3MachineType); infraMachine != nil {
					tb.setParentChild(machine, infraMachine)
				}
			}
		}

		// Link to bootstrap config (KubeadmConfig)
		if bootstrapRef, found, _ := unstructured.NestedMap(spec, "bootstrap", "configRef"); found {
			if name, ok := bootstrapRef["name"].(string); ok {
				if bootstrapConfig := tb.findComponent(name, machine.Namespace, analyzer.KubeadmConfigType); bootstrapConfig != nil {
					tb.setParentChild(machine, bootstrapConfig)
				}
			}
		}
	}
}

func (tb *TreeBuilder) buildMachineSetRelationships(machineSet *analyzer.Component) {
	// Find machines that belong to this MachineSet
	for _, comp := range tb.components {
		if comp.Type == analyzer.MachineType {
			if tb.isOwnedBy(comp, machineSet.Name, "MachineSet") {
				tb.setParentChild(machineSet, comp)
			}
		}
	}
}

func (tb *TreeBuilder) buildMachineDeploymentRelationships(machineDeployment *analyzer.Component) {
	// Find MachineSets that belong to this MachineDeployment
	for _, comp := range tb.components {
		if comp.Type == analyzer.MachineSetType {
			if tb.isOwnedBy(comp, machineDeployment.Name, "MachineDeployment") {
				tb.setParentChild(machineDeployment, comp)
			}
		}
	}
}

func (tb *TreeBuilder) buildMetal3MachineRelationships(metal3Machine *analyzer.Component) {
	if spec, ok := metal3Machine.Metadata["spec"].(map[string]interface{}); ok {
		// Link to BareMetalHost
		if hostSelector, found, _ := unstructured.NestedMap(spec, "hostSelector"); found {
			// Find BareMetalHost by labels/name
			bmh := tb.findBareMetalHostBySelector(hostSelector, metal3Machine.Namespace)
			if bmh != nil {
				tb.setParentChild(metal3Machine, bmh)
			}
		}
	}
}

func (tb *TreeBuilder) buildClusterRelationships(cluster *analyzer.Component) {
	if spec, ok := cluster.Metadata["spec"].(map[string]interface{}); ok {
		// Link to infrastructure (Metal3Cluster)
		if infraRef, found, _ := unstructured.NestedMap(spec, "infrastructureRef"); found {
			if name, ok := infraRef["name"].(string); ok {
				if infraCluster := tb.findComponent(name, cluster.Namespace, analyzer.Metal3ClusterType); infraCluster != nil {
					tb.setParentChild(cluster, infraCluster)
				}
			}
		}

		// Link to control plane
		if cpRef, found, _ := unstructured.NestedMap(spec, "controlPlaneRef"); found {
			if name, ok := cpRef["name"].(string); ok {
				if controlPlane := tb.findComponent(name, cluster.Namespace, analyzer.KubeadmControlPlaneType); controlPlane != nil {
					tb.setParentChild(cluster, controlPlane)
				}
			}
		}
	}
}

func (tb *TreeBuilder) buildKubeadmControlPlaneRelationships(kcp *analyzer.Component) {
	// Find machines that belong to this KubeadmControlPlane
	for _, comp := range tb.components {
		if comp.Type == analyzer.MachineType {
			if tb.isOwnedBy(comp, kcp.Name, "KubeadmControlPlane") {
				tb.setParentChild(kcp, comp)
			}
		}
	}
}

func (tb *TreeBuilder) findComponent(name, namespace string, compType analyzer.ComponentType) *analyzer.Component {
	for _, comp := range tb.components {
		if comp.Name == name && comp.Namespace == namespace && comp.Type == compType {
			return comp
		}
	}
	return nil
}

func (tb *TreeBuilder) findBareMetalHostBySelector(selector map[string]interface{}, namespace string) *analyzer.Component {
	// Simplified: look for BareMetalHost with matching name or labels
	for _, comp := range tb.components {
		if comp.Type == analyzer.BareMetalHostType && comp.Namespace == namespace {
			// In a real implementation, this would match labels
			return comp
		}
	}
	return nil
}

func (tb *TreeBuilder) isOwnedBy(comp *analyzer.Component, ownerName, ownerKind string) bool {
	if metadata, ok := comp.Metadata["metadata"].(map[string]interface{}); ok {
		if ownerRefs, found, _ := unstructured.NestedSlice(metadata, "ownerReferences"); found {
			for _, ref := range ownerRefs {
				if refMap, ok := ref.(map[string]interface{}); ok {
					if name, ok := refMap["name"].(string); ok && name == ownerName {
						if kind, ok := refMap["kind"].(string); ok && kind == ownerKind {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func (tb *TreeBuilder) setParentChild(parent, child *analyzer.Component) {
	if child.Parent == nil {
		child.Parent = parent
		parent.Children = append(parent.Children, child)
	}
}

func (tb *TreeBuilder) getComponentKey(comp *analyzer.Component) string {
	return comp.Namespace + "/" + comp.Name + "/" + string(comp.Type)
}

func (tb *TreeBuilder) PrintTree(components []*analyzer.Component) string {
	var result strings.Builder
	for _, comp := range components {
		tb.printComponent(&result, comp, 0)
	}
	return result.String()
}

func (tb *TreeBuilder) printComponent(result *strings.Builder, comp *analyzer.Component, depth int) {
	indent := strings.Repeat("  ", depth)
	statusIcon := tb.getStatusIcon(comp.Status)

	result.WriteString(fmt.Sprintf("%s%s %s/%s (%s)\n",
		indent, statusIcon, comp.Type, comp.Name, comp.Status))

	// Print conditions
	for _, condition := range comp.Conditions {
		conditionIcon := "?"
		if condition.Status == metav1.ConditionTrue {
			conditionIcon = "✓"
		} else if condition.Status == metav1.ConditionFalse {
			conditionIcon = "✗"
		}
		result.WriteString(fmt.Sprintf("%s  %s %s: %s\n",
			indent, conditionIcon, condition.Type, condition.Message))
	}

	// Print children
	for _, child := range comp.Children {
		tb.printComponent(result, child, depth+1)
	}
}

func (tb *TreeBuilder) getStatusIcon(status analyzer.ComponentStatus) string {
	switch status {
	case analyzer.StatusHealthy:
		return "✅"
	case analyzer.StatusDegraded:
		return "⚠️"
	case analyzer.StatusFailed:
		return "❌"
	case analyzer.StatusPending:
		return "⏳"
	default:
		return "❓"
	}
}