# NSO Operator Architecture

This document describes the architecture and design of the NSO Operator, including its components, interactions, and key design decisions.

## Overview

The NSO Operator is a Kubernetes operator built using the Operator SDK and controller-runtime framework. It manages the lifecycle of Cisco NSO instances and their associated package bundles through custom resources and controllers.

## High-Level Architecture

``` mermaid
sequenceDiagram
  autonumber
  actor User
  User->>Kubernetes: Create NSO instance
  loop control loop
    Operator-->>Kubernetes: WATCH NSO
    Operator->>Operator: reconcile
    Operator-->>Kubernetes: Create|Delete|Patch
  end
  Kubernetes->>User: Success|Failure
```

## Core Components

### 1. NSO Controller

The NSO Controller manages the lifecycle of NSO instances.

#### Responsibilities
- Watch NSO custom resources for changes
- Create and manage Kubernetes Deployments for NSO instances
- Manage Services, ConfigMaps, and Secrets
- Update NSO resource status
- Handle scaling and updates

#### Controller Logic Flow
```go
func (r *NSOReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch NSO resource
    nso := &orchestrationv1alpha1.NSO{}
    if err := r.Get(ctx, req.NamespacedName, nso); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // 2. Create or update Deployment
    if err := r.reconcileDeployment(ctx, nso); err != nil {
        return ctrl.Result{}, err
    }
    
    // 3. Create or update Service
    if err := r.reconcileService(ctx, nso); err != nil {
        return ctrl.Result{}, err
    }
    
    // 4. Update status
    if err := r.updateStatus(ctx, nso); err != nil {
        return ctrl.Result{}, err
    }
    
    return ctrl.Result{}, nil
}
```

### 2. PackageBundle Controller

The PackageBundle Controller manages NSO package downloads and installations.

#### Responsibilities
- Watch PackageBundle custom resources
- Create Jobs for package downloads
- Manage persistent storage for packages
- Track download and installation status
- Handle package updates and rollbacks

#### Package Download Process
```go
func (r *PackageBundleReconciler) reconcileDownload(ctx context.Context, pb *orchestrationv1alpha1.PackageBundle) error {
    // 1. Create PVC for package storage
    if err := r.ensurePVC(ctx, pb); err != nil {
        return err
    }
    
    // 2. Create download Job based on source type
    switch pb.Spec.Origin {
    case orchestrationv1alpha1.OriginTypeSCM:
        return r.createGitDownloadJob(ctx, pb)
    case orchestrationv1alpha1.OriginTypeURL:
        return r.createHTTPDownloadJob(ctx, pb)
    }
    
    return nil
}
```

### 3. Helper Components

#### Resource Manager
Handles creation and management of Kubernetes resources:

```go
type ResourceManager struct {
    client.Client
    Scheme *runtime.Scheme
}

func (rm *ResourceManager) EnsureDeployment(ctx context.Context, nso *orchestrationv1alpha1.NSO) error {
    deployment := rm.buildDeployment(nso)
    
    // Use server-side apply for idempotent operations
    return rm.Patch(ctx, deployment, client.Apply, client.ForceOwnership, client.FieldOwner("nso-operator"))
}
```

#### Status Manager
Manages resource status updates:

```go
type StatusManager struct {
    client.Client
}

func (sm *StatusManager) UpdateNSOStatus(ctx context.Context, nso *orchestrationv1alpha1.NSO, deployment *appsv1.Deployment) error {
    nso.Status.Replicas = deployment.Status.Replicas
    nso.Status.ReadyReplicas = deployment.Status.ReadyReplicas
    nso.Status.Conditions = sm.buildConditions(deployment)
    
    return sm.Status().Update(ctx, nso)
}
```

## Custom Resource Definitions

### NSO Resource Structure

```go
type NSOSpec struct {
    Image            string                 `json:"image"`
    ServiceName      string                 `json:"serviceName"`
    Replicas         int32                  `json:"replicas"`
    LabelSelector    map[string]string      `json:"labelSelector"`
    Ports            []corev1.ServicePort   `json:"ports"`
    NsoConfigRef     string                 `json:"nsoConfigRef"`
    AdminCredentials Credentials            `json:"adminCredentials"`
    Env              []corev1.EnvVar        `json:"env,omitempty"`
    VolumeMounts     []corev1.VolumeMount   `json:"volumeMounts,omitempty"`
    Volumes          []corev1.Volume        `json:"volumes,omitempty"`
}

type NSOStatus struct {
    Conditions      []metav1.Condition `json:"conditions,omitempty"`
    Replicas        int32              `json:"replicas,omitempty"`
    ReadyReplicas   int32              `json:"readyReplicas,omitempty"`
    ObservedGeneration int64           `json:"observedGeneration,omitempty"`
}
```

### PackageBundle Resource Structure

```go
type PackageBundleSpec struct {
    TargetName    string            `json:"targetName"`
    StorageSize   string            `json:"storageSize,omitempty"`
    Origin        OriginType        `json:"origin"`
    InsecureTLS   bool             `json:"insecureTLS,omitempty"`
    Credentials   AccessCredentials `json:"credentials,omitempty"`
    Source        PackageSource     `json:"source"`
}

type PackageBundleStatus struct {
    Phase              PackageBundlePhase `json:"phase,omitempty"`
    Message            string             `json:"message,omitempty"`
    JobName            string             `json:"jobName,omitempty"`
    LastTransitionTime *metav1.Time       `json:"lastTransitionTime,omitempty"`
}
```

## Control Loops and Reconciliation

### NSO Reconciliation Loop

```
NSO Resource Change
       │
       ▼
┌─────────────────┐
│   Get NSO       │
│   Resource      │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Reconcile      │
│  Deployment     │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Reconcile      │
│   Service       │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Reconcile      │
│  ConfigMaps     │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Update         │
│  Status         │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│   Requeue       │
│  (if needed)    │
└─────────────────┘
```

### PackageBundle Reconciliation Loop

```
PackageBundle Change
       │
       ▼
┌─────────────────┐
│   Get PB        │
│   Resource      │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Check Phase    │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Ensure PVC     │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Create/Check   │
│  Download Job   │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Update Status  │
│  and Phase      │
└─────────────────┘
```

## Package Management Architecture

### Download Job Architecture

Package downloads are handled by Kubernetes Jobs with different containers based on source type:

#### Git Download Container
```yaml
apiVersion: batch/v1
kind: Job
spec:
  template:
    spec:
      containers:
      - name: git-downloader
        image: alpine/git
        command: ["/scripts/git-download.sh"]
        volumeMounts:
        - name: packages
          mountPath: /packages
        - name: ssh-key
          mountPath: /ssh
          readOnly: true
```

#### HTTP Download Container
```yaml
apiVersion: batch/v1
kind: Job
spec:
  template:
    spec:
      containers:
      - name: http-downloader
        image: curlimages/curl
        command: ["/scripts/http-download.sh"]
        volumeMounts:
        - name: packages
          mountPath: /packages
```

### Package Installation Flow

```
Download Job Complete
       │
       ▼
┌─────────────────┐
│  Verify         │
│  Downloaded     │
│  Packages       │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Create NSO     │
│  Package Job    │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Install        │
│  Packages       │
│  in NSO         │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Update PB      │
│  Status         │
└─────────────────┘
```

## Error Handling and Retry Logic

### Controller Error Handling

```go
func (r *NSOReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    nso := &orchestrationv1alpha1.NSO{}
    if err := r.Get(ctx, req.NamespacedName, nso); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // Handle reconciliation with retries
    if err := r.reconcileNSO(ctx, nso); err != nil {
        // Log error and update status
        r.Log.Error(err, "Failed to reconcile NSO", "nso", nso.Name)
        
        // Update error condition
        condition := metav1.Condition{
            Type:    "Ready",
            Status:  metav1.ConditionFalse,
            Reason:  "ReconcileError",
            Message: err.Error(),
        }
        meta.SetStatusCondition(&nso.Status.Conditions, condition)
        r.Status().Update(ctx, nso)
        
        // Exponential backoff retry
        return ctrl.Result{RequeueAfter: time.Minute * 2}, err
    }
    
    return ctrl.Result{}, nil
}
```

### Retry Strategies

Different retry strategies for different operations:

```go
type RetryConfig struct {
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
    MaxRetries   int
}

var (
    // Fast retry for transient errors
    QuickRetry = RetryConfig{
        InitialDelay: time.Second * 5,
        MaxDelay:     time.Minute * 2,
        Multiplier:   1.5,
        MaxRetries:   5,
    }
    
    // Slower retry for resource creation
    SlowRetry = RetryConfig{
        InitialDelay: time.Minute,
        MaxDelay:     time.Minute * 10,
        Multiplier:   2.0,
        MaxRetries:   3,
    }
)
```

## Metrics and Observability

### Controller Metrics

The operator exposes Prometheus metrics:

```go
var (
    reconcileCount = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "nso_operator_reconcile_total",
            Help: "Total number of reconciliations",
        },
        []string{"controller", "result"},
    )
    
    reconcileDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "nso_operator_reconcile_duration_seconds",
            Help: "Time spent reconciling",
        },
        []string{"controller"},
    )
)
```

### Health Checks

The operator includes health check endpoints:

```go
func (r *NSOReconciler) SetupWithManager(mgr ctrl.Manager) error {
    // Add health checks
    if err := mgr.AddHealthzCheck("nso-controller", r.healthCheck); err != nil {
        return err
    }
    
    return ctrl.NewControllerManagedBy(mgr).
        For(&orchestrationv1alpha1.NSO{}).
        Owns(&appsv1.Deployment{}).
        Owns(&corev1.Service{}).
        Complete(r)
}

func (r *NSOReconciler) healthCheck(req *http.Request) error {
    // Perform health check logic
    if r.isHealthy() {
        return nil
    }
    return fmt.Errorf("controller is not healthy")
}
```

## Security Considerations

### RBAC Configuration

The operator requires specific RBAC permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nso-operator-role
rules:
# NSO and PackageBundle resources
- apiGroups: ["orchestration.cisco.com"]
  resources: ["nsos", "packagebundles"]
  verbs: ["*"]
# Kubernetes resources the operator manages
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["*"]
- apiGroups: [""]
  resources: ["services", "configmaps", "secrets", "persistentvolumeclaims"]
  verbs: ["*"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["*"]
```

### Security Context

The operator runs with restricted security context:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
```

## Extension Points

### Webhooks

The operator supports admission webhooks for validation:

```go
func (r *NSO) ValidateCreate() error {
    return r.validateNSO()
}

func (r *NSO) ValidateUpdate(old runtime.Object) error {
    return r.validateNSO()
}

func (r *NSO) validateNSO() error {
    if r.Spec.Replicas < 0 {
        return fmt.Errorf("replicas must be >= 0")
    }
    
    if r.Spec.Image == "" {
        return fmt.Errorf("image is required")
    }
    
    return nil
}
```

### Custom Finalizers

For cleanup operations:

```go
const NSOFinalizer = "nso.orchestration.cisco.com/finalizer"

func (r *NSOReconciler) handleFinalizer(ctx context.Context, nso *orchestrationv1alpha1.NSO) error {
    if nso.ObjectMeta.DeletionTimestamp.IsZero() {
        // Add finalizer if not present
        if !controllerutil.ContainsFinalizer(nso, NSOFinalizer) {
            controllerutil.AddFinalizer(nso, NSOFinalizer)
            return r.Update(ctx, nso)
        }
    } else {
        // Handle deletion
        if controllerutil.ContainsFinalizer(nso, NSOFinalizer) {
            if err := r.cleanupNSO(ctx, nso); err != nil {
                return err
            }
            
            controllerutil.RemoveFinalizer(nso, NSOFinalizer)
            return r.Update(ctx, nso)
        }
    }
    
    return nil
}
```

## Testing Architecture

### Unit Testing

Controllers use fake clients for unit testing:

```go
func TestNSOController_Reconcile(t *testing.T) {
    scheme := runtime.NewScheme()
    _ = orchestrationv1alpha1.AddToScheme(scheme)
    _ = appsv1.AddToScheme(scheme)
    
    client := fake.NewClientBuilder().
        WithScheme(scheme).
        Build()
    
    reconciler := &NSOReconciler{
        Client: client,
        Scheme: scheme,
    }
    
    // Test reconciliation logic
}
```

### Integration Testing

Uses envtest for integration testing:

```go
func TestNSOController_Integration(t *testing.T) {
    testEnv := &envtest.Environment{
        CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
    }
    
    cfg, err := testEnv.Start()
    require.NoError(t, err)
    
    defer testEnv.Stop()
    
    // Run integration tests
}
```

## Performance Considerations

### Resource Management

- Use resource limits for operator pods
- Implement efficient reconciliation with minimal API calls
- Use caching for frequently accessed resources
- Implement batch operations where possible

### Scalability

- Support horizontal scaling with leader election
- Implement efficient watch filters
- Use pagination for large resource lists
- Optimize memory usage with proper garbage collection

## Related Documentation

- [Development Setup](development-setup.md) - Local development environment
- [Testing Guide](testing.md) - Running and writing tests
- [Contributing](contributing.md) - How to contribute to the project
- [API Reference](../api-reference/) - Detailed API documentation
