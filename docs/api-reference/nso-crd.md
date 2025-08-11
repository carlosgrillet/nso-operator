# NSO Custom Resource Reference

The NSO Custom Resource defines how to deploy and manage Cisco NSO instances in Kubernetes.

## API Version and Kind

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
```

## Overview

The NSO resource allows you to declaratively manage NSO instances with the following capabilities:
- NSO container deployment and lifecycle management
- Service configuration and port management
- Admin credential management
- Environment and volume configuration

## Spec Fields

### Required Fields

#### `image` (string, required)
The container image to use for the NSO instance.

```yaml
spec:
  image: "cisco/nso:6.3.1"
```

#### `serviceName` (string, required)
Name of the headless service that will be created for the NSO instance.

```yaml
spec:
  serviceName: "my-nso-service"
```

#### `replicas` (int32, required)
Number of NSO replicas to deploy.

```yaml
spec:
  replicas: 1
```

#### `labelSelector` (map[string]string, required)
Labels used to select pods for the NSO deployment.

```yaml
spec:
  labelSelector:
    app: nso
    instance: production
```

#### `ports` ([]corev1.ServicePort, required)
Service ports to expose for the NSO instance.

```yaml
spec:
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
```

#### `nsoConfigRef` (string, required)
Reference to a ConfigMap containing NSO configuration.

```yaml
spec:
  nsoConfigRef: "nso-config-cm"
```

#### `adminCredentials` (Credentials, required)
NSO admin user credentials configuration.

```yaml
spec:
  adminCredentials:
    username: "admin"
    passwordSecretRef: "nso-admin-password"
```

### Optional Fields

#### `env` ([]corev1.EnvVar, optional)
Environment variables to set in the NSO container.

```yaml
spec:
  env:
    - name: NSO_LOG_LEVEL
      value: "info"
    - name: JAVA_OPTS
      value: "-Xmx2g"
    - name: SECRET_VALUE
      valueFrom:
        secretKeyRef:
          name: my-secret
          key: secret-key
```

#### `volumeMounts` ([]corev1.VolumeMount, optional)
Volume mounts for the NSO container.

```yaml
spec:
  volumeMounts:
    - name: nso-data
      mountPath: /nso/data
    - name: nso-logs
      mountPath: /nso/logs
    - name: custom-config
      mountPath: /etc/nso/custom
      readOnly: true
```

#### `volumes` ([]corev1.Volume, optional)
Volumes to make available to the NSO container.

```yaml
spec:
  volumes:
    - name: nso-data
      persistentVolumeClaim:
        claimName: nso-data-pvc
    - name: nso-logs
      emptyDir: {}
    - name: custom-config
      configMap:
        name: custom-nso-config
```

## Credentials Type

The `Credentials` type is used for NSO admin authentication:

### Fields

#### `username` (string, required)
The NSO admin username.

#### `passwordSecretRef` (string, required)
Reference to a Secret containing the NSO admin password.

The referenced Secret must contain a key with the password value. For example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: nso-admin-password
type: Opaque
data:
  password: <base64-encoded-password>
```

## Status Fields

The NSO resource status is currently minimal and will be expanded in future versions:

```yaml
status: {}
```

## Complete Example

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
  
  # Scaling
  replicas: 2
  
  # Service configuration
  serviceName: "nso-service"
  labelSelector:
    app: nso
    instance: production
  
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
    - name: ipc
      port: 4569
      targetPort: 4569
      protocol: TCP
  
  # Configuration
  nsoConfigRef: "nso-production-config"
  
  # Admin credentials
  adminCredentials:
    username: "admin"
    passwordSecretRef: "nso-admin-secret"
  
  # Environment variables
  env:
    - name: NSO_LOG_LEVEL
      value: "info"
    - name: JAVA_OPTS
      value: "-Xmx3g -XX:+UseG1GC"
    - name: NSO_ENVIRONMENT
      value: "production"
  
  # Storage and configuration volumes
  volumeMounts:
    - name: nso-data
      mountPath: /nso/data
    - name: nso-logs
      mountPath: /var/log/nso
    - name: custom-config
      mountPath: /etc/nso/custom
      readOnly: true
  
  volumes:
    - name: nso-data
      persistentVolumeClaim:
        claimName: nso-data-pvc
    - name: nso-logs
      persistentVolumeClaim:
        claimName: nso-logs-pvc
    - name: custom-config
      configMap:
        name: nso-custom-config
```

## kubectl Commands

### Create NSO Instance
```bash
kubectl apply -f nso-instance.yaml
```

### Get NSO Instances
```bash
# List all NSO instances
kubectl get nso

# List NSO instances in all namespaces
kubectl get nso -A

# Get detailed information
kubectl describe nso my-nso
```

### Update NSO Instance
```bash
# Edit NSO resource
kubectl edit nso my-nso

# Patch NSO resource
kubectl patch nso my-nso -p '{"spec":{"replicas":3}}'
```

### Delete NSO Instance
```bash
kubectl delete nso my-nso
```

## Validation

The NSO CRD includes the following validations:

### Required Field Validation
All required fields must be specified:
- `image`
- `serviceName`
- `replicas`
- `labelSelector`
- `ports`
- `nsoConfigRef`
- `adminCredentials.username`
- `adminCredentials.passwordSecretRef`

### Example Validation Errors

**Missing required field:**
```
error validating data: ValidationError(NSO.spec): missing required field "image"
```

**Invalid field type:**
```
error validating data: ValidationError(NSO.spec.replicas): invalid value: "two", expected integer
```

## Best Practices

### Resource Naming
- Use descriptive names for NSO instances
- Include environment indicators (dev, staging, prod)
- Use consistent naming conventions across resources

### Configuration Management
- Store NSO configuration in ConfigMaps
- Use Secrets for sensitive data
- Version your configurations

### Security
- Always use Secrets for admin passwords
- Limit container privileges
- Use network policies to restrict access

### High Availability
- Use multiple replicas for production
- Configure appropriate pod disruption budgets
- Use persistent storage for NSO data

## Related Resources

- [PackageBundle CRD Reference](packagebundle-crd.md)
- [Status Conditions Reference](status-conditions.md)
- [User Guide: NSO Instances](../user-guide/nso-instances.md)
- [Configuration Guide](../user-guide/configuration.md)