package main

import (
	"fmt"
	"os"

	"capi-advisor/cmd"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "capi-advisor",
	Short: "Analyze Cluster API and Metal3 components and provide recommendations",
	Long: `A CLI tool to analyze Cluster API and Metal3 cluster components,
check their conditions, build dependency trees, and provide clear advice
on how to resolve any issues found.

Available commands:
  analyze  - Comprehensive analysis with recommendations
  doctor   - Focus on health diagnostics and issue resolution
  tree     - Show component dependency relationships

Examples:
  # Analyze all components and get recommendations
  capi-advisor analyze

  # Check health of components in a specific namespace
  capi-advisor doctor -n cluster-system

  # Show component dependency tree
  capi-advisor tree

  # Get detailed analysis as JSON
  capi-advisor analyze -o json`,
}

func init() {
	rootCmd.AddCommand(cmd.AnalyzeCmd)
	rootCmd.AddCommand(cmd.DoctorCmd)
	rootCmd.AddCommand(cmd.TreeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}