package cmd

import (
	"capi-advisor/pkg/gui"

	"github.com/spf13/cobra"
)

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Launch graphical user interface for cluster visualization",
	Long: `Launch a GUI application to visualize Cluster API and Metal3 components,
their dependencies, health status, and get recommendations for resolving issues.

The GUI provides:
  - Interactive overview of cluster health and component statistics
  - Visual component dependency tree with detailed information
  - Organized view of issues with severity levels and recommendations
`,
	RunE: runGUI,
}

func init() {
	guiCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace to analyze (empty for all namespaces)")
	guiCmd.Flags().StringVarP(&clusterName, "cluster", "c", "", "CAPI cluster name to analyze (empty for all clusters)")
}

func runGUI(cmd *cobra.Command, args []string) error {
	app := gui.NewApp()
	return app.Run(namespace, clusterName)
}
