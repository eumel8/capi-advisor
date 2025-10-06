package cmd

import (
	"context"
	"fmt"

	"capi-advisor/pkg/client"
	"capi-advisor/pkg/orphaned"

	"github.com/spf13/cobra"
)

var (
	orphanedNamespace string
	orphanedDryRun    bool
)

var orphanedCmd = &cobra.Command{
	Use:   "orphaned",
	Short: "Find and clean up orphaned Metal3 resources",
	Long: `Find and optionally clean up orphaned Metal3 resources that block re-provisioning.

This command identifies:
- Metal3DataClaims without valid ownerRef to Metal3Machine
- Metal3Data without valid claimRef to Metal3DataClaim
- Secrets related to Metal3 without proper ownerReferences

Examples:
  # Find orphaned resources (dry-run)
  capi-advisor orphaned -n cluster-namespace

  # Clean up orphaned resources
  capi-advisor orphaned -n cluster-namespace --delete

  # Find orphaned resources in all namespaces
  capi-advisor orphaned --all-namespaces`,
	RunE: runOrphaned,
}

func init() {
	orphanedCmd.Flags().StringVarP(&orphanedNamespace, "namespace", "n", "", "Namespace to check for orphaned resources")
	orphanedCmd.Flags().BoolVar(&orphanedDryRun, "delete", false, "Delete orphaned resources (default is dry-run)")
}

func runOrphaned(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Kubernetes client
	k8sClient, err := client.NewK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	if orphanedNamespace == "" {
		return fmt.Errorf("namespace is required, use -n flag")
	}

	fmt.Printf("Scanning namespace '%s' for orphaned resources...\n\n", orphanedNamespace)

	// Create orphaned resource finder
	finder := orphaned.NewFinder(k8sClient, orphanedNamespace)

	// Find orphaned resources
	results, err := finder.FindOrphaned(ctx)
	if err != nil {
		return fmt.Errorf("failed to find orphaned resources: %w", err)
	}

	// Display results
	displayResults(results, orphanedDryRun)

	// Clean up if requested
	if orphanedDryRun {
		fmt.Println("\nDeleting orphaned resources...")
		if err := finder.CleanupOrphaned(ctx, results); err != nil {
			return fmt.Errorf("failed to clean up orphaned resources: %w", err)
		}
		fmt.Println("✓ Cleanup completed")
	} else {
		fmt.Println("\nDry-run mode: No resources were deleted.")
		fmt.Println("Use --delete flag to actually remove orphaned resources.")
	}

	return nil
}

func displayResults(results *orphaned.OrphanedResults, willDelete bool) {
	action := "kubectl delete -n"
	if willDelete {
		action = "Deleting"
	}

	fmt.Println("=== Orphaned Metal3DataClaims ===")
	if len(results.Metal3DataClaims) == 0 {
		fmt.Println("None found")
	} else {
		for _, claim := range results.Metal3DataClaims {
			if willDelete {
				fmt.Printf("%s %s: %s\n", action, results.Namespace, claim)
			} else {
				fmt.Printf("%s %s metal3dataclaim/%s\n", action, results.Namespace, claim)
			}
		}
	}

	fmt.Println("\n=== Orphaned Metal3Data ===")
	if len(results.Metal3Data) == 0 {
		fmt.Println("None found")
	} else {
		for _, data := range results.Metal3Data {
			if willDelete {
				fmt.Printf("%s %s: %s\n", action, results.Namespace, data)
			} else {
				fmt.Printf("%s %s metal3data/%s\n", action, results.Namespace, data)
			}
		}
	}

	fmt.Println("\n=== Orphaned Secrets ===")
	if len(results.Secrets) == 0 {
		fmt.Println("None found")
	} else {
		for _, secret := range results.Secrets {
			if willDelete {
				fmt.Printf("%s %s: %s\n", action, results.Namespace, secret)
			} else {
				fmt.Printf("%s %s secret/%s\n", action, results.Namespace, secret)
			}
		}
	}

	total := len(results.Metal3DataClaims) + len(results.Metal3Data) + len(results.Secrets)
	fmt.Printf("\nTotal orphaned resources: %d\n", total)
}
