# CAPI Advisor GUI

The CAPI Advisor now includes a graphical user interface (GUI) for visualizing cluster state and resources.

## Features

The GUI provides three main views:

### 1. Overview Tab
- **Cluster Statistics**: Total components and overall cluster health
- **Status Breakdown**: Count of components by status (Healthy, Degraded, Failed, Pending, Unknown)
- **Severity Breakdown**: Count of issues by severity (Critical, Warning, Info)
- **Component List**: Interactive list of all discovered components with their status

### 2. Component Tree Tab
- **Dependency Visualization**: Interactive tree view showing component relationships
- **Component Details**: Click any component to view detailed information including:
  - Name, namespace, and type
  - Current status
  - All conditions with their status, reason, and messages
  - Child components

### 3. Issues & Recommendations Tab
- **Categorized Issues**: Issues grouped by severity (Critical, Warning, Info)
- **Detailed Information**: For each issue:
  - Component information
  - Condition details
  - Description and possible cause
  - Recommended actions to resolve
  - Affected dependencies

## System Requirements

To build and run the GUI, you need the following system dependencies:

### Linux (Debian/Ubuntu)
```bash
sudo apt-get install -y pkg-config libgl1-mesa-dev xorg-dev
```

### Linux (Fedora/RHEL)
```bash
sudo dnf install -y pkg-config mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
```

### macOS
```bash
# Xcode command line tools are required
xcode-select --install
```

### Windows
No additional dependencies required. Go includes the necessary build tools.

## Building

Once system dependencies are installed:

```bash
go build -o capi-advisor
```

## Usage

Launch the GUI with:

```bash
./capi-advisor gui
```

### Options

- `-n, --namespace <namespace>`: Analyze components in a specific namespace (default: all namespaces)
- `-c, --cluster <cluster-name>`: Analyze a specific CAPI cluster (default: all clusters)

### Examples

```bash
# Analyze all clusters in all namespaces
./capi-advisor gui

# Analyze a specific namespace
./capi-advisor gui -n cluster-system

# Analyze a specific cluster
./capi-advisor gui -c my-cluster

# Analyze a specific cluster in a specific namespace
./capi-advisor gui -n cluster-system -c my-cluster
```

## GUI Framework

The GUI is built using [Fyne](https://fyne.io/), a cross-platform GUI toolkit for Go that provides:
- Native look and feel on each platform
- Support for Linux, macOS, Windows
- Responsive and modern UI components
- Easy deployment without external dependencies (once built)

## CLI Alternative

If you prefer the command-line interface or cannot install GUI dependencies, all functionality remains available through the CLI commands:

```bash
# Text-based analysis
./capi-advisor analyze

# JSON output for scripting
./capi-advisor analyze -o json

# Show component tree in terminal
./capi-advisor analyze --tree
```

## Troubleshooting

### Build Errors

If you encounter build errors related to missing system libraries:

1. Verify system dependencies are installed (see System Requirements above)
2. Run `go mod tidy` to ensure all Go dependencies are properly resolved
3. Try cleaning the build cache: `go clean -cache`

### Runtime Errors

If the GUI fails to start:

1. Check that you have a display server running (X11 or Wayland on Linux)
2. Ensure you have proper permissions to access the Kubernetes cluster
3. Verify your kubeconfig is properly configured: `kubectl get nodes`

### "No components found" Message

If the GUI shows no components:

1. Verify Cluster API is installed in your cluster
2. Check the namespace selection (try without `-n` flag to scan all namespaces)
3. Run the CLI version to debug: `./capi-advisor analyze -v`
