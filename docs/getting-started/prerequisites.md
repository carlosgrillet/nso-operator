# Prerequisites

Before installing and using the NSO Operator, ensure your environment meets the following requirements.

## System Requirements

### Kubernetes Cluster
- **Kubernetes version**: v1.11.3 or higher
- **kubectl**: v1.11.3 or higher
- **Cluster access**: Admin permissions to install CRDs and RBAC resources

### Development Tools (for building from source)
- **Go**: v1.24.0 or higher
- **Docker**: v17.03 or higher
- **Make**: For build automation
- **Git**: For source code management

### Optional Tools
- **kind**: v0.29.0 or higher (for local development)
- **Helm**: v3.0+ (if using Helm installation method)

## NSO Requirements

### NSO License
The NSO Operator manages NSO instances, but you need:
- Valid Cisco NSO license
- Access to NSO container images
- NSO packages (NEDs, services) as needed

### Container Registry Access
- Access to pull NSO container images
- Ability to push custom images (if building custom NSO containers)

## Network Requirements

### Cluster Networking
- Container-to-container communication
- Service discovery (DNS)
- Ingress controller (for external access to NSO web UI)

### External Connectivity
- Internet access for downloading packages (if using external package repositories)
- Access to managed network devices from NSO instances

## Storage Requirements

### Persistent Storage
- StorageClass available for persistent volumes
- Sufficient storage for NSO databases and logs
- Backup storage (recommended)

### Performance Considerations
- SSD storage recommended for NSO databases
- Network latency to managed devices should be minimal

## Security Considerations

### RBAC
- Service accounts for NSO Operator
- Appropriate cluster roles and bindings
- Network policies (if required by security policies)

### Secrets Management
- Secure storage for NSO admin passwords
- TLS certificates for secure communication
- SSH keys for device connectivity

## Verification

Run these commands to verify your environment:

```bash
# Check Kubernetes version
kubectl version --client --short

# Check cluster access
kubectl cluster-info

# Verify you can create CRDs (admin access)
kubectl auth can-i create customresourcedefinitions

# Check available storage classes
kubectl get storageclass
```

## Next Steps

Once your environment meets these prerequisites, proceed to:
- [Installation Guide](installation.md) - Install the NSO Operator
- [Quick Start](quick-start.md) - Deploy your first NSO instance