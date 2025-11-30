# Basic Examples

This directory contains simple examples to get you started with the NSO Operator.

## Examples Overview

| File | Description | Use Case |
|------|-------------|----------|
| `simple-nso.yaml` | Minimal NSO deployment | Development, testing, learning |
| `simple-packagebundle.yaml` | Basic package management | Package download examples |
| `nso-with-storage.yaml` | NSO with persistent storage | Data persistence |
| `private-registry.yaml` | NSO from private registry | Enterprise environments |

## Prerequisites

Before using these examples, ensure you have:

- NSO Operator installed in your cluster
- kubectl configured to access your cluster
- Access to NSO container images

## Quick Start

### Deploy Simple NSO

```bash
# Deploy the simple NSO example
kubectl apply -f simple-nso.yaml

# Check deployment status
kubectl get nso simple-nso

# Access Web UI (in another terminal)
kubectl port-forward svc/simple-nso-service 8080:8080
```

### Add Package Bundle

```bash
# Deploy package bundle
kubectl apply -f simple-packagebundle.yaml

# Monitor package download
kubectl get packagebundle demo-packages -w
```

## Customization

### Changing NSO Image

Edit the `image` field in any example:

```yaml
spec:
  image: "your-registry/nso:your-tag"
```

### Modifying Resources

Adjust resource requirements:

```yaml
spec:
  env:
    - name: JAVA_OPTS
      value: "-Xmx2g"  # Adjust heap size
```

### Adding Environment Variables

```yaml
spec:
  env:
    - name: NSO_LOG_LEVEL
      value: "debug"
    - name: CUSTOM_VAR
      value: "custom-value"
```

## Common Modifications

### Different Namespace

To deploy in a different namespace:

```bash
# Create namespace
kubectl create namespace my-namespace

# Deploy with namespace override
kubectl apply -f simple-nso.yaml -n my-namespace
```

### External Access

To expose NSO externally via LoadBalancer:

```bash
# Patch service type
kubectl patch svc simple-nso-service -p '{"spec":{"type":"LoadBalancer"}}'

# Get external IP
kubectl get svc simple-nso-service
```

### Custom Admin Password

```bash
# Create secret with custom password
kubectl create secret generic custom-admin \
  --from-literal=password=your-secure-password

# Update NSO spec to reference new secret
# Edit the adminCredentials.passwordSecretRef field
```

## Validation

### Check NSO Status

```bash
# Get NSO resource status
kubectl get nso -o wide

# Detailed status information
kubectl describe nso simple-nso
```

### Verify Services

```bash
# Check services
kubectl get svc

# Test connectivity
kubectl port-forward svc/simple-nso-service 8080:8080 &
curl -v http://localhost:8080
```

### Check Logs

```bash
# Get pod name
POD=$(kubectl get pods -l app=nso -o jsonpath='{.items[0].metadata.name}')

# View logs
kubectl logs $POD
```

## Cleanup

Remove all resources:

```bash
# Delete NSO instance
kubectl delete -f simple-nso.yaml

# Delete package bundles (if deployed)
kubectl delete -f simple-packagebundle.yaml
```

## Next Steps

Once you're comfortable with these basic examples:

1. **Advanced Examples**: Check out [Production Examples](../production/)
2. **Tutorials**: Follow step-by-step [Tutorials](../../tutorials/)
3. **Configuration**: Learn about advanced [Configuration](../../user-guide/configuration.md)
4. **Monitoring**: Set up [Monitoring](../../user-guide/monitoring.md)

## Support

If you encounter issues with these examples:

1. Check the [Troubleshooting Guide](../../user-guide/troubleshooting.md)
2. Review [FAQ](../../faq.md)
3. Check operator logs: `kubectl logs -n nso-operator-system deployment/nso-operator-controller-manager`