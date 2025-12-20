# Quick Start Guide

Get up and running with the NSO Operator in 5 minutes! This guide will walk you through deploying your first NSO instance.

## Before You Begin

Ensure you have:
- NSO Operator installed ([Installation Guide](installation.md))
- Access to NSO container images
- A Kubernetes cluster with appropriate permissions

## Step 1: Prepare NSO Image

You'll need access to a Cisco NSO container image. For this example, we'll assume you have:
- NSO image: `your-registry/nso:6.3`
- Proper authentication to pull the image

## Step 2: Create Namespace

Create a namespace for your NSO instances:

```bash
kubectl create namespace nso-demo
```

## Step 3: Create Secrets

### NSO Admin Password
Create a secret for NSO admin authentication:

```bash
kubectl create secret generic nso-admin-secret \
  --from-literal=username=admin \
  --from-literal=password=admin123 \
  -n nso-demo
```

### Image Pull Secret (if needed)
If using a private registry:

```bash
kubectl create secret docker-registry nso-registry-secret \
  --docker-server=your-registry.com \
  --docker-username=your-username \
  --docker-password=your-password \
  --docker-email=your-email@company.com \
  -n nso-demo
```

## Step 4: Create Your First NSO Instance

Create a file called `my-first-nso.yaml`:

```yaml
apiVersion: orchestration.cisco.com.cisco.com/v1alpha1
kind: NSO
metadata:
  name: nso-sample
spec:
  image: containers.cisco.com/cisco-nso/cisco-nso-prod:6.1.11
  serviceName: "nso-prod"
  replicas: 1
  labelSelector:
    app: "nso"
  ports:
    - name: "http"
      port: 8080 
    - name: "https"
      port: 8888 
  adminCredentials:
    username: admin
    passwordSecretRef: nso-admin-password
  nsoConfigRef: nso-config
  env:
    - name: MY_ENV
      value: "container variable"
```

Apply the configuration:

```bash
kubectl apply -f my-first-nso.yaml
```

## Step 5: Monitor Deployment

Watch the NSO resource status:

```bash
kubectl get nso my-first-nso -n nso-demo -w
```

Check the pod status:

```bash
kubectl get pods -n nso-demo
```

View operator logs if needed:

```bash
kubectl logs -n nso-system deployment/nso-operator-controller-manager -f
```

## Step 6: Access NSO

### Get Service Information
```bash
kubectl get svc -n nso-demo
```

### Access NSO Web UI
If using LoadBalancer service type, get the external IP:

```bash
kubectl get svc my-first-nso -n nso-demo -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

Then access the Web UI at: `http://<EXTERNAL-IP>:8080`

### Port Forward (Alternative)
If LoadBalancer is not available, use port forwarding:

```bash
kubectl port-forward svc/my-first-nso -n nso-demo 8080:8080
```

Access the Web UI at: `http://localhost:8080`

Login with:
- Username: `admin`
- Password: `admin123`

## Step 7: Verify NSO is Working

### Check NSO Status
```bash
kubectl get nso my-first-nso -n nso-demo -o yaml
```

Look for status conditions indicating successful deployment.

### Connect via NETCONF
Test NETCONF connectivity:

```bash
kubectl port-forward svc/my-first-nso -n nso-demo 2022:2022
```

Then connect with an SSH client:
```bash
ssh admin@localhost -p 2022 -s netconf
```

## Step 8: Add a Package Bundle (Optional)

Create a PackageBundle to manage NSO packages:

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: sample-packages
  namespace: nso-demo
spec:
  # Git repository with NSO packages
  source:
    git:
      url: "https://github.com/your-org/nso-packages.git"
      ref: "main"
      path: "packages/"
  
  # Target NSO instance
  nsoSelector:
    matchLabels:
      app: my-first-nso
```

Apply the PackageBundle:

```bash
kubectl apply -f package-bundle.yaml
```

## Next Steps

Congratulations! You've successfully deployed your first NSO instance. Here's what you can do next:

### Learn More
- [NSO Instances Guide](../user-guide/nso-instances.md) - Deep dive into NSO configuration
- [Package Bundles Guide](../user-guide/package-bundles.md) - Managing NSO packages
- [Configuration Guide](../user-guide/configuration.md) - Advanced configuration options

### Explore Examples
- [Basic Examples](../examples/basic/) - Simple configuration examples
- [Production Examples](../examples/production/) - Production-ready setups

### Monitoring & Operations
- [Monitoring Guide](../user-guide/monitoring.md) - Set up metrics and alerts
- [Operations Guide](../operations/) - Production deployment strategies

## Clean Up

To remove the resources created in this guide:

```bash
# Delete NSO instance
kubectl delete nso my-first-nso -n nso-demo

# Delete secrets
kubectl delete secret nso-admin-secret nso-registry-secret -n nso-demo

# Delete namespace
kubectl delete namespace nso-demo
```

## Troubleshooting

If you encounter issues:

1. **Pod not starting**: Check logs with `kubectl logs <pod-name> -n nso-demo`
2. **Image pull errors**: Verify image name and pull secrets
3. **Storage issues**: Check if StorageClass exists and has capacity
4. **Access issues**: Verify service configuration and networking

For more help, see the [Troubleshooting Guide](../user-guide/troubleshooting.md).
