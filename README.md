# CAPI Advisor

A CLI tool to analyze Cluster API and Metal3 cluster components, check their conditions, build dependency trees, and provide clear advice on how to resolve any issues found.

## Features

- **Comprehensive Component Discovery**: Automatically discovers all Cluster API and Metal3 components in your cluster
- **Condition Analysis**: Analyzes all component conditions and identifies issues
- **Dependency Tree Building**: Builds hierarchical dependency relationships between components
- **Intelligent Advisory System**: Provides specific recommendations for resolving issues
- **Multiple Output Formats**: Supports human-readable reports, JSON, and YAML output
- **Focused Health Diagnostics**: Dedicated doctor mode for quick health checks

## Supported Components

- **Cluster API Core**: Clusters, Machines, MachineSets, MachineDeployments
- **Control Plane**: KubeadmControlPlane, KubeadmConfig
- **Metal3 Infrastructure**: Metal3Cluster, Metal3Machine, BareMetalHost

## Installation

```bash
go build -o capi-advisor .
```

## Usage

### Comprehensive Analysis

Get a full analysis with recommendations for all components:

```bash
# Analyze all components across all namespaces
./capi-advisor analyze

# Analyze components in a specific namespace
./capi-advisor analyze -n cluster-system

# Analyze a specific CAPI cluster
./capi-advisor analyze -c my-cluster

# Show dependency tree along with analysis
./capi-advisor analyze --tree

# Get results in JSON format
./capi-advisor analyze -o json
```

### Health Diagnostics

Focus on health issues and their solutions:

```bash
# Run health diagnostics
./capi-advisor doctor

# Check specific namespace
./capi-advisor doctor -n cluster-system

# Check specific CAPI cluster
./capi-advisor doctor -c my-cluster
```

### Dependency Tree View

Visualize component relationships:

```bash
# Show component dependency tree
./capi-advisor tree

# Focus on specific namespace
./capi-advisor tree -n cluster-system

# Show tree for specific CAPI cluster
./capi-advisor tree -c my-cluster
```

## Examples

### Example Output - Health Report

```
🏥 CLUSTER HEALTH REPORT
==================================================

Overall Health: ✅ Healthy

📊 COMPONENT SUMMARY
Total Components: 12
Status Distribution:
  ✅ Healthy: 10
  ⚠️ Degraded: 2
  ❌ Failed: 0

🚨 ISSUES FOUND
------------------------------

1. 🟡 Machine InfrastructureReady is False
   Component: Machine/worker-1
   Cause: Metal3Machine is not ready
   💡 Resolution: Check BareMetalHost status and provisioning
   🔗 Check these dependencies:
      ⏳ Metal3Machine/worker-1-metal3
      ❌ BareMetalHost/worker-1-bmh

2. 🟡 BareMetalHost Available is False
   Component: BareMetalHost/worker-1-bmh
   Cause: Host is not available for provisioning
   💡 Resolution: Check if host is powered on and BMC is accessible
```

### Example Output - Dependency Tree

```
🌳 COMPONENT DEPENDENCY TREE
============================
✅ Cluster/test-cluster (Healthy)
  ✓ Ready: Cluster is ready
  ✓ InfrastructureReady: Infrastructure is ready
  ✅ Metal3Cluster/test-cluster-metal3 (Healthy)
    ✓ Ready: Metal3Cluster is ready
  ✅ KubeadmControlPlane/test-cluster-control-plane (Healthy)
    ✓ Ready: Control plane is ready
    ✅ Machine/test-cluster-control-plane-abc123 (Healthy)
      ✓ Ready: Machine is ready
      ✅ Metal3Machine/test-cluster-control-plane-abc123-metal3 (Healthy)
        ✓ Ready: Metal3Machine is ready
        ✅ BareMetalHost/master-0 (Healthy)
          ✓ Ready: Host is ready and provisioned
```

## Architecture

The tool is structured into several key packages:

- `pkg/client`: Kubernetes client configuration and management
- `pkg/analyzer`: Component discovery and condition analysis
- `pkg/tree`: Dependency tree building and relationship mapping
- `pkg/advisor`: Knowledge base and issue resolution recommendations
- `cmd`: CLI commands and user interface

## Configuration

The tool uses your existing Kubernetes configuration:

1. In-cluster configuration (when running as a pod)
2. `~/.kube/config` file
3. `$KUBECONFIG` environment variable

## Contributing

This tool is designed to be extensible. To add support for new component types:

1. Add the component type to `SupportedGVKs` in `pkg/analyzer/discovery.go`
2. Add relationship logic in `pkg/tree/builder.go`
3. Add condition knowledge to the advisor in `pkg/advisor/advisor.go`

## Alternates

### clusterctl

from [Git Repo](https://github.com/kubernetes-sigs/cluster-api/releases)

example:

<details>
```bash
$ clusterctl describe cluster my-cluster-1 -n metal3-my-infra --show-conditions all
NAME                                                READY  SEVERITY  REASON                                                          SINCE  MESSAGE                       
Cluster/my-cluster-1                                  True                                                                             31s                                  
│           ├─ControlPlaneInitialized               True                                                                             15m                                  
│           ├─ControlPlaneReady                     True                                                                             31s                                  
│           └─InfrastructureReady                   True                                                                             31m                                  
├─ClusterInfrastructure - Metal3Cluster/my-cluster-1  True                                                                             31m                                  
│             └─BaremetalInfrastructureReady        True                                                                             31m                                  
├─ControlPlane - KubeadmControlPlane/my-cluster-1     True                                                                             31s                                  
│ │           ├─Available                           True                                                                             15m                                  
│ │           ├─CertificatesAvailable               True                                                                             31m                                  
│ │           ├─ControlPlaneComponentsHealthy       True                                                                             66s                                  
│ │           ├─EtcdClusterHealthy                  True                                                                             72s                                  
│ │           ├─MachinesCreated                     True                                                                             65s                                  
│ │           ├─MachinesReady                       True                                                                             31s                                  
│ │           └─Resized                             True                                                                             64s                                  
│ └─3 Machines...                                   True                                                                             15m    See my-cluster-1-ftxxh, my-cluster-1-tdh8j, ...
└─Workers                                                                                                                                                                 
  ├─MachineDeployment/my-cluster-1-default-v3         False  Warning   WaitingForAvailableMachines                                     31m    Minimum availability requires 2 replicas, current 0 available
  │ │           ├─Available                         False  Warning   WaitingForAvailableMachines                                     31m    Minimum availability requires 2 replicas, current 0 available
  │ │           └─MachineSetReady                   False  Error     AssociateBMHFailed @ Machine/my-cluster-1-default-v3-b8qqb-67w5p  41s    0 of 2 completed              
  │ ├─Machine/my-cluster-1-default-v3-b8qqb-67w5p     False  Error     AssociateBMHFailed                                              15m    1 of 2 completed              
  │ │             ├─BootstrapReady                  True                                                                             15m                                  
  │ │             ├─InfrastructureReady             False  Error     AssociateBMHFailed                                              15m    No available host found. Requeuing.. Object will be requeued after 30s
  │ │             └─NodeHealthy                     False  Info      WaitingForNodeRef                                               31m                                  
  │ └─Machine/my-cluster-1-default-v3-b8qqb-x79q4     False  Error     SettingProviderIDOnNodeFailed                                   41s    1 of 2 completed              
  │               ├─BootstrapReady                  True                                                                             15m                                  
  │               ├─InfrastructureReady             False  Error     SettingProviderIDOnNodeFailed                                   41s    Some target nodes do not have spec.providerID field set yet, requeuing. Object will be requeued afte ...
  │               └─NodeHealthy                     False  Info      WaitingForNodeRef                                               31m                                  
  └─MachineDeployment/my-cluster-1-sriov              False  Warning   WaitingForAvailableMachines                                     31m    Minimum availability requires 2 replicas, current 0 available
    │           ├─Available                         False  Warning   WaitingForAvailableMachines                                     31m    Minimum availability requires 2 replicas, current 0 available
    │           └─MachineSetReady                   False  Warning   ScalingUp                                                       13s    Scaling up MachineSet to 2 replicas (actual 0)
    └─2 Machines...                               
```
</details>

### kubectl tree

from [Git Repo](https://github.com/ahmetb/kubectl-tree)

example:

<details>
```bash
$ kubectl tree -n metal3-my-infra cluster my-cluster-1
NAMESPACE         NAME                                                                                 READY  REASON                                                          AGE
metal3-my-infra  Cluster/my-cluster-1                                                                   True                                                                   34m
metal3-my-infra  ├─KubeadmConfigTemplate/my-cluster-1-default-v3-2                                      -                                                                      34m
metal3-my-infra  ├─KubeadmConfigTemplate/my-cluster-1-sriov-7                                           -                                                                      34m
metal3-my-infra  ├─KubeadmControlPlane/my-cluster-1                                                     True                                                                   34m
metal3-my-infra  │ ├─Machine/my-cluster-1-ftxxh           
...
```
</details>

## License

This project is licensed under the MIT License.
