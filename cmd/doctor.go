package cmd

import (
	"context"
	"fmt"

	"capi-advisor/pkg/advisor"
	"capi-advisor/pkg/analyzer"
	"capi-advisor/pkg/client"
	"capi-advisor/pkg/orphaned"

	"github.com/spf13/cobra"
)

var doctorDelete bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run cluster health diagnostics",
	Long: `Run comprehensive health diagnostics on Cluster API and Metal3 components.
This command focuses on identifying and providing solutions for issues.`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace to analyze (empty for all namespaces)")
	doctorCmd.Flags().StringVarP(&clusterName, "cluster", "c", "", "CAPI cluster name to analyze (empty for all clusters)")
	doctorCmd.Flags().BoolVar(&doctorDelete, "delete", false, "Delete interfering resources found during diagnostics")
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
	components, err := discovery.DiscoverComponents(ctx, namespace, clusterName)
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

	// Interference check — requires a specific namespace
	if namespace != "" {
		fmt.Println("\n🔧 Checking for interference...")
		finder := orphaned.NewFinder(k8sClient, namespace)
		orphanedResults, err := finder.FindOrphaned(ctx)
		if err != nil {
			fmt.Printf("⚠️  Warning: could not check for interference: %v\n", err)
		} else {
			totalOrphaned := len(orphanedResults.Metal3DataClaims) + len(orphanedResults.Metal3Data) + len(orphanedResults.Secrets)
			totalMismatches := len(orphanedResults.SecretOwnerMismatches)

			if totalOrphaned == 0 && totalMismatches == 0 {
				fmt.Println("✅ No interference detected")
			} else {
				if totalOrphaned > 0 {
					fmt.Printf("\n⚠️  Found %d orphaned resource(s):\n", totalOrphaned)
					for _, c := range orphanedResults.Metal3DataClaims {
						fmt.Printf("   • Metal3DataClaim/%s\n", c)
					}
					for _, d := range orphanedResults.Metal3Data {
						fmt.Printf("   • Metal3Data/%s\n", d)
					}
					for _, s := range orphanedResults.Secrets {
						fmt.Printf("   • Secret/%s\n", s)
					}
				}
				if totalMismatches > 0 {
					fmt.Printf("\n⚠️  Found %d Metal3Machine secret ownerRef mismatch(es):\n", totalMismatches)
					for _, m := range orphanedResults.SecretOwnerMismatches {
						fmt.Printf("   🔴 Secret/%s (Metal3Machine: %s): %s\n", m.SecretName, m.Metal3Machine, m.Reason)
					}
				}

				if doctorDelete {
					fmt.Println("\n🗑️  Cleaning up interference...")
					if err := finder.CleanupOrphaned(ctx, orphanedResults); err != nil {
						fmt.Printf("⚠️  Warning: cleanup encountered errors: %v\n", err)
					} else {
						fmt.Println("✅ Interference cleaned up")
					}
				} else {
					fmt.Println("\nUse --delete to remove interfering resources.")
				}
			}
		}
	} else {
		fmt.Println("\n💡 Tip: specify -n <namespace> to also check for interference (orphaned resources, secret ownerRef mismatches).")
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