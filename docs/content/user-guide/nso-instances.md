# Managing NSO Instances

This guide covers how to create, configure, and manage NSO instances using the NSO Operator.

## Overview

The NSO Custom Resource allows you to declaratively manage Cisco NSO instances in Kubernetes. The operator handles:

- NSO container lifecycle management
- Persistent storage for NSO data
- Service exposure and networking
- Configuration management
- Health monitoring and recovery

## Basic NSO Resource

Here's a minimal NSO resource definition:

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: my-nso
  namespace: default
spec:
  image: "cisco/nso:6.3"
  replicas: 1
  adminSecret:
    name: nso-admin-secret
```

## Complete Configuration Example

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: production-nso
  namespace: nso-production
  labels:
    app: nso
    environment: production
spec:
  # Container configuration
  image: "your-registry/nso:6.3.1"
  imagePullPolicy: IfNotPresent
  imagePullSecrets:
    - name: registry-secret
  
  # Scaling
  replicas: 2
  
  # Admin credentials
  adminSecret:
    name: nso-admin-credentials
  
  # Resource limits
  resources:
    limits:
      memory: "4Gi"
      cpu: "2000m"
    requests:
      memory: "2Gi"
      cpu: "1000m"
  
  # Storage configuration
  storage:
    size: "50Gi"
    storageClassName: "fast-ssd"
    accessModes:
      - ReadWriteOnce
  
  # Service configuration
  service:
    type: ClusterIP
    ports:
      - name: netconf
        port: 2022
        targetPort: 2022
      - name: webui
        port: 8080
        targetPort: 8080
      - name: rest
        port: 8888
        targetPort: 8888
  
  # NSO configuration
  config:
    # Java heap size
    javaHeapSize: "2G"
    
    # Enable/disable features
    features:
      webUI: true
      restConf: true
      netConf: true
    
    # Custom configuration snippets
    customConfig: |
      <config xmlns="http://tail-f.com/ns/config/1.0">
        <rest xmlns="http://tail-f.com/ns/config/1.0">
          <enabled>true</enabled>
        </rest>
      </config>
  
  # Environment variables
  env:
    - name: NSO_LOG_LEVEL
      value: "info"
    - name: CUSTOM_SETTING
      value: "production"
  
  # Node selection
  nodeSelector:
    kubernetes.io/os: linux
    node-type: nso-worker
  
  # Tolerations
  tolerations:
    - key: "nso-dedicated"
      operator: "Equal"
      value: "true"
      effect: "NoSchedule"
  
  # Pod security context
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 1000
```

## Configuration Options

### Container Configuration

| Field | Description | Required |
|-------|-------------|---------|
| `image` | NSO container image | Yes |
| `imagePullPolicy` | Image pull policy (Always, IfNotPresent, Never) | No |
| `imagePullSecrets` | Secrets for pulling from private registries | No |

### Scaling

| Field | Description | Default |
|-------|-------------|--------|
| `replicas` | Number of NSO instances | 1 |

**Note**: High availability setups require special NSO configuration and shared storage.

### Admin Credentials

| Field | Description | Required |
|-------|-------------|---------|
| `adminSecret.name` | Secret containing NSO admin credentials | Yes |

The secret must contain:
- `username`: NSO admin username
- `password`: NSO admin password

```bash
kubectl create secret generic nso-admin-secret \
  --from-literal=username=admin \
  --from-literal=password=secretpassword
```

### Storage Configuration

| Field | Description | Default |
|-------|-------------|--------|
| `storage.size` | Persistent volume size | 10Gi |
| `storage.storageClassName` | StorageClass name | default |
| `storage.accessModes` | Volume access modes | [ReadWriteOnce] |

### Service Configuration

The operator creates a Service to expose NSO ports:

| Port | Protocol | Description |
|------|----------|-------------|
| 2022 | TCP | NETCONF SSH |
| 8080 | TCP | Web UI (HTTP) |
| 8888 | TCP | RESTCONF API |
| 4569 | TCP | IPC (internal) |

### NSO-Specific Configuration

| Field | Description | Default |
|-------|-------------|--------|
| `config.javaHeapSize` | Java heap size | 1G |
| `config.features.webUI` | Enable Web UI | true |
| `config.features.restConf` | Enable RESTCONF | true |
| `config.features.netConf` | Enable NETCONF | true |
| `config.customConfig` | Custom NSO XML configuration | - |

## Managing NSO Instances

### Creating an NSO Instance

```bash
kubectl apply -f nso-instance.yaml
```

### Viewing NSO Status

```bash
# Get basic status
kubectl get nso my-nso

# Get detailed status
kubectl describe nso my-nso

# View status in YAML format
kubectl get nso my-nso -o yaml
```

### Scaling NSO Instances

```bash
# Scale to 3 replicas
kubectl patch nso my-nso -p '{"spec":{"replicas":3}}'
```

### Updating NSO Configuration

Edit the NSO resource and apply changes:

```bash
kubectl edit nso my-nso
```

### Deleting an NSO Instance

```bash
kubectl delete nso my-nso
```

**Warning**: This will delete the NSO instance and its persistent data.

## Status and Conditions

The NSO resource provides status information:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: NSO_Ready
      message: "NSO instance is ready and accepting connections"
  phase: Running
  replicas: 1
  readyReplicas: 1
  observedGeneration: 1
```

### Condition Types

- **Ready**: NSO instance is ready and accepting connections
- **Progressing**: NSO deployment is in progress
- **ReplicaFailure**: Some replicas failed to start
- **StorageReady**: Persistent storage is available

## Common Operations

### Accessing NSO Web UI

```bash
# Port forward to access Web UI
kubectl port-forward svc/my-nso 8080:8080

# Access at http://localhost:8080
```

### Connecting via NETCONF

```bash
# Port forward NETCONF port
kubectl port-forward svc/my-nso 2022:2022

# Connect with SSH
ssh admin@localhost -p 2022 -s netconf
```

### Accessing NSO CLI

```bash
# Execute NSO CLI in running pod
kubectl exec -it deployment/my-nso -- ncs_cli -C -u admin
```

### Viewing NSO Logs

```bash
# View logs from NSO container
kubectl logs deployment/my-nso -c nso

# Follow logs
kubectl logs -f deployment/my-nso -c nso
```

## Advanced Topics

### High Availability

For HA setups, consider:
- Shared storage for NSO data
- Load balancer configuration
- Proper NSO clustering configuration
- Database replication settings

### Custom Initialization

You can provide custom initialization scripts:

```yaml
spec:
  initContainers:
    - name: init-config
      image: busybox
      command: ['sh', '-c', 'echo "Custom initialization"']
      volumeMounts:
        - name: nso-data
          mountPath: /nso-data
```

### Backup and Recovery

See the [Backup and Recovery Guide](../operations/backup-recovery.md) for detailed backup strategies.

## Troubleshooting

Common issues and solutions:

### Pod Stuck in Pending
- Check node resources and capacity
- Verify StorageClass exists and has available storage
- Check node selectors and tolerations

### Image Pull Errors
- Verify image name and tag
- Check image pull secrets
- Ensure registry accessibility

### NSO Not Ready
- Check NSO logs for startup errors
- Verify admin secret exists and is correct
- Check resource limits and Java heap size

For more troubleshooting help, see the [Troubleshooting Guide](troubleshooting.md).

## Next Steps

- Learn about [Package Bundles](package-bundles.md)
- Explore [Configuration Options](configuration.md)
- Set up [Monitoring](monitoring.md)