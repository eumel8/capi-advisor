package gui

import (
	"fmt"
	"strings"

	"capi-advisor/pkg/analyzer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (a *App) createTreeTab() fyne.CanvasObject {
	if a.rootComponents == nil || len(a.rootComponents) == 0 {
		return widget.NewLabel("No component tree data available")
	}

	title := widget.NewLabelWithStyle("Component Dependency Tree", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Create tree widget
	treeWidget := a.buildTreeWidget()

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		treeWidget,
	)

	return container.NewScroll(content)
}

func (a *App) buildTreeWidget() fyne.CanvasObject {
	// Build tree structure
	treeData := make(map[string][]string)
	treeComponents := make(map[string]*analyzer.Component)

	var buildTree func(comp *analyzer.Component, parentID string)
	buildTree = func(comp *analyzer.Component, parentID string) {
		if comp == nil {
			return
		}

		nodeID := fmt.Sprintf("%s/%s/%s", comp.Namespace, comp.Type, comp.Name)
		treeComponents[nodeID] = comp

		if parentID != "" {
			treeData[parentID] = append(treeData[parentID], nodeID)
		}

		// Process children
		for _, child := range comp.Children {
			buildTree(child, nodeID)
		}
	}

	// Build from root components
	var rootIDs []string
	for _, root := range a.rootComponents {
		rootID := fmt.Sprintf("%s/%s/%s", root.Namespace, root.Type, root.Name)
		rootIDs = append(rootIDs, rootID)
		treeComponents[rootID] = root
		buildTree(root, "")
	}

	// Create Fyne tree widget
	tree := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			if uid == "" {
				return rootIDs
			}
			return treeData[uid]
		},
		func(uid widget.TreeNodeID) bool {
			children := treeData[uid]
			return len(children) > 0
		},
		func(branch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.DocumentIcon()),
				widget.NewLabel("Component"),
			)
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			comp := treeComponents[uid]
			if comp == nil {
				return
			}

			box := obj.(*fyne.Container)
			icon := box.Objects[0].(*widget.Icon)
			label := box.Objects[1].(*widget.Label)

			// Set icon based on component type
			switch comp.Type {
			case analyzer.ClusterType:
				icon.SetResource(theme.HomeIcon())
			case analyzer.MachineType:
				icon.SetResource(theme.ComputerIcon())
			default:
				icon.SetResource(theme.DocumentIcon())
			}

			// Set label with status color
			labelText := fmt.Sprintf("%s: %s", comp.Type, comp.Name)
			label.SetText(labelText)
		},
	)

	// Detail view on selection
	detailLabel := widget.NewLabel("Select a component to view details")
	detailLabel.Wrapping = fyne.TextWrapWord

	tree.OnSelected = func(uid widget.TreeNodeID) {
		comp := treeComponents[uid]
		if comp != nil {
			details := a.formatComponentDetails(comp)
			detailLabel.SetText(details)
		}
	}

	split := container.NewHSplit(
		container.NewBorder(nil, nil, nil, nil, tree),
		container.NewScroll(detailLabel),
	)
	split.SetOffset(0.6)

	return split
}

func (a *App) formatComponentDetails(comp *analyzer.Component) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Component Details\n"))
	sb.WriteString(strings.Repeat("=", 50))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("Name: %s\n", comp.Name))
	sb.WriteString(fmt.Sprintf("Namespace: %s\n", comp.Namespace))
	sb.WriteString(fmt.Sprintf("Type: %s\n", comp.Type))
	sb.WriteString(fmt.Sprintf("Status: %s\n\n", comp.Status))

	if len(comp.Conditions) > 0 {
		sb.WriteString("Conditions:\n")
		sb.WriteString(strings.Repeat("-", 50))
		sb.WriteString("\n")
		for _, cond := range comp.Conditions {
			sb.WriteString(fmt.Sprintf("  • %s: %s\n", cond.Type, cond.Status))
			if cond.Reason != "" {
				sb.WriteString(fmt.Sprintf("    Reason: %s\n", cond.Reason))
			}
			if cond.Message != "" {
				sb.WriteString(fmt.Sprintf("    Message: %s\n", cond.Message))
			}
			sb.WriteString("\n")
		}
	}

	if len(comp.Children) > 0 {
		sb.WriteString(fmt.Sprintf("\nChildren: %d components\n", len(comp.Children)))
		for _, child := range comp.Children {
			sb.WriteString(fmt.Sprintf("  • %s: %s\n", child.Type, child.Name))
		}
	}

	return sb.String()
}
