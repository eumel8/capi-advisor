package analyzer

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ComponentType string

const (
	ClusterType           ComponentType = "Cluster"
	MachineType          ComponentType = "Machine"
	MachineSetType       ComponentType = "MachineSet"
	MachineDeploymentType ComponentType = "MachineDeployment"
	Metal3MachineType    ComponentType = "Metal3Machine"
	Metal3ClusterType    ComponentType = "Metal3Cluster"
	BareMetalHostType    ComponentType = "BareMetalHost"
	KubeadmControlPlaneType ComponentType = "KubeadmControlPlane"
	KubeadmConfigType    ComponentType = "KubeadmConfig"
)

type Component struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Type       ComponentType     `json:"type"`
	GVK        schema.GroupVersionKind `json:"gvk"`
	Conditions []metav1.Condition `json:"conditions"`
	Status     ComponentStatus    `json:"status"`
	Children   []*Component      `json:"children,omitempty"`
	Parent     *Component        `json:"parent,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type ComponentStatus string

const (
	StatusHealthy   ComponentStatus = "Healthy"
	StatusDegraded  ComponentStatus = "Degraded"
	StatusFailed    ComponentStatus = "Failed"
	StatusPending   ComponentStatus = "Pending"
	StatusUnknown   ComponentStatus = "Unknown"
)

type ConditionSeverity string

const (
	SeverityCritical ConditionSeverity = "Critical"
	SeverityWarning  ConditionSeverity = "Warning"
	SeverityInfo     ConditionSeverity = "Info"
)

type Issue struct {
	Component    *Component        `json:"component"`
	Condition    metav1.Condition  `json:"condition"`
	Severity     ConditionSeverity `json:"severity"`
	Description  string            `json:"description"`
	Cause        string            `json:"cause"`
	Resolution   string            `json:"resolution"`
	Dependencies []*Component      `json:"dependencies,omitempty"`
}

type AnalysisResult struct {
	Components []*Component `json:"components"`
	Issues     []*Issue     `json:"issues"`
	Summary    Summary      `json:"summary"`
}

type Summary struct {
	TotalComponents int                        `json:"total_components"`
	StatusCounts    map[ComponentStatus]int    `json:"status_counts"`
	SeverityCounts  map[ConditionSeverity]int  `json:"severity_counts"`
	ClusterHealth   ComponentStatus            `json:"cluster_health"`
}