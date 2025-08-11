# PackageBundle Custom Resource Reference

The PackageBundle Custom Resource defines how to download and manage NSO packages from various sources.

## API Version and Kind

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
```

## Overview

The PackageBundle resource allows you to declaratively manage NSO package bundles with the following capabilities:
- Download packages from Git repositories (SCM) or HTTP URLs
- Manage package lifecycle and updates
- Handle authentication for private sources
- Track download and installation status

## Spec Fields

### Required Fields

#### `targetName` (string, required)
Name of the NSO instance where the packages will be loaded.

```yaml
spec:
  targetName: "my-nso-instance"
```

#### `origin` (OriginType, required)
Origin type of the packages. Must be either `"SCM"` for Git repositories or `"URL"` for HTTP/HTTPS downloads.

```yaml
spec:
  origin: "SCM"  # or "URL"
```

#### `source` (PackageSource, required)
Configuration for the package source location.

```yaml
spec:
  source:
    url: "https://github.com/your-org/nso-packages.git"
    branch: "main"
    path: "packages/"
```

### Optional Fields

#### `storageSize` (string, optional)
Size of the persistent volume that will store the downloaded packages.

```yaml
spec:
  storageSize: "10Gi"  # Default is typically "5Gi"
```

#### `insecureTLS` (bool, optional)
If set to true, self-signed certificates will be accepted for downloading or pulling.

```yaml
spec:
  insecureTLS: true  # Default: false
```

#### `credentials` (AccessCredentials, optional)
Authentication credentials for accessing private sources.

```yaml
spec:
  credentials:
    sshKeySecretRef: "git-ssh-key"      # For Git SSH access
    httpAuthSecretRef: "http-auth"      # For HTTP authentication
```

## OriginType Values

The `origin` field accepts the following values:

| Value | Description | Use Case |
|-------|-------------|----------|
| `"SCM"` | Source Code Management (Git) | Git repositories |
| `"URL"` | HTTP/HTTPS URL | Direct file downloads |

## PackageSource Type

The `PackageSource` type configures the source location:

### Fields

#### `url` (string, required)
URL of the repository or web location where packages are stored.

For Git repositories:
```yaml
source:
  url: "https://github.com/your-org/nso-packages.git"
```

For HTTP downloads:
```yaml
source:
  url: "https://packages.example.com/nso-packages-v2.1.0.tar.gz"
```

#### `branch` (string, optional)
Git branch, tag, or commit hash to use. Only applicable when `origin` is `"SCM"`.

```yaml
source:
  url: "https://github.com/your-org/nso-packages.git"
  branch: "v2.1.0"  # Can be branch name, tag, or commit hash
```

#### `path` (string, optional)
Path within the repository or archive where packages are located.

```yaml
source:
  url: "https://github.com/your-org/nso-packages.git"
  branch: "main"
  path: "production-packages/"
```

## AccessCredentials Type

The `AccessCredentials` type handles authentication for private sources:

### Fields

#### `sshKeySecretRef` (string, optional)
Reference to a Secret containing SSH private key for Git repository access.

```yaml
credentials:
  sshKeySecretRef: "git-ssh-key"
```

The referenced Secret should be created like this:
```bash
kubectl create secret generic git-ssh-key \
  --from-file=ssh-privatekey=/path/to/private/key \
  --from-file=known_hosts=/path/to/known_hosts
```

#### `httpAuthSecretRef` (string, optional)
Reference to a Secret containing HTTP authentication credentials.

```yaml
credentials:
  httpAuthSecretRef: "http-auth-secret"
```

The referenced Secret should contain:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: http-auth-secret
type: Opaque
data:
  username: <base64-encoded-username>
  password: <base64-encoded-password>
```

## Status Fields

The PackageBundle status provides information about the download and installation process:

### `phase` (PackageBundlePhase)
Current phase of the PackageBundle process:

| Phase | Description |
|-------|-------------|
| `Pending` | Initial state, waiting to start |
| `ContainerCreating` | Download container is being created |
| `Downloading` | Packages are being downloaded |
| `Downloaded` | Packages successfully downloaded |
| `FailedToDownload` | Download process failed |

### `message` (string, optional)
Additional information about the current status.

### `jobName` (string, optional)
Name of the Kubernetes Job responsible for downloading the packages.

### `lastTransitionTime` (*metav1.Time, optional)
Timestamp of the last phase transition.

## Complete Examples

### Git Repository Example

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: cisco-neds
  namespace: nso-production
  labels:
    package-type: neds
    environment: production
spec:
  # Target NSO instance
  targetName: "production-nso"
  
  # Storage configuration
  storageSize: "20Gi"
  
  # Git source configuration
  origin: "SCM"
  source:
    url: "https://github.com/cisco/nso-packages.git"
    branch: "v6.3.1"
    path: "neds/"
  
  # Authentication for private repository
  credentials:
    sshKeySecretRef: "cisco-git-ssh-key"
```

### HTTP URL Example

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: custom-services
  namespace: nso-production
spec:
  # Target NSO instance
  targetName: "production-nso"
  
  # Storage configuration
  storageSize: "5Gi"
  
  # HTTP source configuration
  origin: "URL"
  source:
    url: "https://packages.internal.company.com/services-v2.1.0.tar.gz"
  
  # Accept self-signed certificates
  insecureTLS: true
  
  # HTTP authentication
  credentials:
    httpAuthSecretRef: "internal-packages-auth"
```

### Public Repository Example

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: community-packages
  namespace: nso-dev
spec:
  # Target NSO instance
  targetName: "dev-nso"
  
  # Public Git repository (no credentials needed)
  origin: "SCM"
  source:
    url: "https://github.com/nso-community/packages.git"
    branch: "main"
    path: "examples/"
```

## kubectl Commands

### Create PackageBundle
```bash
kubectl apply -f packagebundle.yaml
```

### Get PackageBundles
```bash
# List all PackageBundles (shortname: pb)
kubectl get packagebundle
kubectl get pb

# List in all namespaces
kubectl get pb -A

# Get detailed information
kubectl describe pb cisco-neds
```

### Monitor PackageBundle Status
```bash
# Watch PackageBundle status
kubectl get pb cisco-neds -w

# Check status in YAML format
kubectl get pb cisco-neds -o yaml
```

### Update PackageBundle
```bash
# Edit PackageBundle
kubectl edit pb cisco-neds

# Update source branch
kubectl patch pb cisco-neds -p '{"spec":{"source":{"branch":"v6.3.2"}}}'
```

### Delete PackageBundle
```bash
kubectl delete pb cisco-neds
```

## Status Examples

### Successful Download
```yaml
status:
  phase: Downloaded
  message: "Successfully downloaded 15 packages from Git repository"
  jobName: "cisco-neds-download-abc123"
  lastTransitionTime: "2024-01-15T10:30:00Z"
```

### Download in Progress
```yaml
status:
  phase: Downloading
  message: "Cloning repository and extracting packages"
  jobName: "cisco-neds-download-def456"
  lastTransitionTime: "2024-01-15T10:25:00Z"
```

### Failed Download
```yaml
status:
  phase: FailedToDownload
  message: "Authentication failed: invalid SSH key"
  jobName: "cisco-neds-download-ghi789"
  lastTransitionTime: "2024-01-15T10:32:00Z"
```

## Validation

The PackageBundle CRD includes the following validations:

### Enum Validation
- `origin` must be either `"SCM"` or `"URL"`
- `phase` must be one of the defined PackageBundlePhase values

### Required Field Validation
- `targetName` must be specified
- `origin` must be specified
- `source.url` must be specified

### Example Validation Errors

**Invalid origin type:**
```
error validating data: ValidationError(PackageBundle.spec.origin): invalid value: "GIT", allowed values: ["SCM", "URL"]
```

**Missing required field:**
```
error validating data: ValidationError(PackageBundle.spec): missing required field "targetName"
```

## Best Practices

### Source Management
- Use specific tags or commit hashes for production
- Use branches for development environments
- Keep package repositories organized with clear directory structures

### Authentication
- Use SSH keys for Git repositories when possible
- Store credentials in Kubernetes Secrets
- Rotate authentication credentials regularly

### Storage Planning
- Estimate package sizes and plan storage accordingly
- Use appropriate storage classes for performance needs
- Monitor storage usage and clean up old packages

### Monitoring
- Monitor PackageBundle phases and status
- Set up alerts for failed downloads
- Track package versions and updates

## Troubleshooting

### Common Issues

**Authentication Failures**
- Verify SSH key or HTTP credentials are correct
- Check Secret references and data encoding
- Test access manually with the same credentials

**Download Timeouts**
- Check network connectivity to source
- Verify firewall and security group settings
- Consider increasing timeout values

**Storage Issues**
- Check available storage capacity
- Verify StorageClass exists and is accessible
- Monitor PVC status and events

## Related Resources

- [NSO CRD Reference](nso-crd.md)
- [Status Conditions Reference](status-conditions.md)
- [User Guide: Package Bundles](../user-guide/package-bundles.md)
- [Configuration Guide](../user-guide/configuration.md)