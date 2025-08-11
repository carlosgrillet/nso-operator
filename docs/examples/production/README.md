# Production Examples

This directory contains production-ready examples for deploying NSO with the NSO Operator in enterprise environments.

## Examples Overview

| File | Description | Features |
|------|-------------|----------|
| `ha-nso-cluster.yaml` | High-availability NSO setup | Multiple replicas, shared storage, load balancing |
| `secure-nso.yaml` | Security-hardened NSO | TLS, RBAC, network policies, security contexts |
| `monitored-nso.yaml` | NSO with comprehensive monitoring | Prometheus metrics, health checks, alerting |
| `enterprise-packagebundle.yaml` | Enterprise package management | Private repos, authentication, validation |
| `multi-environment.yaml` | Multi-environment deployment | Separate dev/staging/prod configurations |

## Production Considerations

### Resource Planning

Production deployments should consider:

- **CPU**: 2+ cores per NSO instance for production workloads
- **Memory**: 4GB+ RAM, adjust Java heap accordingly  
- **Storage**: SSD storage for better performance
- **Network**: Dedicated network policies for security

### High Availability

Key aspects for HA deployment:

```yaml
spec:
  replicas: 3  # Odd number for quorum
  
  # Anti-affinity to spread across nodes
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - topologyKey: kubernetes.io/hostname
          labelSelector:
            matchLabels:
              app: nso
```

### Security Hardening

Production security measures:

```yaml
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 1000
    seccompProfile:
      type: RuntimeDefault
  
  containerSecurityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop:
        - ALL
```

### Monitoring Integration

Essential monitoring components:

```yaml
spec:
  monitoring:
    prometheus:
      enabled: true
      serviceMonitor:
        enabled: true
        interval: 30s
    
  probes:
    readiness:
      initialDelaySeconds: 60
      periodSeconds: 10
    liveness:
      initialDelaySeconds: 90
      periodSeconds: 30
```

## Deployment Strategies

### Blue-Green Deployment

Deploy new version alongside existing:

```bash
# Deploy blue version
kubectl apply -f production-nso-blue.yaml

# Test blue deployment
# ... validation steps ...

# Switch traffic to blue
kubectl patch service nso-production --patch '{"spec":{"selector":{"version":"blue"}}}'

# Remove green deployment
kubectl delete -f production-nso-green.yaml
```

### Rolling Updates

Configure rolling update strategy:

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
```

### Canary Deployment

Deploy canary version with traffic splitting:

```yaml
# Canary service (10% traffic)
apiVersion: v1
kind: Service
metadata:
  name: nso-canary
spec:
  selector:
    app: nso
    version: canary
  # Configure ingress for 10% traffic split
```

## Environment-Specific Configurations

### Development Environment

```yaml
spec:
  replicas: 1
  resources:
    requests:
      memory: "1Gi"
      cpu: "500m"
    limits:
      memory: "2Gi"
      cpu: "1000m"
  
  env:
    - name: NSO_LOG_LEVEL
      value: "debug"
```

### Staging Environment

```yaml
spec:
  replicas: 2
  resources:
    requests:
      memory: "2Gi"
      cpu: "1000m"
    limits:
      memory: "4Gi"
      cpu: "2000m"
  
  env:
    - name: NSO_LOG_LEVEL
      value: "info"
```

### Production Environment

```yaml
spec:
  replicas: 3
  resources:
    requests:
      memory: "4Gi"
      cpu: "2000m"
    limits:
      memory: "8Gi"
      cpu: "4000m"
  
  env:
    - name: NSO_LOG_LEVEL
      value: "warn"
```

## Package Management Strategy

### Production Package Sources

```yaml
spec:
  source:
    git:
      url: "git@enterprise-git.company.com:nso/packages.git"
      ref: "production-v2.1.0"  # Use specific tags
      
  # Package validation
  validation:
    enabled: true
    signatureVerification: true
    
  # Update strategy
  updatePolicy:
    autoUpdate: "off"  # Manual updates in production
```

### Package Approval Workflow

```yaml
spec:
  updatePolicy:
    approvalRequired: true
    approvers:
      - "nso-admin@company.com"
      - "network-ops-lead@company.com"
    
    # Maintenance window
    window:
      start: "02:00"
      duration: "4h"
      timezone: "UTC"
```

## Backup and Disaster Recovery

### Automated Backups

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nso-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: nso-backup:latest
            command:
            - /scripts/backup-nso.sh
            env:
            - name: NSO_HOST
              value: "nso-production"
            - name: BACKUP_LOCATION
              value: "s3://nso-backups/"
```

### Recovery Procedures

Document recovery steps:

1. **Database Recovery**
   ```bash
   # Restore from backup
   kubectl exec -it nso-pod -- ncs --restore /backups/nso-backup-latest.tar.gz
   ```

2. **Configuration Recovery**
   ```bash
   # Restore configuration
   kubectl apply -f production-config-backup.yaml
   ```

## Monitoring and Alerting

### Production Metrics

Key metrics to monitor:

- NSO instance health and readiness
- Transaction throughput and latency
- Device connection status
- Memory and CPU utilization
- Package bundle status

### Critical Alerts

```yaml
# High-priority production alerts
groups:
  - name: nso-production.rules
    rules:
      - alert: NSOInstanceDown
        expr: up{job="nso-production"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Production NSO instance is down"
```

## Deployment Checklist

Before deploying to production:

### Pre-deployment
- [ ] Resource requirements validated
- [ ] Security policies reviewed
- [ ] Backup strategy implemented
- [ ] Monitoring configured
- [ ] Network policies defined
- [ ] SSL/TLS certificates ready

### Deployment
- [ ] Staged deployment (dev → staging → prod)
- [ ] Health checks passing
- [ ] Monitoring active
- [ ] Logs flowing to aggregation system
- [ ] Backup verification

### Post-deployment
- [ ] Load testing completed
- [ ] Disaster recovery tested
- [ ] Documentation updated
- [ ] Team training completed
- [ ] Runbooks updated

## Troubleshooting Production Issues

### Performance Issues

```bash
# Check resource usage
kubectl top pods -l app=nso

# Check Java heap usage
kubectl exec nso-pod -- jstat -gc $(pgrep java)

# Review slow queries
kubectl logs nso-pod | grep "slow query"
```

### High Availability Issues

```bash
# Check replica distribution
kubectl get pods -l app=nso -o wide

# Verify anti-affinity rules
kubectl describe pod nso-pod | grep -A 10 "Node-Selectors"
```

### Security Incidents

```bash
# Check security contexts
kubectl get pod nso-pod -o jsonpath='{.spec.securityContext}'

# Audit access logs
kubectl logs nso-pod | grep "authentication\\|authorization"
```

## Best Practices Summary

1. **Always use specific image tags** in production
2. **Implement comprehensive monitoring** and alerting
3. **Use resource limits** to prevent resource exhaustion
4. **Enable security contexts** and network policies
5. **Plan for disaster recovery** with regular backup testing
6. **Use GitOps** for configuration management
7. **Implement proper RBAC** for access control
8. **Monitor package updates** and validate before deployment

## Related Documentation

- [Configuration Guide](../../user-guide/configuration.md) - Advanced configuration options
- [Monitoring Guide](../../user-guide/monitoring.md) - Comprehensive monitoring setup
- [Operations Guide](../../operations/) - Production operations procedures
- [Security Best Practices](../../operations/security.md) - Security hardening guide