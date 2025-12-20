# Working with Package Bundles

Package Bundles allow you to manage NSO packages (NEDs, services, etc.) declaratively in Kubernetes. This guide covers how to create and manage PackageBundle resources.

## Overview

The PackageBundle Custom Resource provides:

- Automatic package download from various sources (Git, HTTP, etc.)
- Package lifecycle management
- Integration with NSO instances
- Version tracking and updates
- Package dependency handling

## Basic PackageBundle Resource

Here's a minimal PackageBundle definition:

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: cisco-neds
  namespace: default
spec:
  source:
    git:
      url: "https://github.com/cisco/cisco-neds.git"
      ref: "main"
  nsoSelector:
    matchLabels:
      app: my-nso
```

## Complete Configuration Example

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: production-packages
  namespace: nso-production
  labels:
    environment: production
    package-type: mixed
spec:
  # Package source configuration
  source:
    git:
      url: "https://github.com/your-org/nso-packages.git"
      ref: "v2.1.0"
      path: "packages/"
      auth:
        secretRef:
          name: git-credentials
    
    # Alternative: HTTP source
    # http:
    #   url: "https://packages.example.com/bundle.tar.gz"
    #   auth:
    #     secretRef:
    #       name: http-credentials
  
  # Target NSO instances
  nsoSelector:
    matchLabels:
      environment: production
    matchExpressions:
      - key: nso-version
        operator: In
        values: ["6.3", "6.3.1"]
  
  # Package management options
  packageManagement:
    # Automatic package reloading
    autoReload: true
    
    # Package installation order
    installOrder:
      - "cisco-iosxr-cli-7.38"
      - "cisco-ios-cli-6.77"
      - "my-service-package"
    
    # Skip packages that fail to load
    continueOnError: false
    
    # Package verification
    verifySignatures: true
  
  # Update policy
  updatePolicy:
    # Check for updates every 5 minutes
    interval: "5m"
    
    # Automatic updates for patch versions
    autoUpdate: patch
  
  # Resource limits for package operations
  resources:
    limits:
      memory: "1Gi"
      cpu: "500m"
    requests:
      memory: "512Mi"
      cpu: "200m"
```

## Source Types

### Git Source

Pull packages from Git repositories:

```yaml
spec:
  source:
    git:
      url: "https://github.com/your-org/packages.git"
      ref: "main"                    # branch, tag, or commit
      path: "nso-packages/"          # subdirectory (optional)
      depth: 1                       # shallow clone depth
      auth:
        secretRef:
          name: git-credentials      # optional for private repos
```

#### Git Authentication

For private repositories, create a secret:

```bash
# SSH key authentication
kubectl create secret generic git-credentials \
  --from-file=ssh-privatekey=/path/to/private-key \
  --from-literal=known_hosts="github.com ssh-rsa AAAAB3NzaC1yc2E..."

# Username/password authentication  
kubectl create secret generic git-credentials \
  --from-literal=username=your-username \
  --from-literal=password=your-token
```

### HTTP Source

Download packages from HTTP endpoints:

```yaml
spec:
  source:
    http:
      url: "https://packages.example.com/nso-packages-v2.1.0.tar.gz"
      checksum:
        sha256: "e3b0c44298fc1c149afbf4c8996fb924..."
      auth:
        secretRef:
          name: http-credentials
```

#### HTTP Authentication

```bash
kubectl create secret generic http-credentials \
  --from-literal=username=your-username \
  --from-literal=password=your-password
```

### Local Source

Use packages from ConfigMaps or Secrets:

```yaml
spec:
  source:
    configMap:
      name: package-files
    # or
    secret:
      name: package-files
```

## NSO Instance Selection

### Label Selectors

Select NSO instances using labels:

```yaml
spec:
  nsoSelector:
    matchLabels:
      environment: production
      region: us-west
```

### Expression-based Selection

More complex selection logic:

```yaml
spec:
  nsoSelector:
    matchExpressions:
      - key: environment
        operator: In
        values: ["production", "staging"]
      - key: nso-version
        operator: NotIn
        values: ["6.2"]
```

### Namespace Selection

Target NSO instances in specific namespaces:

```yaml
spec:
  namespaceSelector:
    matchLabels:
      team: network-ops
  nsoSelector:
    matchLabels:
      app: nso
```

## Package Management

### Installation Order

Control package installation sequence:

```yaml
spec:
  packageManagement:
    installOrder:
      - "foundation-packages"
      - "cisco-iosxr-cli-*"
      - "service-packages"
```

### Auto-reload

Automatically reload packages when changed:

```yaml
spec:
  packageManagement:
    autoReload: true
    reloadTimeout: "30s"
```

### Error Handling

Configure behavior when package operations fail:

```yaml
spec:
  packageManagement:
    continueOnError: true
    maxRetries: 3
    retryInterval: "10s"
```

## Update Policies

### Manual Updates

Packages update only when explicitly changed:

```yaml
spec:
  updatePolicy:
    autoUpdate: "off"
```

### Automatic Updates

Configure automatic update strategies:

```yaml
spec:
  updatePolicy:
    # Update interval
    interval: "10m"
    
    # Auto-update policy: off, patch, minor, major
    autoUpdate: "patch"
    
    # Update window (optional)
    window:
      start: "02:00"
      duration: "2h"
      timezone: "UTC"
```

## Managing PackageBundles

### Creating a PackageBundle

```bash
kubectl apply -f package-bundle.yaml
```

### Viewing PackageBundle Status

```bash
# Get basic status
kubectl get packagebundle cisco-neds

# Detailed status
kubectl describe packagebundle cisco-neds

# YAML output
kubectl get packagebundle cisco-neds -o yaml
```

### Forcing Package Updates

Trigger immediate update:

```bash
kubectl annotate packagebundle cisco-neds \
  orchestration.cisco.com/force-update="$(date +%s)"
```

### Pausing Updates

Temporarily disable automatic updates:

```bash
kubectl patch packagebundle cisco-neds \
  -p '{"spec":{"updatePolicy":{"autoUpdate":"off"}}}'
```

## Status Information

PackageBundle resources provide detailed status:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: PackagesInstalled
      message: "All packages successfully installed"
    - type: Downloaded
      status: "True"
      reason: SourceAvailable
      message: "Packages downloaded from source"
  
  # Package information
  packages:
    - name: "cisco-iosxr-cli-7.38"
      version: "7.38.1"
      status: "Installed"
    - name: "my-service-package"
      version: "2.1.0"
      status: "Installed"
  
  # Source status
  source:
    type: "git"
    url: "https://github.com/your-org/packages.git"
    ref: "v2.1.0"
    lastUpdate: "2024-01-15T10:30:00Z"
    checksum: "abc123..."
  
  # Target NSO instances
  targetInstances:
    - name: "production-nso-1"
      namespace: "nso-production"
      status: "Synced"
    - name: "production-nso-2"
      namespace: "nso-production"
      status: "Synced"
```

## Common Operations

### Package Verification

Check which packages are installed:

```bash
# List packages in NSO instance
kubectl exec deployment/my-nso -- ncs_cli -C -c "show packages"

# Check PackageBundle status
kubectl get packagebundle -o custom-columns="NAME:.metadata.name,PACKAGES:.status.packages[*].name"
```

### Rollback Packages

Revert to previous package version:

```bash
# Update PackageBundle to use previous Git tag
kubectl patch packagebundle cisco-neds \
  -p '{"spec":{"source":{"git":{"ref":"v2.0.0"}}}}'
```

### Package Dependencies

Handle package dependencies:

```yaml
spec:
  packageManagement:
    dependencies:
      cisco-iosxr-cli-7.38:
        requires:
          - "foundation-package >= 1.0.0"
        conflicts:
          - "old-iosxr-package"
```

## Advanced Configuration

### Custom Package Processing

Add custom processing steps:

```yaml
spec:
  processing:
    preInstall:
      - name: validate-packages
        image: your-validator:latest
        command: ["validate-nso-packages"]
    
    postInstall:
      - name: run-tests
        image: nso-test-runner:latest
        command: ["test-package-installation"]
```

### Package Filtering

Include/exclude specific packages:

```yaml
spec:
  packageFilter:
    include:
      - "cisco-*"
      - "my-service-*"
    exclude:
      - "*-dev"
      - "*-test"
```

### Multi-source Bundles

Combine packages from multiple sources:

```yaml
spec:
  sources:
    - name: cisco-neds
      git:
        url: "https://github.com/cisco/neds.git"
        ref: "latest"
    - name: custom-services
      http:
        url: "https://internal.company.com/packages.tar.gz"
```

## Troubleshooting

### Common Issues

**PackageBundle Stuck in Downloading**
- Check network connectivity to source
- Verify authentication credentials
- Check source URL and path

**Packages Not Installing in NSO**
- Verify NSO instance is running and accessible
- Check package compatibility with NSO version
- Review NSO logs for package loading errors

**Update Failures**
- Check PackageBundle conditions in status
- Verify source integrity (checksums, signatures)
- Review package dependencies

### Debug Commands

```bash
# Check PackageBundle events
kubectl get events --field-selector involvedObject.name=cisco-neds

# View operator logs
kubectl logs -n nso-operator-system deployment/nso-operator-controller-manager

# Check package download job logs (if applicable)
kubectl logs job/cisco-neds-download
```

## Integration Examples

### CI/CD Integration

Integrate PackageBundles with your CI/CD pipeline:

```yaml
# In your CI/CD pipeline, update package version
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: app-packages
spec:
  source:
    git:
      url: "https://github.com/your-org/packages.git"
      ref: "${CI_COMMIT_TAG}"  # Updated by CI/CD
```

### GitOps Workflow

Use with ArgoCD or Flux:

```yaml
# ArgoCD Application
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nso-packages
spec:
  source:
    repoURL: https://github.com/your-org/nso-config
    path: packagebundles/
    targetRevision: HEAD
  destination:
    server: https://kubernetes.default.svc
    namespace: nso-production
```

## Next Steps

- Learn about [NSO Configuration](configuration.md)
- Set up [Monitoring](monitoring.md) for package operations
- Explore [Advanced Examples](../examples/) for complex scenarios