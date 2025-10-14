package gui

import (
	"fmt"
	"image/color"

	"capi-advisor/pkg/analyzer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (a *App) createOverviewTab() fyne.CanvasObject {
	if a.analysisResult == nil {
		return widget.NewLabel("No data available")
	}

	// Title
	title := widget.NewLabelWithStyle("Cluster Overview", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Summary statistics
	summary := a.analysisResult.Summary
	statsGrid := a.createStatsGrid(summary)

	// Status breakdown
	statusBreakdown := a.createStatusBreakdown(summary)

	// Severity breakdown
	severityBreakdown := a.createSeverityBreakdown(summary)

	// Component list
	componentList := a.createComponentList()

	// Layout
	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		statsGrid,
		widget.NewSeparator(),
		container.NewHBox(
			statusBreakdown,
			widget.NewSeparator(),
			severityBreakdown,
		),
		widget.NewSeparator(),
		widget.NewLabel("Components:"),
		componentList,
	)

	return container.NewScroll(content)
}

func (a *App) createStatsGrid(summary analyzer.Summary) fyne.CanvasObject {
	totalLabel := widget.NewLabel("Total Components:")
	totalValue := widget.NewLabelWithStyle(fmt.Sprintf("%d", summary.TotalComponents), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	healthLabel := widget.NewLabel("Cluster Health:")
	healthValue := a.createStatusLabel(summary.ClusterHealth)

	grid := container.NewGridWithColumns(4,
		totalLabel, totalValue,
		healthLabel, healthValue,
	)

	return container.NewVBox(
		widget.NewLabelWithStyle("Statistics", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		grid,
	)
}

func (a *App) createStatusBreakdown(summary analyzer.Summary) fyne.CanvasObject {
	content := container.NewVBox(
		widget.NewLabelWithStyle("Status Breakdown", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	for status, count := range summary.StatusCounts {
		statusLabel := a.createStatusLabel(status)
		countLabel := widget.NewLabel(fmt.Sprintf(": %d", count))
		row := container.NewHBox(statusLabel, countLabel)
		content.Add(row)
	}

	return content
}

func (a *App) createSeverityBreakdown(summary analyzer.Summary) fyne.CanvasObject {
	content := container.NewVBox(
		widget.NewLabelWithStyle("Issue Severity", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	if len(summary.SeverityCounts) == 0 {
		content.Add(widget.NewLabel("No issues found"))
	} else {
		for severity, count := range summary.SeverityCounts {
			severityLabel := a.createSeverityLabel(severity)
			countLabel := widget.NewLabel(fmt.Sprintf(": %d", count))
			row := container.NewHBox(severityLabel, countLabel)
			content.Add(row)
		}
	}

	return content
}

func (a *App) createComponentList() fyne.CanvasObject {
	list := widget.NewList(
		func() int {
			return len(a.components)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Type"),
				widget.NewLabel("Name"),
				widget.NewLabel("Status"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			comp := a.components[id]
			box := obj.(*fyne.Container)

			typeLabel := box.Objects[0].(*widget.Label)
			nameLabel := box.Objects[1].(*widget.Label)

			typeLabel.SetText(string(comp.Type))
			nameLabel.SetText(fmt.Sprintf("%s/%s", comp.Namespace, comp.Name))

			// Replace status label with colored version
			box.Objects[2] = a.createStatusLabel(comp.Status)
		},
	)

	return container.NewVBox(
		list,
	)
}

func (a *App) createStatusLabel(status analyzer.ComponentStatus) fyne.CanvasObject {
	var textColor color.Color
	switch status {
	case analyzer.StatusHealthy:
		textColor = color.NRGBA{R: 0, G: 150, B: 0, A: 255} // Green
	case analyzer.StatusDegraded:
		textColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255} // Orange
	case analyzer.StatusFailed:
		textColor = color.NRGBA{R: 200, G: 0, B: 0, A: 255} // Red
	case analyzer.StatusPending:
		textColor = color.NRGBA{R: 100, G: 100, B: 255, A: 255} // Blue
	default:
		textColor = theme.ForegroundColor()
	}

	label := canvas.NewText(string(status), textColor)
	label.TextStyle.Bold = true
	return container.NewHBox(label)
}

func (a *App) createSeverityLabel(severity analyzer.ConditionSeverity) fyne.CanvasObject {
	var textColor color.Color
	switch severity {
	case analyzer.SeverityCritical:
		textColor = color.NRGBA{R: 200, G: 0, B: 0, A: 255} // Red
	case analyzer.SeverityWarning:
		textColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255} // Orange
	case analyzer.SeverityInfo:
		textColor = color.NRGBA{R: 100, G: 100, B: 255, A: 255} // Blue
	default:
		textColor = theme.ForegroundColor()
	}

	label := canvas.NewText(string(severity), textColor)
	label.TextStyle.Bold = true
	return container.NewHBox(label)
}
