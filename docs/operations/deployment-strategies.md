# Deployment Strategies

This guide covers various deployment strategies for NSO instances in production environments using the NSO Operator.

## Overview

Choosing the right deployment strategy is crucial for maintaining service availability while updating NSO instances. This guide covers:

- Rolling updates
- Blue-green deployments
- Canary deployments
- Multi-environment strategies
- Zero-downtime deployment patterns

## Rolling Updates

Rolling updates gradually replace old NSO instances with new ones, maintaining service availability throughout the process.

### Configuration

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: production-nso
spec:
  replicas: 3
  
  # Rolling update strategy
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1      # Maximum pods unavailable during update
      maxSurge: 1           # Maximum additional pods during update
  
  # Readiness probe for update validation
  probes:
    readiness:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
      failureThreshold: 3
```

### Rolling Update Process

1. **Prepare New Version**
   ```bash
   # Update NSO image version
   kubectl patch nso production-nso -p '{"spec":{"image":"cisco/nso:6.3.2"}}'
   ```

2. **Monitor Update Progress**
   ```bash
   # Watch rollout status
   kubectl rollout status deployment/production-nso
   
   # Monitor pod replacement
   kubectl get pods -l app=nso -w
   ```

3. **Validate Each Step**
   ```bash
   # Check readiness of new pods
   kubectl get pods -l app=nso -o jsonpath='{range .items[*]}{.metadata.name}{\"\\t\"}{.status.phase}{\"\\t\"}{.spec.containers[0].image}{\"\\n\"}{end}'
   ```

### Rollback Strategy

```bash
# Rollback to previous version if issues occur
kubectl rollout undo deployment/production-nso

# Check rollback status
kubectl rollout status deployment/production-nso
```

## Blue-Green Deployments

Blue-green deployments maintain two identical environments, switching traffic between them for zero-downtime updates.

### Environment Setup

```yaml
# Blue environment (current production)
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: nso-blue
  labels:
    app: nso
    version: blue
    environment: production
spec:
  replicas: 3
  image: \"cisco/nso:6.3.1\"
  
  labelSelector:
    app: nso
    version: blue
  
  serviceName: \"nso-blue-service\"
  # ... other configuration
---
# Green environment (new version)
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: nso-green
  labels:
    app: nso
    version: green
    environment: production
spec:
  replicas: 3
  image: \"cisco/nso:6.3.2\"  # New version
  
  labelSelector:
    app: nso
    version: green
  
  serviceName: \"nso-green-service\"
  # ... other configuration
---
# Production service (traffic routing)
apiVersion: v1
kind: Service
metadata:
  name: nso-production
spec:
  selector:
    app: nso
    version: blue  # Initially points to blue
  ports:
    - port: 8080
      targetPort: 8080
```

### Blue-Green Deployment Process

1. **Deploy Green Environment**
   ```bash
   # Deploy green environment alongside blue
   kubectl apply -f nso-green.yaml
   
   # Wait for green to be ready
   kubectl wait --for=condition=ready pod -l version=green --timeout=300s
   ```

2. **Validate Green Environment**
   ```bash
   # Test green environment
   kubectl port-forward svc/nso-green-service 8081:8080 &
   curl -f http://localhost:8081/health
   
   # Run smoke tests
   ./scripts/smoke-test.sh localhost:8081
   ```

3. **Switch Traffic to Green**
   ```bash
   # Update service selector to point to green
   kubectl patch service nso-production -p '{\"spec\":{\"selector\":{\"version\":\"green\"}}}'
   
   # Verify traffic switch
   kubectl get service nso-production -o yaml | grep -A 5 selector
   ```

4. **Monitor and Validate**
   ```bash
   # Monitor application metrics
   curl http://production-url/metrics
   
   # Check error rates and response times
   kubectl logs -l version=green | grep -i error
   ```

5. **Cleanup or Rollback**
   ```bash
   # If successful, remove blue environment
   kubectl delete nso nso-blue
   
   # If issues, rollback to blue
   kubectl patch service nso-production -p '{\"spec\":{\"selector\":{\"version\":\"blue\"}}}'
   ```

## Canary Deployments

Canary deployments gradually shift traffic to a new version, allowing for real-world testing with minimal risk.

### Canary Configuration

```yaml
# Production version (stable)
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: nso-stable
spec:
  replicas: 9  # 90% of traffic
  image: \"cisco/nso:6.3.1\"
  labelSelector:
    app: nso
    version: stable
---
# Canary version (new)
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: nso-canary
spec:
  replicas: 1  # 10% of traffic
  image: \"cisco/nso:6.3.2\"
  labelSelector:
    app: nso
    version: canary
---
# Service for traffic distribution
apiVersion: v1
kind: Service
metadata:
  name: nso-production
spec:
  selector:
    app: nso  # Selects both stable and canary
  ports:
    - port: 8080
      targetPort: 8080
```

### Canary Deployment Process

1. **Deploy Canary**
   ```bash
   # Deploy small canary deployment
   kubectl apply -f nso-canary.yaml
   
   # Verify canary is ready
   kubectl get pods -l version=canary
   ```

2. **Monitor Canary Metrics**
   ```bash
   # Compare error rates between versions
   kubectl logs -l version=stable | grep -c ERROR
   kubectl logs -l version=canary | grep -c ERROR
   
   # Monitor response times
   curl http://production-url/metrics | grep response_time
   ```

3. **Gradual Traffic Shift**
   ```bash
   # Increase canary traffic (20%)
   kubectl scale nso nso-stable --replicas=8
   kubectl scale nso nso-canary --replicas=2
   
   # Continue increasing if metrics look good
   kubectl scale nso nso-stable --replicas=5
   kubectl scale nso nso-canary --replicas=5
   ```

4. **Complete Rollout or Rollback**
   ```bash
   # If successful, complete rollout
   kubectl scale nso nso-stable --replicas=0
   kubectl scale nso nso-canary --replicas=10
   
   # If issues, rollback
   kubectl scale nso nso-stable --replicas=10
   kubectl scale nso nso-canary --replicas=0
   ```

## Multi-Environment Strategy

Manage multiple environments (dev, staging, production) with consistent deployment patterns.

### Environment Structure

```yaml
# Development environment
apiVersion: v1
kind: Namespace
metadata:
  name: nso-dev
  labels:
    environment: development
---
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: nso
  namespace: nso-dev
spec:
  replicas: 1
  image: \"cisco/nso:6.3.2-dev\"
  resources:
    requests:
      memory: \"1Gi\"
      cpu: \"500m\"
  env:
    - name: NSO_LOG_LEVEL
      value: \"debug\"
---
# Staging environment
apiVersion: v1
kind: Namespace
metadata:
  name: nso-staging
  labels:
    environment: staging
---
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: nso
  namespace: nso-staging
spec:
  replicas: 2
  image: \"cisco/nso:6.3.2-rc1\"
  resources:
    requests:
      memory: \"2Gi\"
      cpu: \"1000m\"
  env:
    - name: NSO_LOG_LEVEL
      value: \"info\"
---
# Production environment
apiVersion: v1
kind: Namespace
metadata:
  name: nso-production
  labels:
    environment: production
---
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: nso
  namespace: nso-production
spec:
  replicas: 3
  image: \"cisco/nso:6.3.1\"  # Stable version
  resources:
    requests:
      memory: \"4Gi\"
      cpu: \"2000m\"
  env:
    - name: NSO_LOG_LEVEL
      value: \"warn\"
```

### Promotion Pipeline

```bash
#!/bin/bash
# Deployment pipeline script

ENVIRONMENT=$1
VERSION=$2

case $ENVIRONMENT in
  \"dev\")
    kubectl set image nso/nso nso=cisco/nso:$VERSION-dev -n nso-dev
    ;;
  \"staging\")
    kubectl set image nso/nso nso=cisco/nso:$VERSION-rc -n nso-staging
    ;;
  \"production\")
    kubectl set image nso/nso nso=cisco/nso:$VERSION -n nso-production
    ;;
esac

# Wait for rollout
kubectl rollout status nso/nso -n nso-$ENVIRONMENT

# Run validation tests
./scripts/validate-deployment.sh $ENVIRONMENT
```

## Zero-Downtime Patterns

### Database Migration Strategy

For NSO updates requiring database schema changes:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: nso-migration-job
spec:
  template:
    spec:
      containers:
      - name: migration
        image: cisco/nso:6.3.2
        command:
        - /scripts/migrate-database.sh
        env:
        - name: NSO_DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: nso-db-secret
              key: url
      restartPolicy: OnFailure
```

### Configuration Hot-Reload

For configuration updates without restarts:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nso-config-v2
data:
  nso.conf: |
    # Updated configuration
    /ncs-config/logs/syslog-config/facility local6
    # ... other config changes
---
# Update NSO to use new config
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: production-nso
spec:
  nsoConfigRef: \"nso-config-v2\"  # Reference new config
  
  # Trigger config reload annotation
  annotations:
    nso.operator/config-reload: \"$(date +%s)\"
```

## Package Bundle Deployment Strategies

### Staged Package Rollouts

```yaml
# Development packages
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: dev-packages
  namespace: nso-dev
spec:
  targetName: \"nso\"
  source:
    git:
      url: \"https://github.com/company/nso-packages.git\"
      ref: \"main\"  # Latest development
---
# Staging packages
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: staging-packages
  namespace: nso-staging
spec:
  targetName: \"nso\"
  source:
    git:
      url: \"https://github.com/company/nso-packages.git\"
      ref: \"v2.1.0-rc1\"  # Release candidate
---
# Production packages
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: prod-packages
  namespace: nso-production
spec:
  targetName: \"nso\"
  source:
    git:
      url: \"https://github.com/company/nso-packages.git\"
      ref: \"v2.0.5\"  # Stable release
  
  # Production safety measures
  updatePolicy:
    autoUpdate: \"off\"
    window:
      start: \"02:00\"
      duration: \"2h\"
```

## Monitoring Deployment Health

### Deployment Metrics

```yaml
# ServiceMonitor for deployment metrics
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nso-deployment-metrics
spec:
  selector:
    matchLabels:
      app: nso
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### Health Check Automation

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: deployment-health-check
spec:
  template:
    spec:
      containers:
      - name: health-check
        image: curlimages/curl
        command:
        - /bin/sh
        - -c
        - |
          # Wait for deployment
          sleep 60
          
          # Test endpoints
          curl -f http://nso-production:8080/health || exit 1
          curl -f http://nso-production:8888/restconf/data || exit 1
          
          echo \"Deployment health check passed\"
      restartPolicy: Never
```

## Automation and CI/CD Integration

### GitOps Workflow

```yaml
# ArgoCD Application for automated deployment
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nso-production
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/company/nso-k8s-config
    targetRevision: production
    path: manifests/nso
  destination:
    server: https://kubernetes.default.svc
    namespace: nso-production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

### CI/CD Pipeline Integration

```bash
#!/bin/bash
# CI/CD deployment script

set -e

ENVIRONMENT=$1
IMAGE_TAG=$2

echo \"Deploying NSO $IMAGE_TAG to $ENVIRONMENT\"

# Update manifest with new image
sed -i \"s|image: cisco/nso:.*|image: cisco/nso:$IMAGE_TAG|g\" manifests/$ENVIRONMENT/nso.yaml

# Apply manifest
kubectl apply -f manifests/$ENVIRONMENT/

# Wait for rollout
kubectl rollout status nso/nso -n nso-$ENVIRONMENT --timeout=600s

# Run health checks
./scripts/health-check.sh $ENVIRONMENT

# Run integration tests
./scripts/integration-test.sh $ENVIRONMENT

echo \"Deployment to $ENVIRONMENT completed successfully\"
```

## Best Practices

### Pre-Deployment Checklist

- [ ] Backup current configuration and data
- [ ] Validate new image in lower environments
- [ ] Check resource availability
- [ ] Verify dependent services are healthy
- [ ] Confirm rollback plan
- [ ] Notify stakeholders of maintenance window

### During Deployment

- [ ] Monitor deployment progress continuously
- [ ] Watch for error spikes in metrics
- [ ] Validate functionality at each stage
- [ ] Keep rollback procedure ready
- [ ] Maintain communication with stakeholders

### Post-Deployment

- [ ] Verify all functionality works correctly
- [ ] Monitor performance and error rates
- [ ] Update documentation
- [ ] Clean up old resources
- [ ] Document lessons learned

### Rollback Considerations

Always have a rollback plan:

```bash
# Quick rollback commands
kubectl rollout undo deployment/nso-production
kubectl patch service nso-production -p '{\"spec\":{\"selector\":{\"version\":\"previous\"}}}'
```

## Related Documentation

- [Configuration Guide](../user-guide/configuration.md) - Advanced NSO configuration
- [Monitoring Guide](../user-guide/monitoring.md) - Deployment monitoring
- [Backup and Recovery](backup-recovery.md) - Data protection strategies
- [Security Guide](security.md) - Security considerations for deployments