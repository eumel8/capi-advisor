package advisor

import (
	"fmt"
	"sort"
	"strings"

	"capi-advisor/pkg/analyzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Advisor struct {
	knowledgeBase map[string]KnowledgeEntry
}

type KnowledgeEntry struct {
	Condition   string
	Severity    analyzer.ConditionSeverity
	Cause       string
	Resolution  string
	Dependencies []string
}

func NewAdvisor() *Advisor {
	advisor := &Advisor{
		knowledgeBase: make(map[string]KnowledgeEntry),
	}
	advisor.loadKnowledgeBase()
	return advisor
}

func (a *Advisor) loadKnowledgeBase() {
	// Cluster API conditions
	a.knowledgeBase["Cluster.Ready.False"] = KnowledgeEntry{
		Condition:  "Cluster Ready is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Infrastructure or control plane is not ready",
		Resolution: "Check infrastructure and control plane components",
		Dependencies: []string{"infrastructureRef", "controlPlaneRef"},
	}

	a.knowledgeBase["Cluster.InfrastructureReady.False"] = KnowledgeEntry{
		Condition:  "Cluster InfrastructureReady is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Infrastructure provider is not ready",
		Resolution: "Check Metal3Cluster status and network configuration",
		Dependencies: []string{"Metal3Cluster"},
	}

	a.knowledgeBase["Cluster.ControlPlaneReady.False"] = KnowledgeEntry{
		Condition:  "Cluster ControlPlaneReady is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Control plane nodes are not ready",
		Resolution: "Check KubeadmControlPlane and control plane machines",
		Dependencies: []string{"KubeadmControlPlane", "Machine"},
	}

	// Machine conditions
	a.knowledgeBase["Machine.Ready.False"] = KnowledgeEntry{
		Condition:  "Machine Ready is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Machine infrastructure or bootstrap is not ready",
		Resolution: "Check Metal3Machine and KubeadmConfig status",
		Dependencies: []string{"Metal3Machine", "KubeadmConfig"},
	}

	a.knowledgeBase["Machine.InfrastructureReady.False"] = KnowledgeEntry{
		Condition:  "Machine InfrastructureReady is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Metal3Machine is not ready",
		Resolution: "Check BareMetalHost status and provisioning",
		Dependencies: []string{"Metal3Machine", "BareMetalHost"},
	}

	a.knowledgeBase["Machine.BootstrapReady.False"] = KnowledgeEntry{
		Condition:  "Machine BootstrapReady is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Bootstrap configuration is not ready",
		Resolution: "Check KubeadmConfig and cluster connectivity",
		Dependencies: []string{"KubeadmConfig"},
	}

	// Metal3Machine conditions
	a.knowledgeBase["Metal3Machine.Ready.False"] = KnowledgeEntry{
		Condition:  "Metal3Machine Ready is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "BareMetalHost is not available or not provisioned",
		Resolution: "Check BareMetalHost status, BMC connectivity, and provisioning",
		Dependencies: []string{"BareMetalHost"},
	}

	a.knowledgeBase["Metal3Machine.AssociationReady.False"] = KnowledgeEntry{
		Condition:  "Metal3Machine AssociationReady is False",
		Severity:   analyzer.SeverityWarning,
		Cause:      "Unable to associate with BareMetalHost",
		Resolution: "Check hostSelector configuration and BareMetalHost availability",
		Dependencies: []string{"BareMetalHost"},
	}

	// BareMetalHost conditions
	a.knowledgeBase["BareMetalHost.Ready.False"] = KnowledgeEntry{
		Condition:  "BareMetalHost Ready is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Hardware is not available or provisioning failed",
		Resolution: "Check BMC connectivity, hardware status, and provisioning image",
		Dependencies: []string{},
	}

	a.knowledgeBase["BareMetalHost.Available.False"] = KnowledgeEntry{
		Condition:  "BareMetalHost Available is False",
		Severity:   analyzer.SeverityWarning,
		Cause:      "Host is not available for provisioning",
		Resolution: "Check if host is powered on and BMC is accessible",
		Dependencies: []string{},
	}

	a.knowledgeBase["BareMetalHost.Provisioned.False"] = KnowledgeEntry{
		Condition:  "BareMetalHost Provisioned is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Provisioning process failed or is in progress",
		Resolution: "Check provisioning logs, image availability, and network connectivity",
		Dependencies: []string{},
	}

	// KubeadmControlPlane conditions
	a.knowledgeBase["KubeadmControlPlane.Ready.False"] = KnowledgeEntry{
		Condition:  "KubeadmControlPlane Ready is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Control plane nodes are not ready",
		Resolution: "Check control plane machine status and etcd health",
		Dependencies: []string{"Machine"},
	}

	a.knowledgeBase["KubeadmControlPlane.Initialized.False"] = KnowledgeEntry{
		Condition:  "KubeadmControlPlane Initialized is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Control plane initialization failed",
		Resolution: "Check first control plane machine and kubeadm logs",
		Dependencies: []string{"Machine"},
	}

	// KubeadmConfig conditions
	a.knowledgeBase["KubeadmConfig.Ready.False"] = KnowledgeEntry{
		Condition:  "KubeadmConfig Ready is False",
		Severity:   analyzer.SeverityWarning,
		Cause:      "Bootstrap configuration is not ready",
		Resolution: "Check cluster connectivity and certificates",
		Dependencies: []string{},
	}
}

func (a *Advisor) AnalyzeComponents(components []*analyzer.Component) *analyzer.AnalysisResult {
	var issues []*analyzer.Issue
	statusCounts := make(map[analyzer.ComponentStatus]int)
	severityCounts := make(map[analyzer.ConditionSeverity]int)

	for _, comp := range components {
		statusCounts[comp.Status]++
		componentIssues := a.analyzeComponent(comp)
		issues = append(issues, componentIssues...)

		for _, issue := range componentIssues {
			severityCounts[issue.Severity]++
		}
	}

	// Sort issues by severity
	sort.Slice(issues, func(i, j int) bool {
		severityOrder := map[analyzer.ConditionSeverity]int{
			analyzer.SeverityCritical: 0,
			analyzer.SeverityWarning:  1,
			analyzer.SeverityInfo:     2,
		}
		return severityOrder[issues[i].Severity] < severityOrder[issues[j].Severity]
	})

	// Determine overall cluster health
	clusterHealth := a.determineClusterHealth(statusCounts, severityCounts)

	return &analyzer.AnalysisResult{
		Components: components,
		Issues:     issues,
		Summary: analyzer.Summary{
			TotalComponents: len(components),
			StatusCounts:    statusCounts,
			SeverityCounts:  severityCounts,
			ClusterHealth:   clusterHealth,
		},
	}
}

func (a *Advisor) analyzeComponent(comp *analyzer.Component) []*analyzer.Issue {
	var issues []*analyzer.Issue

	for _, condition := range comp.Conditions {
		if condition.Status == metav1.ConditionFalse {
			key := fmt.Sprintf("%s.%s.%s", comp.Type, condition.Type, condition.Status)
			if knowledge, exists := a.knowledgeBase[key]; exists {
				issue := &analyzer.Issue{
					Component:   comp,
					Condition:   condition,
					Severity:    knowledge.Severity,
					Description: knowledge.Condition,
					Cause:       knowledge.Cause,
					Resolution:  knowledge.Resolution,
				}

				// Find dependency components
				issue.Dependencies = a.findDependencies(comp, knowledge.Dependencies)
				issues = append(issues, issue)
			} else {
				// Generic issue for unknown conditions
				issue := &analyzer.Issue{
					Component:   comp,
					Condition:   condition,
					Severity:    analyzer.SeverityWarning,
					Description: fmt.Sprintf("%s %s is %s", comp.Type, condition.Type, condition.Status),
					Cause:       condition.Reason,
					Resolution:  "Check the component logs and configuration",
				}
				issues = append(issues, issue)
			}
		}
	}

	return issues
}

func (a *Advisor) findDependencies(comp *analyzer.Component, depTypes []string) []*analyzer.Component {
	var deps []*analyzer.Component

	for _, depType := range depTypes {
		// Look in children first
		for _, child := range comp.Children {
			if string(child.Type) == depType {
				deps = append(deps, child)
			}
		}

		// Look in parent's children
		if comp.Parent != nil {
			for _, sibling := range comp.Parent.Children {
				if string(sibling.Type) == depType && sibling != comp {
					deps = append(deps, sibling)
				}
			}
		}
	}

	return deps
}

func (a *Advisor) determineClusterHealth(statusCounts map[analyzer.ComponentStatus]int, severityCounts map[analyzer.ConditionSeverity]int) analyzer.ComponentStatus {
	if severityCounts[analyzer.SeverityCritical] > 0 || statusCounts[analyzer.StatusFailed] > 0 {
		return analyzer.StatusFailed
	}
	if severityCounts[analyzer.SeverityWarning] > 0 || statusCounts[analyzer.StatusDegraded] > 0 {
		return analyzer.StatusDegraded
	}
	if statusCounts[analyzer.StatusPending] > 0 {
		return analyzer.StatusPending
	}
	return analyzer.StatusHealthy
}

func (a *Advisor) GenerateReport(result *analyzer.AnalysisResult) string {
	var report strings.Builder

	// Summary
	report.WriteString("🏥 CLUSTER HEALTH REPORT\n")
	report.WriteString(strings.Repeat("=", 50) + "\n\n")

	healthIcon := a.getHealthIcon(result.Summary.ClusterHealth)
	report.WriteString(fmt.Sprintf("Overall Health: %s %s\n\n", healthIcon, result.Summary.ClusterHealth))

	// Component summary
	report.WriteString("📊 COMPONENT SUMMARY\n")
	report.WriteString(fmt.Sprintf("Total Components: %d\n", result.Summary.TotalComponents))
	report.WriteString("Status Distribution:\n")
	for status, count := range result.Summary.StatusCounts {
		if count > 0 {
			icon := a.getStatusIcon(status)
			report.WriteString(fmt.Sprintf("  %s %s: %d\n", icon, status, count))
		}
	}
	report.WriteString("\n")

	// Issues
	if len(result.Issues) == 0 {
		report.WriteString("✅ No issues found! All components are healthy.\n")
	} else {
		report.WriteString("🚨 ISSUES FOUND\n")
		report.WriteString(strings.Repeat("-", 30) + "\n")

		for i, issue := range result.Issues {
			report.WriteString(fmt.Sprintf("\n%d. %s %s\n", i+1, a.getSeverityIcon(issue.Severity), issue.Description))
			report.WriteString(fmt.Sprintf("   Component: %s/%s\n", issue.Component.Type, issue.Component.Name))
			report.WriteString(fmt.Sprintf("   Cause: %s\n", issue.Cause))
			report.WriteString(fmt.Sprintf("   💡 Resolution: %s\n", issue.Resolution))

			if len(issue.Dependencies) > 0 {
				report.WriteString("   🔗 Check these dependencies:\n")
				for _, dep := range issue.Dependencies {
					depStatus := a.getStatusIcon(dep.Status)
					report.WriteString(fmt.Sprintf("      %s %s/%s\n", depStatus, dep.Type, dep.Name))
				}
			}
		}
	}

	return report.String()
}

func (a *Advisor) getHealthIcon(status analyzer.ComponentStatus) string {
	return a.getStatusIcon(status)
}

func (a *Advisor) getStatusIcon(status analyzer.ComponentStatus) string {
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

func (a *Advisor) getSeverityIcon(severity analyzer.ConditionSeverity) string {
	switch severity {
	case analyzer.SeverityCritical:
		return "🔴"
	case analyzer.SeverityWarning:
		return "🟡"
	case analyzer.SeverityInfo:
		return "🔵"
	default:
		return "⚪"
	}
}