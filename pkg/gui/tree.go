package gui

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"capi-advisor/pkg/analyzer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) createTreeTab() fyne.CanvasObject {
	if a.components == nil || len(a.components) == 0 {
		return widget.NewLabel("No component data available")
	}

	title := widget.NewLabelWithStyle("Component Dependency Graph", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Create spider/grid visualization widget
	graphWidget := a.buildGraphWidget()

	content := container.NewBorder(
		container.NewVBox(title, widget.NewSeparator()),
		nil,
		nil,
		nil,
		graphWidget,
	)

	return content
}

// GraphNode represents a node in the dependency graph
type GraphNode struct {
	Component *analyzer.Component
	X, Y      float32
	Rect      *canvas.Rectangle
	Label     *canvas.Text
	StatusDot *canvas.Circle
}

func (a *App) buildGraphWidget() fyne.CanvasObject {
	// Create the graph visualization
	graph := NewDependencyGraph(a.components)

	// Detail view on selection
	detailLabel := widget.NewLabel("Click on a component to view details")
	detailLabel.Wrapping = fyne.TextWrapWord

	// Create detail scroll container
	detailScroll := container.NewScroll(detailLabel)
	detailScroll.SetMinSize(fyne.NewSize(350, 400))

	// Set up callback for node selection
	graph.OnNodeSelected = func(comp *analyzer.Component) {
		if comp != nil {
			details := a.formatComponentDetails(comp)
			detailLabel.SetText(details)
			detailLabel.Refresh()
		}
	}

	split := container.NewHSplit(
		container.NewScroll(graph),
		detailScroll,
	)
	split.SetOffset(0.65)

	return split
}

// DependencyGraph is a custom widget that renders components as a spider/grid graph
type DependencyGraph struct {
	widget.BaseWidget
	components     []*analyzer.Component
	nodes          map[*analyzer.Component]*GraphNode
	OnNodeSelected func(*analyzer.Component)
	selectedNode   *analyzer.Component
}

func NewDependencyGraph(components []*analyzer.Component) *DependencyGraph {
	g := &DependencyGraph{
		components: components,
		nodes:      make(map[*analyzer.Component]*GraphNode),
	}
	g.ExtendBaseWidget(g)
	g.layoutNodes()
	return g
}

func (g *DependencyGraph) CreateRenderer() fyne.WidgetRenderer {
	return &dependencyGraphRenderer{
		graph:   g,
		objects: []fyne.CanvasObject{},
	}
}

func (g *DependencyGraph) layoutNodes() {
	if len(g.components) == 0 {
		return
	}

	// Organize components by type/level for grid layout
	levels := make(map[analyzer.ComponentType][]*analyzer.Component)
	for _, comp := range g.components {
		levels[comp.Type] = append(levels[comp.Type], comp)
	}

	// Define level order (top to bottom)
	levelOrder := []analyzer.ComponentType{
		analyzer.ClusterType,
		analyzer.Metal3ClusterType,
		analyzer.KubeadmControlPlaneType,
		analyzer.MachineDeploymentType,
		analyzer.MachineSetType,
		analyzer.MachineType,
		analyzer.Metal3MachineType,
		analyzer.BareMetalHostType,
		analyzer.KubeadmConfigType,
	}

	// Calculate positions in grid
	horizontalSpacing := float32(180)
	verticalSpacing := float32(100)

	startX := float32(50)
	startY := float32(50)

	currentY := startY

	for _, compType := range levelOrder {
		comps := levels[compType]
		if len(comps) == 0 {
			continue
		}

		// Calculate horizontal centering
		totalWidth := float32(len(comps)) * horizontalSpacing
		_ = startX + (totalWidth / 2) - (horizontalSpacing / 2)

		for i, comp := range comps {
			x := startX + float32(i)*horizontalSpacing
			y := currentY

			// Create node
			rect := canvas.NewRectangle(g.getStatusColor(comp.Status))
			rect.StrokeColor = color.RGBA{R: 100, G: 100, B: 100, A: 255}
			rect.StrokeWidth = 2

			labelText := fmt.Sprintf("%s\n%s", comp.Type, comp.Name)
			label := canvas.NewText(labelText, color.Black)
			label.Alignment = fyne.TextAlignCenter
			label.TextSize = 10

			// Status indicator dot
			statusDot := canvas.NewCircle(g.getConditionColor(comp))

			node := &GraphNode{
				Component: comp,
				X:         x,
				Y:         y,
				Rect:      rect,
				Label:     label,
				StatusDot: statusDot,
			}

			g.nodes[comp] = node
		}

		currentY += verticalSpacing
	}
}

func (g *DependencyGraph) getStatusColor(status analyzer.ComponentStatus) color.Color {
	switch status {
	case analyzer.StatusHealthy:
		return color.RGBA{R: 200, G: 255, B: 200, A: 255}
	case analyzer.StatusDegraded:
		return color.RGBA{R: 255, G: 255, B: 180, A: 255}
	case analyzer.StatusFailed:
		return color.RGBA{R: 255, G: 200, B: 200, A: 255}
	case analyzer.StatusPending:
		return color.RGBA{R: 220, G: 220, B: 255, A: 255}
	default:
		return color.RGBA{R: 240, G: 240, B: 240, A: 255}
	}
}

func (g *DependencyGraph) getConditionColor(comp *analyzer.Component) color.Color {
	// Determine overall condition health
	hasError := false
	hasWarning := false

	for _, cond := range comp.Conditions {
		if cond.Status == "False" && (cond.Type == "Ready" || cond.Type == "Available") {
			hasError = true
		} else if cond.Status == "Unknown" {
			hasWarning = true
		}
	}

	if hasError {
		return color.RGBA{R: 255, G: 50, B: 50, A: 255}
	}
	if hasWarning {
		return color.RGBA{R: 255, G: 200, B: 50, A: 255}
	}
	return color.RGBA{R: 50, G: 200, B: 50, A: 255}
}

func (g *DependencyGraph) Tapped(ev *fyne.PointEvent) {
	// Find which node was tapped
	for _, node := range g.nodes {
		nodeWidth := float32(150)
		nodeHeight := float32(60)

		if ev.Position.X >= node.X && ev.Position.X <= node.X+nodeWidth &&
			ev.Position.Y >= node.Y && ev.Position.Y <= node.Y+nodeHeight {
			g.selectedNode = node.Component
			if g.OnNodeSelected != nil {
				g.OnNodeSelected(node.Component)
			}
			g.Refresh()
			break
		}
	}
}

func (g *DependencyGraph) MinSize() fyne.Size {
	// Calculate minimum size based on node positions
	maxX := float32(0)
	maxY := float32(0)

	for _, node := range g.nodes {
		if node.X > maxX {
			maxX = node.X
		}
		if node.Y > maxY {
			maxY = node.Y
		}
	}

	return fyne.NewSize(maxX+300, maxY+200)
}

// Renderer implementation
type dependencyGraphRenderer struct {
	graph   *DependencyGraph
	objects []fyne.CanvasObject
}

func (r *dependencyGraphRenderer) Layout(size fyne.Size) {
	nodeWidth := float32(150)
	nodeHeight := float32(60)
	dotSize := float32(10)

	// Clear and rebuild objects
	r.objects = []fyne.CanvasObject{}

	// Draw dependency lines first (so they appear behind nodes)
	for _, node := range r.graph.nodes {
		if node.Component.Parent != nil {
			parentNode, exists := r.graph.nodes[node.Component.Parent]
			if exists {
				line := r.createDependencyLine(
					parentNode.X+nodeWidth/2, parentNode.Y+nodeHeight,
					node.X+nodeWidth/2, node.Y,
					node.Component,
				)
				r.objects = append(r.objects, line...)
			}
		}

		// Also draw lines to children for visibility
		for _, child := range node.Component.Children {
			childNode, exists := r.graph.nodes[child]
			if exists {
				line := r.createDependencyLine(
					node.X+nodeWidth/2, node.Y+nodeHeight,
					childNode.X+nodeWidth/2, childNode.Y,
					child,
				)
				r.objects = append(r.objects, line...)
			}
		}
	}

	// Draw nodes on top
	for _, node := range r.graph.nodes {
		// Position rectangle
		node.Rect.Move(fyne.NewPos(node.X, node.Y))
		node.Rect.Resize(fyne.NewSize(nodeWidth, nodeHeight))

		// Highlight if selected
		if r.graph.selectedNode == node.Component {
			node.Rect.StrokeWidth = 4
			node.Rect.StrokeColor = color.RGBA{R: 0, G: 100, B: 255, A: 255}
		} else {
			node.Rect.StrokeWidth = 2
			node.Rect.StrokeColor = color.RGBA{R: 100, G: 100, B: 100, A: 255}
		}

		// Position label (centered in rectangle)
		node.Label.Move(fyne.NewPos(node.X+10, node.Y+10))
		node.Label.Resize(fyne.NewSize(nodeWidth-20, nodeHeight-20))

		// Position status dot (top-right corner)
		node.StatusDot.Move(fyne.NewPos(node.X+nodeWidth-dotSize-5, node.Y+5))
		node.StatusDot.Resize(fyne.NewSize(dotSize, dotSize))

		r.objects = append(r.objects, node.Rect, node.Label, node.StatusDot)
	}
}

func (r *dependencyGraphRenderer) createDependencyLine(x1, y1, x2, y2 float32, childComp *analyzer.Component) []fyne.CanvasObject {
	// Create a curved line using multiple line segments
	objects := []fyne.CanvasObject{}

	// Determine line color based on child component's condition state
	lineColor := color.RGBA{R: 150, G: 150, B: 150, A: 200}
	lineWidth := float32(2)

	// Check if child has any failed conditions
	for _, cond := range childComp.Conditions {
		if cond.Status == "False" && (cond.Type == "Ready" || cond.Type == "Available") {
			lineColor = color.RGBA{R: 255, G: 100, B: 100, A: 200}
			lineWidth = 3
			break
		}
	}

	// Create bezier curve for more organic look
	segments := 20
	for i := 0; i < segments; i++ {
		t1 := float32(i) / float32(segments)
		t2 := float32(i+1) / float32(segments)

		// Calculate control points for bezier curve
		midY := (y1 + y2) / 2
		cx1, cy1 := x1, midY
		cx2, cy2 := x2, midY

		// Calculate points on bezier curve
		p1x := r.bezier(x1, cx1, cx2, x2, t1)
		p1y := r.bezier(y1, cy1, cy2, y2, t1)
		p2x := r.bezier(x1, cx1, cx2, x2, t2)
		p2y := r.bezier(y1, cy1, cy2, y2, t2)

		line := canvas.NewLine(lineColor)
		line.StrokeWidth = lineWidth
		line.Position1 = fyne.NewPos(p1x, p1y)
		line.Position2 = fyne.NewPos(p2x, p2y)

		objects = append(objects, line)
	}

	// Add arrow head at the end
	arrowSize := float32(8)
	angle := float32(math.Atan2(float64(y2-y1), float64(x2-x1)))

	arrow1 := canvas.NewLine(lineColor)
	arrow1.StrokeWidth = lineWidth
	arrow1.Position1 = fyne.NewPos(x2, y2)
	arrow1.Position2 = fyne.NewPos(
		x2-arrowSize*float32(math.Cos(float64(angle-0.5))),
		y2-arrowSize*float32(math.Sin(float64(angle-0.5))),
	)

	arrow2 := canvas.NewLine(lineColor)
	arrow2.StrokeWidth = lineWidth
	arrow2.Position1 = fyne.NewPos(x2, y2)
	arrow2.Position2 = fyne.NewPos(
		x2-arrowSize*float32(math.Cos(float64(angle+0.5))),
		y2-arrowSize*float32(math.Sin(float64(angle+0.5))),
	)

	objects = append(objects, arrow1, arrow2)

	return objects
}

func (r *dependencyGraphRenderer) bezier(p0, p1, p2, p3, t float32) float32 {
	// Cubic bezier curve formula
	u := 1 - t
	return u*u*u*p0 + 3*u*u*t*p1 + 3*u*t*t*p2 + t*t*t*p3
}

func (r *dependencyGraphRenderer) MinSize() fyne.Size {
	return r.graph.MinSize()
}

func (r *dependencyGraphRenderer) Refresh() {
	canvas.Refresh(r.graph)
}

func (r *dependencyGraphRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *dependencyGraphRenderer) Destroy() {}

func (g *DependencyGraph) TappedSecondary(_ *fyne.PointEvent) {}

func (g *DependencyGraph) DoubleTapped(_ *fyne.PointEvent) {}

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
