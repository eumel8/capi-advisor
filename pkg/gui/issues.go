package gui

import (
	"fmt"
	"strings"

	"capi-advisor/pkg/analyzer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) createIssuesTab() fyne.CanvasObject {
	if a.analysisResult == nil {
		return widget.NewLabel("No analysis data available")
	}

	title := widget.NewLabelWithStyle("Issues & Recommendations", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	var content *fyne.Container

	if len(a.analysisResult.Issues) == 0 {
		noIssues := widget.NewLabelWithStyle(
			"✓ No issues found! All components are healthy.",
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		)
		content = container.NewVBox(
			title,
			widget.NewSeparator(),
			container.NewCenter(noIssues),
		)
	} else {
		// Group issues by severity
		criticalIssues := []*analyzer.Issue{}
		warningIssues := []*analyzer.Issue{}
		infoIssues := []*analyzer.Issue{}

		for _, issue := range a.analysisResult.Issues {
			switch issue.Severity {
			case analyzer.SeverityCritical:
				criticalIssues = append(criticalIssues, issue)
			case analyzer.SeverityWarning:
				warningIssues = append(warningIssues, issue)
			case analyzer.SeverityInfo:
				infoIssues = append(infoIssues, issue)
			}
		}

		issuesContent := container.NewVBox()

		// Add critical issues
		if len(criticalIssues) > 0 {
			issuesContent.Add(a.createIssueSection("Critical Issues", criticalIssues, analyzer.SeverityCritical))
			issuesContent.Add(widget.NewSeparator())
		}

		// Add warnings
		if len(warningIssues) > 0 {
			issuesContent.Add(a.createIssueSection("Warnings", warningIssues, analyzer.SeverityWarning))
			issuesContent.Add(widget.NewSeparator())
		}

		// Add info
		if len(infoIssues) > 0 {
			issuesContent.Add(a.createIssueSection("Information", infoIssues, analyzer.SeverityInfo))
		}

		content = container.NewVBox(
			title,
			widget.NewSeparator(),
			issuesContent,
		)
	}

	return container.NewScroll(content)
}

func (a *App) createIssueSection(title string, issues []*analyzer.Issue, severity analyzer.ConditionSeverity) fyne.CanvasObject {
	header := container.NewHBox(
		a.createSeverityLabel(severity),
		widget.NewLabelWithStyle(fmt.Sprintf(" %s (%d)", title, len(issues)), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	issuesList := container.NewVBox()

	for i, issue := range issues {
		issueCard := a.createIssueCard(issue, i+1)
		issuesList.Add(issueCard)
		if i < len(issues)-1 {
			issuesList.Add(widget.NewSeparator())
		}
	}

	return container.NewVBox(
		header,
		issuesList,
	)
}

func (a *App) createIssueCard(issue *analyzer.Issue, index int) fyne.CanvasObject {
	// Component info
	componentInfo := widget.NewLabelWithStyle(
		fmt.Sprintf("#%d - %s: %s/%s",
			index,
			issue.Component.Type,
			issue.Component.Namespace,
			issue.Component.Name,
		),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	// Condition info
	conditionInfo := widget.NewLabel(fmt.Sprintf("Condition: %s - Status: %s",
		issue.Condition.Type,
		issue.Condition.Status,
	))

	// Description
	var descParts []string
	if issue.Description != "" {
		descParts = append(descParts, fmt.Sprintf("Description: %s", issue.Description))
	}
	if issue.Condition.Reason != "" {
		descParts = append(descParts, fmt.Sprintf("Reason: %s", issue.Condition.Reason))
	}
	if issue.Condition.Message != "" {
		descParts = append(descParts, fmt.Sprintf("Message: %s", issue.Condition.Message))
	}

	description := widget.NewLabel(strings.Join(descParts, "\n"))
	description.Wrapping = fyne.TextWrapWord

	// Cause
	var causeWidget fyne.CanvasObject
	if issue.Cause != "" {
		causeLabel := widget.NewLabelWithStyle("Possible Cause:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
		causeText := widget.NewLabel(issue.Cause)
		causeText.Wrapping = fyne.TextWrapWord
		causeWidget = container.NewVBox(causeLabel, causeText)
	}

	// Resolution
	var resolutionWidget fyne.CanvasObject
	if issue.Resolution != "" {
		resolutionLabel := widget.NewLabelWithStyle("Recommended Action:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		resolutionText := widget.NewLabel(issue.Resolution)
		resolutionText.Wrapping = fyne.TextWrapWord
		resolutionWidget = container.NewVBox(resolutionLabel, resolutionText)
	}

	// Dependencies
	var depsWidget fyne.CanvasObject
	if len(issue.Dependencies) > 0 {
		depsLabel := widget.NewLabel(fmt.Sprintf("Affected dependencies (%d):", len(issue.Dependencies)))
		depsList := container.NewVBox()
		for _, dep := range issue.Dependencies {
			depText := widget.NewLabel(fmt.Sprintf("  • %s: %s", dep.Type, dep.Name))
			depsList.Add(depText)
		}
		depsWidget = container.NewVBox(depsLabel, depsList)
	}

	// Combine all parts
	card := container.NewVBox(
		componentInfo,
		conditionInfo,
		description,
	)

	if causeWidget != nil {
		card.Add(causeWidget)
	}
	if resolutionWidget != nil {
		card.Add(resolutionWidget)
	}
	if depsWidget != nil {
		card.Add(depsWidget)
	}

	// Add border/padding effect
	return container.NewPadded(card)
}
