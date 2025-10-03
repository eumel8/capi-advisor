package cmd

import (
	"context"
	"fmt"

	"capi-advisor/pkg/analyzer"
	"capi-advisor/pkg/client"
	"capi-advisor/pkg/tree"

	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Show component dependency tree",
	Long: `Display the dependency tree of Cluster API and Metal3 components,
showing the hierarchical relationships between clusters, machines, and infrastructure.`,
	RunE: runTree,
}

func init() {
	treeCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace to analyze (empty for all namespaces)")
	treeCmd.Flags().StringVarP(&clusterName, "cluster", "c", "", "CAPI cluster name to analyze (empty for all clusters)")
}

func runTree(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Kubernetes client
	k8sClient, err := client.NewK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Discover components
	discovery := analyzer.NewComponentDiscovery(k8sClient.Client)
	components, err := discovery.DiscoverComponents(ctx, namespace, clusterName)
	if err != nil {
		return fmt.Errorf("failed to discover components: %v", err)
	}

	if len(components) == 0 {
		fmt.Println("ℹ️  No Cluster API or Metal3 components found")
		return nil
	}

	// Build and print dependency tree
	treeBuilder := tree.NewTreeBuilder()
	rootComponents := treeBuilder.BuildDependencyTree(components)

	fmt.Println("🌳 COMPONENT DEPENDENCY TREE")
	fmt.Println("============================")
	tree := treeBuilder.PrintTree(rootComponents)
	fmt.Print(tree)

	return nil
}