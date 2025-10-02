package cmd

import (
	"context"
	"fmt"

	"capi-advisor/pkg/advisor"
	"capi-advisor/pkg/analyzer"
	"capi-advisor/pkg/client"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run cluster health diagnostics",
	Long: `Run comprehensive health diagnostics on Cluster API and Metal3 components.
This command focuses on identifying and providing solutions for issues.`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace to analyze (empty for all namespaces)")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fmt.Println("🏥 Running cluster health diagnostics...")
	fmt.Println("======================================")

	// Create Kubernetes client
	k8sClient, err := client.NewK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Get cluster info
	clusterInfo, err := k8sClient.GetClusterInfo(ctx)
	if err != nil {
		fmt.Printf("⚠️  Warning: could not get cluster info: %v\n", err)
	} else {
		fmt.Printf("📡 Cluster: %s\n", clusterInfo)
	}

	// Discover components
	discovery := analyzer.NewComponentDiscovery(k8sClient.Client)
	components, err := discovery.DiscoverComponents(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to discover components: %v", err)
	}

	if len(components) == 0 {
		fmt.Println("\n✅ No Cluster API or Metal3 components found - nothing to diagnose")
		return nil
	}

	// Analyze components
	advisor := advisor.NewAdvisor()
	result := advisor.AnalyzeComponents(components)

	// Generate focused health report
	fmt.Printf("\n🔍 Analyzed %d components\n", len(components))

	if len(result.Issues) == 0 {
		fmt.Println("\n🎉 Excellent! No issues found.")
		fmt.Println("All Cluster API and Metal3 components are healthy.")
	} else {
		fmt.Printf("\n🚨 Found %d issue(s) that need attention:\n", len(result.Issues))

		for i, issue := range result.Issues {
			severityIcon := getSeverityIcon(issue.Severity)
			fmt.Printf("\n%d. %s %s\n", i+1, severityIcon, issue.Description)
			fmt.Printf("   📍 Component: %s/%s (namespace: %s)\n",
				issue.Component.Type, issue.Component.Name, issue.Component.Namespace)

			if issue.Condition.Message != "" {
				fmt.Printf("   📝 Message: %s\n", issue.Condition.Message)
			}

			fmt.Printf("   🔍 Cause: %s\n", issue.Cause)
			fmt.Printf("   💡 Resolution: %s\n", issue.Resolution)

			if len(issue.Dependencies) > 0 {
				fmt.Println("   🔗 Dependencies to check:")
				for _, dep := range issue.Dependencies {
					depStatus := getStatusIcon(dep.Status)
					fmt.Printf("      %s %s/%s\n", depStatus, dep.Type, dep.Name)
				}
			}
		}

		fmt.Printf("\n📊 Summary by severity:\n")
		for severity, count := range result.Summary.SeverityCounts {
			if count > 0 {
				icon := getSeverityIcon(severity)
				fmt.Printf("   %s %s: %d\n", icon, severity, count)
			}
		}
	}

	return nil
}

func getSeverityIcon(severity analyzer.ConditionSeverity) string {
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

func getStatusIcon(status analyzer.ComponentStatus) string {
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