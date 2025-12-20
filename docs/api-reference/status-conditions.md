# Status Conditions Reference

This document describes the status conditions used by NSO Operator resources to communicate their current state and health.

## Overview

Status conditions provide a standardized way to understand the state of resources managed by the NSO Operator. Each condition has a type, status, reason, and message that help you understand what's happening with your resources.

## Common Condition Fields

All status conditions follow the standard Kubernetes pattern:

```yaml
status:
  conditions:
    - type: "Ready"
      status: "True"         # True, False, or Unknown
      reason: "NSO_Ready"    # Machine-readable reason
      message: "NSO instance is ready and accepting connections"
      lastTransitionTime: "2024-01-15T10:30:00Z"
```

### Field Descriptions

- **`type`**: The aspect of the resource being reported
- **`status`**: Current status - `"True"`, `"False"`, or `"Unknown"`
- **`reason`**: Machine-readable reason code (CamelCase)
- **`message`**: Human-readable description
- **`lastTransitionTime`**: When the condition last changed status

## NSO Resource Conditions

### Ready Condition

Indicates whether the NSO instance is ready and accepting connections.

| Status | Reason | Description |
|--------|--------|-------------|
| `True` | `NSO_Ready` | NSO is healthy and accepting connections |
| `False` | `NSO_NotReady` | NSO is not ready to accept connections |
| `False` | `ContainerNotReady` | NSO container is not ready |
| `False` | `ServiceUnavailable` | NSO service is not available |
| `Unknown` | `StatusUnknown` | NSO status cannot be determined |

**Examples:**
```yaml
# NSO ready and operational
- type: Ready
  status: "True"
  reason: "NSO_Ready"
  message: "NSO instance is ready and accepting connections"

# NSO starting up
- type: Ready
  status: "False"
  reason: "ContainerNotReady"
  message: "NSO container is starting, waiting for readiness probe"
```

### Progressing Condition

Indicates whether the NSO deployment is making progress.

| Status | Reason | Description |
|--------|--------|-------------|
| `True` | `DeploymentProgressing` | Deployment is making progress |
| `True` | `ReplicaSetUpdated` | ReplicaSet has been updated |
| `False` | `ProgressDeadlineExceeded` | Deployment progress deadline exceeded |
| `False` | `ReplicaFailure` | Some replicas failed to start |

**Examples:**
```yaml
# Deployment progressing normally
- type: Progressing
  status: "True"
  reason: "DeploymentProgressing"
  message: "ReplicaSet has successfully progressed"

# Deployment stuck
- type: Progressing
  status: "False"
  reason: "ProgressDeadlineExceeded"
  message: "ReplicaSet exceeded its progress deadline"
```

### Available Condition

Indicates whether the minimum number of NSO replicas are available.

| Status | Reason | Description |
|--------|--------|-------------|
| `True` | `MinimumReplicasAvailable` | Minimum replicas are available |
| `False` | `MinimumReplicasUnavailable` | Not enough replicas available |

**Examples:**
```yaml
# Sufficient replicas available
- type: Available
  status: "True"
  reason: "MinimumReplicasAvailable"
  message: "Deployment has minimum availability"

# Insufficient replicas
- type: Available
  status: "False"
  reason: "MinimumReplicasUnavailable" 
  message: "Deployment does not have minimum availability"
```

### StorageReady Condition

Indicates whether persistent storage is ready for the NSO instance.

| Status | Reason | Description |
|--------|--------|-------------|
| `True` | `StorageProvisioned` | Storage is provisioned and ready |
| `False` | `StoragePending` | Storage is still being provisioned |
| `False` | `StorageFailed` | Storage provisioning failed |

**Examples:**
```yaml
# Storage ready
- type: StorageReady
  status: "True"
  reason: "StorageProvisioned"
  message: "Persistent volume is bound and ready"

# Storage provisioning failed
- type: StorageReady
  status: "False" 
  reason: "StorageFailed"
  message: "Failed to provision storage: insufficient capacity"
```

## PackageBundle Resource Conditions

### Downloaded Condition

Indicates whether packages have been successfully downloaded from the source.

| Status | Reason | Description |
|--------|--------|-------------|
| `True` | `PackagesDownloaded` | All packages downloaded successfully |
| `False` | `DownloadFailed` | Package download failed |
| `False` | `SourceUnavailable` | Package source is not accessible |
| `Unknown` | `DownloadInProgress` | Download is currently in progress |

**Examples:**
```yaml
# Packages downloaded successfully
- type: Downloaded
  status: "True"
  reason: "PackagesDownloaded"
  message: "Successfully downloaded 15 packages from Git repository"

# Download failed due to authentication
- type: Downloaded
  status: "False"
  reason: "DownloadFailed"
  message: "Authentication failed: invalid SSH key"
```

### Installed Condition

Indicates whether packages have been installed in the target NSO instance.

| Status | Reason | Description |
|--------|--------|-------------|
| `True` | `PackagesInstalled` | Packages installed successfully |
| `False` | `InstallationFailed` | Package installation failed |
| `False` | `TargetUnavailable` | Target NSO instance unavailable |
| `Unknown` | `InstallationInProgress` | Installation in progress |

**Examples:**
```yaml
# Packages installed successfully
- type: Installed
  status: "True"
  reason: "PackagesInstalled"
  message: "All 15 packages installed successfully in NSO"

# Installation failed
- type: Installed
  status: "False"
  reason: "InstallationFailed"
  message: "Package 'cisco-iosxr-cli' failed to load: dependency missing"
```

### Validated Condition

Indicates whether package integrity and compatibility have been validated.

| Status | Reason | Description |
|--------|--------|-------------|
| `True` | `PackagesValid` | All packages are valid and compatible |
| `False` | `ValidationFailed` | Package validation failed |
| `False` | `IncompatibleVersion` | Package incompatible with NSO version |

**Examples:**
```yaml
# Packages validated successfully
- type: Validated
  status: "True"
  reason: "PackagesValid"
  message: "All packages are compatible with NSO 6.3.1"

# Validation failed
- type: Validated
  status: "False"
  reason: "IncompatibleVersion"
  message: "Package 'old-service' requires NSO 6.2.x, current version is 6.3.1"
```

## Common Status Patterns

### Healthy Resource

A healthy NSO resource typically shows:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: "NSO_Ready"
      message: "NSO instance is ready and accepting connections"
    - type: Available
      status: "True"
      reason: "MinimumReplicasAvailable"
      message: "Deployment has minimum availability"
    - type: Progressing
      status: "True"
      reason: "ReplicaSetUpdated"
      message: "ReplicaSet has successfully progressed"
```

### Resource in Transition

During updates or scaling:

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: "ContainerNotReady"
      message: "New replica is starting up"
    - type: Progressing
      status: "True"
      reason: "DeploymentProgressing"
      message: "ReplicaSet is being updated"
    - type: Available
      status: "True"
      reason: "MinimumReplicasAvailable"
      message: "Old replicas still available during update"
```

### Failed Resource

When something goes wrong:

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: "NSO_NotReady"
      message: "NSO failed to start: configuration error"
    - type: Progressing
      status: "False"
      reason: "ProgressDeadlineExceeded"
      message: "ReplicaSet exceeded its progress deadline"
    - type: Available
      status: "False"
      reason: "MinimumReplicasUnavailable"
      message: "No replicas are available"
```

## Monitoring Conditions

### kubectl Commands

Check resource conditions:

```bash
# Get NSO resource status
kubectl get nso my-nso -o jsonpath='{.status.conditions[*].type}'

# Get detailed condition information
kubectl get nso my-nso -o jsonpath='{range .status.conditions[*]}{.type}={.status} {.reason} {.message}{\"\\n\"}{end}'

# Check specific condition
kubectl get nso my-nso -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}'

# Get PackageBundle conditions
kubectl get packagebundle my-packages -o jsonpath='{range .status.conditions[*]}{.type}={.status} {.reason}{\"\\n\"}{end}'
```

### Using jq for Complex Queries

```bash
# Get all non-ready resources
kubectl get nso -o json | jq -r '.items[] | select(.status.conditions[]?.type=="Ready" and .status.conditions[]?.status=="False") | .metadata.name'

# Get condition summary
kubectl get nso my-nso -o json | jq -r '.status.conditions[] | \"\\(.type): \\(.status) (\\(.reason))\"'

# Get resources with failed conditions
kubectl get packagebundle -o json | jq -r '.items[] | select(.status.conditions[]?.status=="False") | .metadata.name'
```

### Prometheus Metrics

Monitor conditions using Prometheus:

```promql
# Count resources by condition status
sum by (type, status) (nso_operator_condition_status)

# Alert on non-ready resources
nso_operator_condition_status{type="Ready",status="False"} > 0

# Track condition transitions
increase(nso_operator_condition_transitions_total[5m])
```

## Best Practices

### Condition Interpretation

1. **Check Multiple Conditions**: Don't rely on a single condition
2. **Consider Transitions**: Look at `lastTransitionTime` to understand timing
3. **Read Messages**: Human-readable messages provide important context
4. **Monitor Trends**: Track condition changes over time

### Automation

1. **Health Checks**: Use conditions in health check scripts
2. **Alerting**: Set up alerts based on condition status
3. **GitOps**: Use conditions to determine deployment success
4. **Monitoring**: Include conditions in dashboards and metrics

### Troubleshooting Workflow

1. **Check Ready Condition**: Start with overall readiness
2. **Review Progressing**: Understand if changes are happening
3. **Verify Available**: Ensure minimum availability
4. **Read Messages**: Get specific error information
5. **Check Events**: Correlate with Kubernetes events

## Integration Examples

### Health Check Script

```bash
#!/bin/bash
# Check if NSO is ready
STATUS=$(kubectl get nso my-nso -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}')
if [ \"$STATUS\" = \"True\" ]; then
    echo \"NSO is ready\"
    exit 0
else
    echo \"NSO is not ready\"
    kubectl get nso my-nso -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].message}'
    exit 1
fi
```

### Alertmanager Rule

```yaml
groups:
  - name: nso-operator.rules
    rules:
      - alert: NSONotReady
        expr: nso_operator_condition_status{type=\"Ready\",status=\"False\"} == 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: \"NSO instance {{ $labels.name }} is not ready\"
          description: \"NSO instance has been not ready for more than 5 minutes\"
```

## Related Resources

- [NSO CRD Reference](nso-crd.md)
- [PackageBundle CRD Reference](packagebundle-crd.md)
- [Troubleshooting Guide](../user-guide/troubleshooting.md)
- [Monitoring Guide](../user-guide/monitoring.md)