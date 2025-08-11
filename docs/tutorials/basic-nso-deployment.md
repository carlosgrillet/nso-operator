# Tutorial: Basic NSO Deployment

This tutorial walks you through deploying your first NSO instance using the NSO Operator.

## Prerequisites

Before starting this tutorial, ensure you have:

- NSO Operator installed in your cluster ([Installation Guide](../getting-started/installation.md))
- kubectl access to your Kubernetes cluster
- Basic understanding of Kubernetes concepts
- Access to an NSO container image

## Tutorial Overview

In this tutorial, you will:
1. Create the necessary secrets and configuration
2. Deploy a basic NSO instance
3. Verify the deployment
4. Access the NSO Web UI
5. Connect via NETCONF
6. Clean up resources

Estimated time: 15 minutes

## Step 1: Prepare the Environment

### Create Namespace

First, create a dedicated namespace for this tutorial:

```bash
kubectl create namespace nso-tutorial
```

### Set Default Namespace

Set the namespace as default for this session:

```bash
kubectl config set-context --current --namespace=nso-tutorial
```

## Step 2: Create Admin Credentials

NSO requires admin credentials to be stored in a Kubernetes Secret.

Create the admin secret:

```bash
kubectl create secret generic nso-admin-secret \
  --from-literal=password=admin123 \
  --namespace=nso-tutorial
```

Verify the secret was created:

```bash
kubectl get secret nso-admin-secret -o yaml
```

## Step 3: Create NSO Configuration

Create a basic NSO configuration using a ConfigMap:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: nso-basic-config
  namespace: nso-tutorial
data:
  nso.conf: |
    # Basic NSO configuration
    /ncs-config/logs/syslog-config/facility local7
    /ncs-config/logs/audit-log/enabled true
    
    # Web UI configuration
    /ncs-config/webui/enabled true
    /ncs-config/webui/transport/tcp/enabled true
    /ncs-config/webui/transport/tcp/ip 0.0.0.0
    /ncs-config/webui/transport/tcp/port 8080
    
    # RESTCONF configuration  
    /ncs-config/restconf/enabled true
    
    # NETCONF configuration
    /ncs-config/netconf-north-bound/enabled true
    /ncs-config/netconf-north-bound/transport/ssh/enabled true
    /ncs-config/netconf-north-bound/transport/ssh/ip 0.0.0.0
    /ncs-config/netconf-north-bound/transport/ssh/port 2022
EOF
```

## Step 4: Deploy NSO Instance

Now create your first NSO instance:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: tutorial-nso
  namespace: nso-tutorial
  labels:
    app: nso
    tutorial: basic
spec:
  # NSO container image (replace with your actual image)
  image: "cisco/nso:6.3"
  
  # Single replica for this tutorial
  replicas: 1
  
  # Service configuration
  serviceName: "tutorial-nso-service"
  
  # Pod labels
  labelSelector:
    app: nso
    instance: tutorial
  
  # Exposed ports
  ports:
    - name: netconf
      port: 2022
      targetPort: 2022
      protocol: TCP
    - name: webui
      port: 8080
      targetPort: 8080
      protocol: TCP
    - name: restconf
      port: 8888
      targetPort: 8888
      protocol: TCP
  
  # Reference to configuration ConfigMap
  nsoConfigRef: "nso-basic-config"
  
  # Admin credentials
  adminCredentials:
    username: "admin"
    passwordSecretRef: "nso-admin-secret"
  
  # Environment variables
  env:
    - name: NSO_LOG_LEVEL
      value: "info"
    - name: JAVA_OPTS
      value: "-Xmx1g"
EOF
```

## Step 5: Monitor Deployment Progress

Watch the NSO resource status:

```bash
kubectl get nso tutorial-nso -w
```

You should see the status change from creating to running. Press Ctrl+C to stop watching.

Check the pods created by the operator:

```bash
kubectl get pods -l app=nso
```

View detailed information about the NSO resource:

```bash
kubectl describe nso tutorial-nso
```

## Step 6: Verify the Deployment

### Check Service

Verify the service was created:

```bash
kubectl get svc tutorial-nso-service
```

### Check Endpoints

Ensure the service has endpoints:

```bash
kubectl get endpoints tutorial-nso-service
```

### View Logs

Check NSO startup logs:

```bash
# Get pod name
POD_NAME=$(kubectl get pods -l app=nso -o jsonpath='{.items[0].metadata.name}')

# View logs
kubectl logs $POD_NAME
```

## Step 7: Access NSO Web UI

### Method 1: Port Forward

Create a port forward to access the Web UI:

```bash
kubectl port-forward svc/tutorial-nso-service 8080:8080
```

Open your browser and navigate to: http://localhost:8080

Login with:
- Username: `admin`
- Password: `admin123`

### Method 2: LoadBalancer Service (if available)

If your cluster supports LoadBalancer services, you can expose NSO externally:

```bash
kubectl patch svc tutorial-nso-service -p '{"spec":{"type":"LoadBalancer"}}'
```

Get the external IP:

```bash
kubectl get svc tutorial-nso-service
```

## Step 8: Connect via NETCONF

### Using SSH Client

Port forward the NETCONF port:

```bash
kubectl port-forward svc/tutorial-nso-service 2022:2022
```

In another terminal, connect via SSH:

```bash
ssh admin@localhost -p 2022 -s netconf
```

When prompted, enter the password: `admin123`

### Using ncclient (Python)

If you have Python's ncclient installed:

```python
from ncclient import manager

# Connect to NSO
with manager.connect(
    host='localhost',
    port=2022,
    username='admin',
    password='admin123',
    hostkey_verify=False
) as m:
    # Get capabilities
    for capability in m.server_capabilities:
        print(capability)
    
    # Get configuration
    config = m.get_config(source='running')
    print(config)
```

## Step 9: Explore NSO

### Web UI Exploration

In the NSO Web UI, explore:

1. **Dashboard**: Overview of NSO status
2. **Device Manager**: Device configuration (empty for now)
3. **Service Manager**: Service instances (empty for now)
4. **Tools**: Various NSO tools and utilities
5. **Admin**: System administration

### CLI Exploration

Access the NSO CLI directly in the pod:

```bash
kubectl exec -it $POD_NAME -- ncs_cli -C -u admin
```

Try some basic NSO CLI commands:

```bash
# Show NSO status
ncs --status

# Show packages
show packages

# Show version
show version

# Exit CLI
exit
```

## Step 10: Add a Simple Configuration

Let's add some basic configuration to NSO through the CLI:

```bash
# Access NSO CLI in configure mode
kubectl exec -it $POD_NAME -- ncs_cli -C -u admin

# Enter configure mode
admin@ncs# configure

# Add some basic configuration
admin@ncs(config)# devices authgroups group tutorial-group default-map remote-name admin remote-password admin
admin@ncs(config)# commit
admin@ncs(config)# exit

# Verify configuration
admin@ncs# show running-config devices authgroups
```

## Step 11: View Resource Usage

Check resource usage of your NSO instance:

```bash
# Pod resource usage
kubectl top pod $POD_NAME

# Describe pod for resource requests/limits
kubectl describe pod $POD_NAME | grep -A 10 "Requests\|Limits"
```

## Step 12: Clean Up

When you're done with the tutorial, clean up the resources:

```bash
# Delete the NSO instance
kubectl delete nso tutorial-nso

# Delete the ConfigMap
kubectl delete configmap nso-basic-config

# Delete the Secret
kubectl delete secret nso-admin-secret

# Delete the namespace (optional)
kubectl delete namespace nso-tutorial
```

## Troubleshooting

### Pod Not Starting

If the pod doesn't start:

```bash
# Check pod events
kubectl describe pod $POD_NAME

# Check operator logs
kubectl logs -n nso-operator-system deployment/nso-operator-controller-manager
```

### Can't Access Web UI

If you can't access the Web UI:

```bash
# Check if port-forward is working
netstat -an | grep 8080

# Test with curl
curl -v http://localhost:8080
```

### NETCONF Connection Issues

If NETCONF connection fails:

```bash
# Check if NETCONF port is accessible
telnet localhost 2022

# Check NSO logs for NETCONF errors
kubectl logs $POD_NAME | grep -i netconf
```

## Next Steps

Congratulations! You've successfully deployed your first NSO instance. Here are some next steps to explore:

### Advanced Tutorials
- [Multi-NSO Setup](multi-nso-setup.md) - Deploy multiple NSO instances
- [Package Management](package-management.md) - Work with NSO packages

### Deep Dive Guides
- [NSO Configuration](../user-guide/configuration.md) - Advanced configuration options
- [Monitoring Setup](../user-guide/monitoring.md) - Set up monitoring and metrics
- [Production Deployment](../operations/deployment-strategies.md) - Production best practices

### Examples
- [Basic Examples](../examples/basic/) - Simple configuration examples
- [Production Examples](../examples/production/) - Production-ready configurations

## Summary

In this tutorial, you learned how to:

✅ Create necessary secrets and configuration  
✅ Deploy a basic NSO instance using the NSO Operator  
✅ Verify the deployment status  
✅ Access NSO via Web UI and NETCONF  
✅ Perform basic NSO operations  
✅ Clean up resources  

You now have a solid foundation for working with the NSO Operator and can proceed to more advanced scenarios.