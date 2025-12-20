# Installation Guide

This guide covers different methods to install the NSO Operator in your Kubernetes cluster.

## Method 1: Using kubectl (Recommended)

### Install CRDs
First, install the Custom Resource Definitions:

```bash
kubectl apply -f //TODO
```

### Install Operator
Install the operator and required RBAC:

```bash
kubectl apply -f //TODO
```

### Verify Installation
Check that the operator is running:

```bash
kubectl get pods -n nso-system
```

You should see output similar to:
```
NAME                                        READY   STATUS    RESTARTS   AGE
nso-operator-controller-manager-xxx-xxx     2/2     Running   0          1m
```

## Method 2: Build and Deploy from Source

### Prerequisites
Ensure you have the development prerequisites from [Prerequisites](prerequisites.md).

### Clone Repository
```bash
git clone https://github.com/carlosgrillet/nso-operator.git
cd nso-operator
```

### Build and Deploy
```bash
# Build the operator image
make docker-build IMG=your-registry/nso-operator:tag

# Push to registry
make docker-push IMG=your-registry/nso-operator:tag

# Install CRDs
make install

# Deploy operator
make deploy IMG=your-registry/nso-operator:tag
```

## Configuration Options

### Operator Configuration
The operator can be configured using environment variables or command-line flags:

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `METRICS_ADDR` | `:8080` | Metrics server address |
| `ENABLE_LEADER_ELECTION` | `false` | Enable leader election |
| `HEALTH_PROBE_ADDR` | `:8081` | Health probe address |

## Post-Installation

### Verify Resources
Check that all resources were created:

```bash
# Check CRDs
kubectl get crd | grep orchestration.cisco.com

# Check operator deployment
kubectl get deployment -n nso-system

# Check service account and RBAC
kubectl get serviceaccount -n nso-system
kubectl get clusterrole | grep nso-operator
```

### Check Logs
View operator logs to ensure it's running correctly:

```bash
kubectl logs -n nso-system deployment/nso-operator-controller-manager
```

### Test Installation
Create a simple NSO instance to test the installation:

```bash
kubectl apply -f - <<EOF
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: test-nso
  namespace: default
spec:
  image: "your-nso-image:latest"
  replicas: 1
EOF
```

Check the NSO resource status:
```bash
kubectl get nso test-nso -o yaml
```

## Upgrading

### Using kubectl
```bash
kubectl apply -f https://raw.githubusercontent.com/your-org/nso-operator/main/dist/install.yaml
```

## Uninstalling

### Using kubectl
```bash
# Delete operator
kubectl delete -f //TODO

# Delete CRDs (this will also delete all NSO and PackageBundle resources)
kubectl delete crd nsos.orchestration.cisco.com
kubectl delete crd packagebundles.orchestration.cisco.com
```

### Clean up CRDs (if needed)
```bash
kubectl delete crd nsos.orchestration.cisco.com
kubectl delete crd packagebundles.orchestration.cisco.com
```

## Troubleshooting

### Common Issues

**Operator Pod Not Starting**
- Check logs: `kubectl logs -n nso-operator-system deployment/nso-operator-controller-manager`
- Verify image pull secrets if using private registry
- Check resource limits and node capacity

**CRD Installation Failed**
- Ensure you have cluster-admin permissions
- Check for conflicting CRDs with same name

**RBAC Issues**
- Verify service account has necessary permissions
- Check cluster role bindings

For more troubleshooting help, see [Troubleshooting Guide](../user-guide/troubleshooting.md).

## Next Steps

- [Quick Start Guide](quick-start.md) - Deploy your first NSO instance
- [User Guide](../user-guide/) - Learn about NSO and PackageBundle resources
