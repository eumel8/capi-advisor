package gui

import (
	"context"
	"fmt"

	"capi-advisor/pkg/advisor"
	"capi-advisor/pkg/analyzer"
	"capi-advisor/pkg/client"
	"capi-advisor/pkg/tree"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type App struct {
	fyneApp        fyne.App
	window         fyne.Window
	k8sClient      *client.K8sClient
	components     []*analyzer.Component
	analysisResult *analyzer.AnalysisResult
	treeBuilder    *tree.TreeBuilder
	rootComponents []*analyzer.Component
}

func NewApp() *App {
	return &App{
		fyneApp: app.New(),
		treeBuilder: tree.NewTreeBuilder(),
	}
}

func (a *App) Run(namespace, clusterName string) error {
	a.window = a.fyneApp.NewWindow("CAPI Advisor - Cluster Visualization")
	a.window.Resize(fyne.NewSize(1200, 800))

	// Show loading screen
	loading := widget.NewLabel("Loading cluster data...")
	a.window.SetContent(container.NewCenter(loading))
	a.window.Show()

	// Load data in background
	go func() {
		if err := a.loadClusterData(namespace, clusterName); err != nil {
			a.showError(err)
			return
		}
		a.setupUI()
	}()

	a.fyneApp.Run()
	return nil
}

func (a *App) loadClusterData(namespace, clusterName string) error {
	ctx := context.Background()

	// Create Kubernetes client
	k8sClient, err := client.NewK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}
	a.k8sClient = k8sClient

	// Discover components
	discovery := analyzer.NewComponentDiscovery(k8sClient.Client)
	components, err := discovery.DiscoverComponents(ctx, namespace, clusterName)
	if err != nil {
		return fmt.Errorf("failed to discover components: %v", err)
	}
	a.components = components

	// Build dependency tree
	a.rootComponents = a.treeBuilder.BuildDependencyTree(components)

	// Analyze components
	adv := advisor.NewAdvisor()
	a.analysisResult = adv.AnalyzeComponents(components)

	return nil
}

func (a *App) setupUI() {
	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Overview", a.createOverviewTab()),
		container.NewTabItem("Component Tree", a.createTreeTab()),
		container.NewTabItem("Issues & Recommendations", a.createIssuesTab()),
	)


	fyne.DoAndWait(func() {
		a.window.SetContent(tabs)
	})
}

func (a *App) showError(err error) {
	errorLabel := widget.NewLabel(fmt.Sprintf("Error: %v", err))
	a.window.SetContent(container.NewCenter(errorLabel))
}
