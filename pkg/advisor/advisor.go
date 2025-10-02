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
		Resolution: "1. Check if InfrastructureReady condition is True\n   2. Verify ControlPlaneReady condition is True\n   3. Inspect Metal3Cluster and KubeadmControlPlane resources\n   4. Review cluster events: kubectl describe cluster <name>",
		Dependencies: []string{"Metal3Cluster", "KubeadmControlPlane"},
	}

	a.knowledgeBase["Cluster.InfrastructureReady.False"] = KnowledgeEntry{
		Condition:  "Cluster InfrastructureReady is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Infrastructure provider is not ready",
		Resolution: "1. Check Metal3Cluster resource: kubectl describe metal3cluster <name>\n   2. Verify network configuration in Metal3Cluster spec\n   3. Check infrastructure provider controller logs\n   4. Ensure required networks (provisioning, external) are configured",
		Dependencies: []string{"Metal3Cluster"},
	}

	a.knowledgeBase["Cluster.ControlPlaneReady.False"] = KnowledgeEntry{
		Condition:  "Cluster ControlPlaneReady is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Control plane nodes are not ready",
		Resolution: "1. Check KubeadmControlPlane status: kubectl describe kcp <name>\n   2. Verify control plane replicas are scheduled\n   3. Check control plane Machine resources status\n   4. Review etcd pod logs if cluster is partially up\n   5. Check for sufficient control plane nodes matching desired replicas",
		Dependencies: []string{"KubeadmControlPlane", "Machine"},
	}

	// Machine conditions
	a.knowledgeBase["Machine.Ready.False"] = KnowledgeEntry{
		Condition:  "Machine Ready is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Machine infrastructure or bootstrap is not ready",
		Resolution: "1. Check Machine status: kubectl describe machine <name>\n   2. Verify InfrastructureReady condition status\n   3. Check BootstrapReady condition status\n   4. Review Metal3Machine and KubeadmConfig resources\n   5. Check node status if partially provisioned",
		Dependencies: []string{"Metal3Machine", "KubeadmConfig"},
	}

	a.knowledgeBase["Machine.InfrastructureReady.False"] = KnowledgeEntry{
		Condition:  "Machine InfrastructureReady is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Metal3Machine is not ready",
		Resolution: "1. Check Metal3Machine: kubectl describe metal3machine <name>\n   2. Verify BareMetalHost association and status\n   3. Check if BareMetalHost is in 'provisioned' state\n   4. Review BMC credentials and connectivity\n   5. Check baremetal-operator logs for provisioning errors",
		Dependencies: []string{"Metal3Machine", "BareMetalHost"},
	}

	a.knowledgeBase["Machine.BootstrapReady.False"] = KnowledgeEntry{
		Condition:  "Machine BootstrapReady is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Bootstrap configuration is not ready",
		Resolution: "1. Check KubeadmConfig: kubectl describe kubeadmconfig <name>\n   2. For control plane: verify API server is accessible\n   3. For workers: ensure control plane is ready\n   4. Check cluster connectivity and certificates\n   5. Review bootstrap provider controller logs",
		Dependencies: []string{"KubeadmConfig"},
	}

	// Metal3Machine conditions
	a.knowledgeBase["Metal3Machine.Ready.False"] = KnowledgeEntry{
		Condition:  "Metal3Machine Ready is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "BareMetalHost is not available or not provisioned",
		Resolution: "1. Check Metal3Machine: kubectl describe metal3machine <name>\n   2. Verify BareMetalHost binding and status\n   3. Check BareMetalHost state (should be 'provisioned')\n   4. Test BMC connectivity: ipmitool -H <bmc-ip> -U <user> -P <pass> power status\n   5. Review image URL and ensure it's accessible\n   6. Check baremetal-operator controller logs",
		Dependencies: []string{"BareMetalHost"},
	}

	a.knowledgeBase["Metal3Machine.AssociationReady.False"] = KnowledgeEntry{
		Condition:  "Metal3Machine AssociationReady is False",
		Severity:   analyzer.SeverityWarning,
		Cause:      "Unable to associate with BareMetalHost",
		Resolution: "1. Check hostSelector labels in Metal3Machine spec\n   2. List available BareMetalHosts: kubectl get bmh -A\n   3. Verify BareMetalHost labels match hostSelector\n   4. Ensure BareMetalHost is not already claimed by another machine\n   5. Check if sufficient available hosts exist for provisioning",
		Dependencies: []string{"BareMetalHost"},
	}

	// BareMetalHost conditions
	a.knowledgeBase["BareMetalHost.Ready.False"] = KnowledgeEntry{
		Condition:  "BareMetalHost Ready is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Hardware is not available or provisioning failed",
		Resolution: "1. Check BareMetalHost: kubectl describe bmh <name> -n <namespace>\n   2. Test BMC connectivity from baremetal-operator pod\n   3. Verify BMC credentials in secret\n   4. Check provisioning state and error messages\n   5. Ensure provisioning image is accessible\n   6. Review hardware compatibility and RAID configuration\n   7. Check Ironic logs for detailed provisioning errors",
		Dependencies: []string{},
	}

	a.knowledgeBase["BareMetalHost.Available.False"] = KnowledgeEntry{
		Condition:  "BareMetalHost Available is False",
		Severity:   analyzer.SeverityWarning,
		Cause:      "Host is not available for provisioning",
		Resolution: "1. Check if host is powered on: kubectl get bmh <name> -o jsonpath='{.status.poweredOn}'\n   2. Test BMC accessibility from cluster network\n   3. Verify BMC credentials are correct\n   4. Check hardware inspection status\n   5. Review operationalStatus and errorMessage fields\n   6. Ensure host is not in maintenance mode",
		Dependencies: []string{},
	}

	a.knowledgeBase["BareMetalHost.Provisioned.False"] = KnowledgeEntry{
		Condition:  "BareMetalHost Provisioned is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Provisioning process failed or is in progress",
		Resolution: "1. Check provisioning state: kubectl get bmh <name> -o jsonpath='{.status.provisioning.state}'\n   2. Review provisioning error message in status\n   3. Verify image URL is accessible from provisioning network\n   4. Check disk format and partitioning settings\n   5. Ensure sufficient disk space for image\n   6. Review Ironic deployment and agent logs\n   7. Check network connectivity during provisioning",
		Dependencies: []string{},
	}

	// KubeadmControlPlane conditions
	a.knowledgeBase["KubeadmControlPlane.Ready.False"] = KnowledgeEntry{
		Condition:  "KubeadmControlPlane Ready is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Control plane nodes are not ready",
		Resolution: "1. Check KubeadmControlPlane: kubectl describe kcp <name>\n   2. List control plane machines: kubectl get machines -l cluster.x-k8s.io/control-plane\n   3. Check machine readiness and node status\n   4. Verify desired vs ready replicas count\n   5. Review etcd health if cluster is accessible\n   6. Check control plane provider controller logs\n   7. Ensure kubeconfig secret exists for workload cluster",
		Dependencies: []string{"Machine"},
	}

	a.knowledgeBase["KubeadmControlPlane.Initialized.False"] = KnowledgeEntry{
		Condition:  "KubeadmControlPlane Initialized is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Control plane initialization failed",
		Resolution: "1. Check first control plane machine status\n   2. Review kubeadm init logs on first control plane node\n   3. Verify bootstrap configuration in KubeadmControlPlane spec\n   4. Check if certificates were generated correctly\n   5. Ensure control plane endpoint is configured\n   6. Review cloud-init logs on control plane node\n   7. Verify network connectivity for API server",
		Dependencies: []string{"Machine"},
	}

	a.knowledgeBase["KubeadmControlPlane.CertificatesAvailable.False"] = KnowledgeEntry{
		Condition:  "KubeadmControlPlane CertificatesAvailable is False",
		Severity:   analyzer.SeverityCritical,
		Cause:      "Control plane certificates are not available",
		Resolution: "1. Check if cluster-certificates secret exists\n   2. Verify certificate generation in first control plane node\n   3. Review kubeadm certificate commands output\n   4. Check bootstrap provider logs for errors\n   5. Ensure control plane has completed initialization",
		Dependencies: []string{"Machine"},
	}

	// KubeadmConfig conditions
	a.knowledgeBase["KubeadmConfig.Ready.False"] = KnowledgeEntry{
		Condition:  "KubeadmConfig Ready is False",
		Severity:   analyzer.SeverityWarning,
		Cause:      "Bootstrap configuration is not ready",
		Resolution: "1. Check KubeadmConfig: kubectl describe kubeadmconfig <name>\n   2. Verify bootstrap data secret was created\n   3. For workers: ensure control plane is ready and accessible\n   4. Check cluster connectivity and certificate validity\n   5. Review bootstrap provider controller logs\n   6. Verify join configuration is correct",
		Dependencies: []string{},
	}

	a.knowledgeBase["KubeadmConfig.DataSecretAvailable.False"] = KnowledgeEntry{
		Condition:  "KubeadmConfig DataSecretAvailable is False",
		Severity:   analyzer.SeverityWarning,
		Cause:      "Bootstrap data secret has not been generated",
		Resolution: "1. Check if bootstrap data secret exists\n   2. Verify KubeadmConfig reconciliation status\n   3. Ensure control plane is accessible for worker nodes\n   4. Review bootstrap provider controller logs\n   5. Check for any errors in KubeadmConfig status",
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
					Cause:       a.enhanceCause(knowledge.Cause, condition),
					Resolution:  a.enhanceResolution(knowledge.Resolution, condition, comp),
				}

				// Find dependency components
				issue.Dependencies = a.findDependencies(comp, knowledge.Dependencies)
				issues = append(issues, issue)
			} else {
				// Enhanced generic issue for unknown conditions
				issue := &analyzer.Issue{
					Component:   comp,
					Condition:   condition,
					Severity:    analyzer.SeverityWarning,
					Description: fmt.Sprintf("%s %s is %s", comp.Type, condition.Type, condition.Status),
					Cause:       a.buildGenericCause(condition),
					Resolution:  a.buildGenericResolution(comp, condition),
				}
				issues = append(issues, issue)
			}
		}
	}

	return issues
}

func (a *Advisor) enhanceCause(baseCause string, condition metav1.Condition) string {
	if condition.Reason != "" && condition.Message != "" {
		return fmt.Sprintf("%s\nReason: %s\nDetails: %s", baseCause, condition.Reason, condition.Message)
	} else if condition.Reason != "" {
		return fmt.Sprintf("%s\nReason: %s", baseCause, condition.Reason)
	} else if condition.Message != "" {
		return fmt.Sprintf("%s\nDetails: %s", baseCause, condition.Message)
	}
	return baseCause
}

func (a *Advisor) enhanceResolution(baseResolution string, condition metav1.Condition, comp *analyzer.Component) string {
	resolution := baseResolution

	// Add specific guidance based on reason
	specificGuidance := a.getSpecificGuidanceFromReason(condition.Reason, condition.Message, comp)
	if specificGuidance != "" {
		resolution = fmt.Sprintf("%s\n\n💡 Specific guidance based on current state:\n%s", resolution, specificGuidance)
	}

	return resolution
}

func (a *Advisor) getSpecificGuidanceFromReason(reason string, message string, comp *analyzer.Component) string {
	reasonLower := strings.ToLower(reason)
	messageLower := strings.ToLower(message)

	// BMC/IPMI related issues
	if strings.Contains(reasonLower, "bmc") || strings.Contains(messageLower, "ipmi") ||
	   strings.Contains(messageLower, "bmc") || strings.Contains(reasonLower, "connection") {
		return "   - BMC connection issue detected. Verify:\n     * BMC IP address is reachable from baremetal-operator pod\n     * BMC credentials are correct in the secret\n     * Firewall rules allow IPMI traffic (port 623)\n     * BMC firmware is up to date"
	}

	// Provisioning image issues
	if strings.Contains(reasonLower, "image") || strings.Contains(messageLower, "image") ||
	   strings.Contains(messageLower, "download") || strings.Contains(messageLower, "http") {
		return "   - Image access issue detected. Verify:\n     * Image URL is accessible from provisioning network\n     * HTTP server hosting the image is running\n     * Image checksum matches if specified\n     * Sufficient disk space on target host"
	}

	// Network/connectivity issues
	if strings.Contains(reasonLower, "timeout") || strings.Contains(messageLower, "timeout") ||
	   strings.Contains(messageLower, "connection refused") || strings.Contains(messageLower, "network") {
		return "   - Network connectivity issue detected. Verify:\n     * Network connectivity between components\n     * DNS resolution is working\n     * No firewall blocking required ports\n     * Check for network policy restrictions"
	}

	// Certificate issues
	if strings.Contains(reasonLower, "certificate") || strings.Contains(messageLower, "certificate") ||
	   strings.Contains(messageLower, "tls") || strings.Contains(messageLower, "x509") {
		return "   - Certificate issue detected. Verify:\n     * Certificates are not expired\n     * Certificate chain is complete\n     * CA bundle is correctly configured\n     * System time is synchronized (NTP)"
	}

	// Insufficient resources
	if strings.Contains(reasonLower, "insufficient") || strings.Contains(messageLower, "insufficient") ||
	   strings.Contains(messageLower, "no available") || strings.Contains(messageLower, "quota") {
		return "   - Resource availability issue detected. Verify:\n     * Sufficient BareMetalHosts are available\n     * Hosts meet the required specifications\n     * No resource quotas are being exceeded\n     * Check cluster capacity and node resources"
	}

	// Authentication/authorization issues
	if strings.Contains(reasonLower, "auth") || strings.Contains(messageLower, "auth") ||
	   strings.Contains(messageLower, "permission") || strings.Contains(messageLower, "forbidden") {
		return "   - Authentication/authorization issue detected. Verify:\n     * Service account has correct permissions\n     * RBAC roles and bindings are configured\n     * Secrets contain valid credentials\n     * API server is accessible"
	}

	// Waiting for dependencies
	if strings.Contains(reasonLower, "waiting") || strings.Contains(messageLower, "waiting") ||
	   strings.Contains(reasonLower, "pending") {
		return "   - Waiting for dependencies. Check:\n     * All prerequisite resources are ready\n     * Dependencies are not blocked\n     * Review the full component hierarchy\n     * Check for circular dependencies"
	}

	return ""
}

func (a *Advisor) buildGenericCause(condition metav1.Condition) string {
	if condition.Reason != "" && condition.Message != "" {
		return fmt.Sprintf("Reason: %s\nDetails: %s", condition.Reason, condition.Message)
	} else if condition.Reason != "" {
		return fmt.Sprintf("Reason: %s", condition.Reason)
	} else if condition.Message != "" {
		return fmt.Sprintf("Details: %s", condition.Message)
	}
	return "Unknown cause - no additional information available"
}

func (a *Advisor) buildGenericResolution(comp *analyzer.Component, condition metav1.Condition) string {
	resolution := fmt.Sprintf("1. Check %s resource: kubectl describe %s %s -n %s\n   2. Review the condition message and reason above\n   3. Check controller logs for this resource type\n   4. Review recent events: kubectl get events -n %s --field-selector involvedObject.name=%s",
		comp.Type, strings.ToLower(string(comp.Type)), comp.Name, comp.Namespace, comp.Namespace, comp.Name)

	// Add specific guidance based on reason/message
	specificGuidance := a.getSpecificGuidanceFromReason(condition.Reason, condition.Message, comp)
	if specificGuidance != "" {
		resolution = fmt.Sprintf("%s\n\n💡 Specific guidance based on error:\n%s", resolution, specificGuidance)
	}

	return resolution
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