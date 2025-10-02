package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"capi-advisor/pkg/advisor"
	"capi-advisor/pkg/analyzer"
	"capi-advisor/pkg/client"
	"capi-advisor/pkg/tree"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	namespace    string
	clusterName  string
	outputFormat string
	showTree     bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze Cluster API and Metal3 components",
	Long: `Analyze all Cluster API and Metal3 components in the cluster,
check their conditions, build dependency trees, and provide recommendations
for resolving any issues.`,
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace to analyze (empty for all namespaces)")
	analyzeCmd.Flags().StringVarP(&clusterName, "cluster", "c", "", "CAPI cluster name to analyze (empty for all clusters)")
	analyzeCmd.Flags().StringVarP(&outputFormat, "output", "o", "report", "Output format: report, json, yaml")
	analyzeCmd.Flags().BoolVar(&showTree, "tree", false, "Show component dependency tree")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Kubernetes client
	fmt.Println("🔗 Connecting to Kubernetes cluster...")
	k8sClient, err := client.NewK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Get cluster info
	clusterInfo, err := k8sClient.GetClusterInfo(ctx)
	if err != nil {
		fmt.Printf("Warning: could not get cluster info: %v\n", err)
	} else {
		fmt.Printf("📡 Connected to %s\n\n", clusterInfo)
	}

	// Discover components
	fmt.Println("🔍 Discovering Cluster API and Metal3 components...")
	discovery := analyzer.NewComponentDiscovery(k8sClient.Client)
	components, err := discovery.DiscoverComponents(ctx, namespace, clusterName)
	if err != nil {
		return fmt.Errorf("failed to discover components: %v", err)
	}

	if len(components) == 0 {
		fmt.Println("ℹ️  No Cluster API or Metal3 components found in the specified namespace")
		return nil
	}

	fmt.Printf("✅ Found %d components\n\n", len(components))

	// Build dependency tree
	fmt.Println("🌳 Building component dependency tree...")
	treeBuilder := tree.NewTreeBuilder()
	rootComponents := treeBuilder.BuildDependencyTree(components)

	// Analyze components
	fmt.Println("🔬 Analyzing component conditions...")
	advisor := advisor.NewAdvisor()
	result := advisor.AnalyzeComponents(components)

	// Output results
	switch outputFormat {
	case "json":
		return outputJSON(result)
	case "yaml":
		return outputYAML(result)
	case "report":
		fallthrough
	default:
		report := advisor.GenerateReport(result)
		fmt.Print(report)

		if showTree {
			fmt.Println("\n🌳 COMPONENT DEPENDENCY TREE")
			fmt.Println(strings.Repeat("=", 50))
			tree := treeBuilder.PrintTree(rootComponents)
			fmt.Print(tree)
		}
	}

	return nil
}

func outputJSON(result *analyzer.AnalysisResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputYAML(result *analyzer.AnalysisResult) error {
	data, err := yaml.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}