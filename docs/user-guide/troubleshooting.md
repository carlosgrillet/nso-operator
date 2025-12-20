# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the NSO Operator, NSO instances, and PackageBundles.

## General Troubleshooting Steps

### 1. Check Resource Status

```bash
# Check NSO instances
kubectl get nso -A

# Check PackageBundles  
kubectl get packagebundle -A

# Get detailed information
kubectl describe nso my-nso
kubectl describe packagebundle my-packages
```

### 2. View Operator Logs

```bash
# Operator logs
kubectl logs -n nso-operator-system deployment/nso-operator-controller-manager

# Follow logs in real-time
kubectl logs -n nso-operator-system deployment/nso-operator-controller-manager -f

# Previous container logs (if operator restarted)
kubectl logs -n nso-operator-system deployment/nso-operator-controller-manager --previous
```

### 3. Check Events

```bash
# Cluster events
kubectl get events --sort-by='.lastTimestamp'

# Resource-specific events
kubectl get events --field-selector involvedObject.name=my-nso
```

## NSO Instance Issues

### NSO Pod Not Starting

**Symptoms:**
- Pod stuck in `Pending`, `ContainerCreating`, or `CrashLoopBackOff`
- NSO resource shows `Progressing` condition

**Debugging Steps:**

```bash
# Check pod status
kubectl get pods -l app=my-nso

# Describe pod for events
kubectl describe pod <pod-name>

# Check pod logs
kubectl logs <pod-name> -c nso

# Check init container logs (if any)
kubectl logs <pod-name> -c <init-container-name>
```

**Common Causes and Solutions:**

1. **Image Pull Issues**
   ```bash
   # Check if image exists and is accessible
   docker pull your-registry/nso:6.3
   
   # Verify image pull secrets
   kubectl get secret <image-pull-secret> -o yaml
   
   # Test secret
   kubectl create secret docker-registry test-secret \
     --docker-server=your-registry \
     --docker-username=username \
     --docker-password=password
   ```

2. **Resource Constraints**
   ```bash
   # Check node resources
   kubectl describe nodes
   
   # Check resource requests vs limits
   kubectl describe nso my-nso | grep -A 10 Resources
   ```

3. **Storage Issues**
   ```bash
   # Check PVC status
   kubectl get pvc
   
   # Check StorageClass
   kubectl get storageclass
   
   # Describe PVC for issues
   kubectl describe pvc <pvc-name>
   ```

4. **Admin Secret Missing**
   ```bash
   # Verify admin secret exists
   kubectl get secret nso-admin-secret
   
   # Check secret content
   kubectl get secret nso-admin-secret -o yaml
   
   # Create missing secret
   kubectl create secret generic nso-admin-secret \
     --from-literal=username=admin \
     --from-literal=password=admin123
   ```

### NSO Pod Running But Not Ready

**Symptoms:**
- Pod shows `Running` but not `Ready`
- NSO resource stuck in `Progressing` state
- Readiness probe failures

**Debugging Steps:**

```bash
# Check readiness probe
kubectl describe pod <pod-name> | grep -A 5 \"Readiness\"

# Test NSO endpoints manually
kubectl port-forward pod/<pod-name> 8080:8080
curl http://localhost:8080/health

# Check NSO startup logs
kubectl logs <pod-name> -c nso | grep -i \"startup\\|ready\\|error\"
```

**Solutions:**

1. **Increase Readiness Probe Delays**
   ```yaml
   spec:
     probes:
       readiness:
         initialDelaySeconds: 60  # Increase from 30
         timeoutSeconds: 10       # Increase from 5
   ```

2. **Check NSO Configuration**
   ```bash
   # Access NSO CLI
   kubectl exec -it <pod-name> -- ncs_cli -C -u admin
   
   # Check NSO status
   ncs --status
   ```

### NSO Performance Issues

**Symptoms:**
- Slow response times
- High CPU or memory usage
- Frequent restarts

**Debugging:**

```bash
# Check resource usage
kubectl top pod <pod-name>

# Monitor resources over time
kubectl top pod <pod-name> --containers

# Check Java heap usage
kubectl exec <pod-name> -- java -XX:+PrintGCDetails
```

**Solutions:**

1. **Adjust Resource Limits**
   ```yaml
   spec:
     resources:
       limits:
         memory: \"4Gi\"    # Increase memory
         cpu: \"2000m\"     # Increase CPU
       requests:
         memory: \"2Gi\"
         cpu: \"1000m\"
   ```

2. **Tune Java Heap**
   ```yaml
   spec:
     config:
       javaHeapSize: \"3G\"  # Adjust based on available memory
       javaOpts: \"-XX:+UseG1GC -XX:MaxGCPauseMillis=200\"
   ```

## PackageBundle Issues

### Packages Not Downloading

**Symptoms:**
- PackageBundle stuck in `Downloading` phase
- Source-related error conditions

**Debugging:**

```bash
# Check PackageBundle status
kubectl describe packagebundle my-packages

# Look for download job logs
kubectl get jobs -l packagebundle=my-packages
kubectl logs job/<download-job-name>
```

**Common Issues:**

1. **Git Repository Access**
   ```bash
   # Test Git access manually
   kubectl run debug --rm -it --image=alpine/git -- \\\n     git clone https://github.com/your-org/packages.git\n   \n   # Check authentication secret\n   kubectl get secret git-credentials -o yaml\n   ```\n\n2. **HTTP Source Issues**\n   ```bash\n   # Test HTTP endpoint\n   kubectl run debug --rm -it --image=curlimages/curl -- \\\n     curl -I https://packages.example.com/bundle.tar.gz\n   ```\n\n3. **Network Policies**\n   ```bash\n   # Check if network policies block access\n   kubectl get networkpolicy -A\n   ```\n\n### Packages Not Installing in NSO\n\n**Symptoms:**\n- Packages downloaded but not loaded in NSO\n- NSO shows package errors\n\n**Debugging:**\n\n```bash\n# Check NSO package status\nkubectl exec <nso-pod> -- ncs_cli -C -c \"show packages\"\n\n# Check package load errors\nkubectl exec <nso-pod> -- ncs_cli -C -c \"show packages package * oper-status\"\n\n# View NSO package logs\nkubectl logs <nso-pod> | grep -i package\n```\n\n**Solutions:**\n\n1. **Package Compatibility**\n   - Verify package is compatible with NSO version\n   - Check package dependencies\n   - Review package Makefile and build requirements\n\n2. **Package Installation Order**\n   ```yaml\n   spec:\n     packageManagement:\n       installOrder:\n         - \"foundation-package\"\n         - \"cisco-iosxr-cli-*\"\n         - \"service-package\"\n   ```\n\n## Service and Networking Issues\n\n### Cannot Access NSO Web UI\n\n**Symptoms:**\n- Connection timeouts to Web UI\n- Service not accessible\n\n**Debugging:**\n\n```bash\n# Check service status\nkubectl get svc my-nso\n\n# Check endpoints\nkubectl get endpoints my-nso\n\n# Test service connectivity\nkubectl run debug --rm -it --image=curlimages/curl -- \\\n  curl http://my-nso:8080\n```\n\n**Solutions:**\n\n1. **Service Configuration**\n   ```yaml\n   spec:\n     service:\n       type: LoadBalancer  # or NodePort for external access\n       ports:\n         - name: webui\n           port: 8080\n           targetPort: 8080\n   ```\n\n2. **Port Forward for Testing**\n   ```bash\n   kubectl port-forward svc/my-nso 8080:8080\n   ```\n\n3. **Check Network Policies**\n   ```bash\n   kubectl get networkpolicy -A\n   ```\n\n### NETCONF/SSH Connection Issues\n\n**Debugging:**\n\n```bash\n# Test NETCONF port\nkubectl port-forward svc/my-nso 2022:2022\ntelnet localhost 2022\n\n# Check SSH configuration\nkubectl exec <nso-pod> -- cat /etc/ssh/sshd_config\n\n# Check NSO NETCONF status\nkubectl exec <nso-pod> -- ncs_cli -C -c \"show netconf-north-bound status\"\n```\n\n## Storage and Persistence Issues\n\n### PVC Stuck in Pending\n\n**Symptoms:**\n- Persistent Volume Claim in `Pending` state\n- Pod cannot start due to volume issues\n\n**Debugging:**\n\n```bash\n# Check PVC status\nkubectl describe pvc <pvc-name>\n\n# Check available storage classes\nkubectl get storageclass\n\n# Check persistent volumes\nkubectl get pv\n```\n\n**Solutions:**\n\n1. **StorageClass Issues**\n   ```bash\n   # List available storage classes\n   kubectl get storageclass\n   \n   # Use correct storage class\n   spec:\n     storage:\n       storageClassName: \"fast-ssd\"  # Use existing class\n   ```\n\n2. **Insufficient Storage**\n   - Check cluster storage capacity\n   - Reduce requested storage size\n   - Clean up unused PVCs\n\n### Data Loss or Corruption\n\n**Prevention:**\n- Regular backups\n- Use reliable storage classes\n- Test disaster recovery procedures\n\n**Recovery:**\n```bash\n# Restore from backup\nkubectl create -f backup-restore-job.yaml\n\n# Check data integrity\nkubectl exec <nso-pod> -- ncs --check-db\n```\n\n## RBAC and Security Issues\n\n### Permission Denied Errors\n\n**Symptoms:**\n- Operator cannot create/update resources\n- \"forbidden\" errors in logs\n\n**Debugging:**\n\n```bash\n# Check service account\nkubectl get sa -n nso-operator-system\n\n# Check cluster role bindings\nkubectl get clusterrolebinding | grep nso\n\n# Test permissions\nkubectl auth can-i create nso --as=system:serviceaccount:nso-operator-system:nso-operator-controller-manager\n```\n\n**Solutions:**\n\n```yaml\n# Ensure proper RBAC\napiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  name: nso-operator-role\nrules:\n  - apiGroups: [\"orchestration.cisco.com\"]\n    resources: [\"nsos\", \"packagebundles\"]\n    verbs: [\"*\"]\n```\n\n## Resource Limit Issues\n\n### OOMKilled Errors\n\n**Symptoms:**\n- Pod restart with exit code 137\n- \"OOMKilled\" in pod events\n\n**Solutions:**\n\n```yaml\nspec:\n  resources:\n    limits:\n      memory: \"4Gi\"  # Increase memory limit\n    requests:\n      memory: \"2Gi\"\n```\n\n### CPU Throttling\n\n**Symptoms:**\n- Slow performance\n- High CPU usage metrics\n\n**Solutions:**\n\n```yaml\nspec:\n  resources:\n    limits:\n      cpu: \"2000m\"  # Increase CPU limit\n    requests:\n      cpu: \"1000m\"\n```\n\n## Operator-Specific Issues\n\n### Operator Not Reconciling\n\n**Symptoms:**\n- Resources not updating\n- Changes not applied\n\n**Debugging:**\n\n```bash\n# Check operator health\nkubectl get pods -n nso-operator-system\n\n# Check leader election\nkubectl logs -n nso-operator-system deployment/nso-operator-controller-manager | grep leader\n\n# Force reconciliation\nkubectl annotate nso my-nso kubectl.kubernetes.io/restartedAt=\"$(date +%Y-%m-%dT%H:%M:%S%z)\"\n```\n\n### Webhook Failures\n\n**Symptoms:**\n- Resource creation/updates rejected\n- Webhook timeout errors\n\n**Debugging:**\n\n```bash\n# Check webhook configuration\nkubectl get validatingwebhookconfiguration\n\n# Check webhook service\nkubectl get svc -n nso-operator-system\n\n# Test webhook endpoint\nkubectl port-forward -n nso-operator-system svc/nso-operator-webhook-service 9443:443\n```\n\n## Diagnostic Commands Cheat Sheet\n\n### Resource Status\n```bash\n# Get all NSO resources\nkubectl get nso -A -o wide\n\n# Get all PackageBundle resources\nkubectl get packagebundle -A -o wide\n\n# Get resource YAML\nkubectl get nso my-nso -o yaml\n```\n\n### Logs and Events\n```bash\n# Operator logs\nkubectl logs -n nso-operator-system deployment/nso-operator-controller-manager\n\n# Pod logs\nkubectl logs <pod-name> -c nso\n\n# Events\nkubectl get events --sort-by='.lastTimestamp'\n```\n\n### Health Checks\n```bash\n# Pod health\nkubectl describe pod <pod-name>\n\n# Service connectivity\nkubectl port-forward svc/my-nso 8080:8080\ncurl http://localhost:8080/health\n\n# NSO status\nkubectl exec <pod-name> -- ncs --status\n```\n\n### Performance\n```bash\n# Resource usage\nkubectl top pod <pod-name>\n\n# Detailed metrics\nkubectl port-forward <pod-name> 9090:9090\ncurl http://localhost:9090/metrics\n```\n\n## Getting Help\n\n### Community Resources\n- GitHub Issues: Report bugs and request features\n- Documentation: Check latest docs for updates\n- Community Forums: Ask questions and share solutions\n\n### Debug Information to Collect\n\nWhen reporting issues, include:\n\n1. **Environment Information**\n   ```bash\n   kubectl version\n   kubectl get nodes -o wide\n   ```\n\n2. **Operator Information**\n   ```bash\n   kubectl get deployment -n nso-operator-system -o yaml\n   kubectl logs -n nso-operator-system deployment/nso-operator-controller-manager --tail=100\n   ```\n\n3. **Resource Definitions**\n   ```bash\n   kubectl get nso my-nso -o yaml\n   kubectl describe nso my-nso\n   ```\n\n4. **Events and Logs**\n   ```bash\n   kubectl get events --sort-by='.lastTimestamp' --output=wide\n   kubectl logs <pod-name> --tail=100\n   ```\n\n### Support Escalation\n\nFor critical issues:\n1. Check known issues in GitHub\n2. Search documentation and FAQ\n3. Open detailed GitHub issue with debug information\n4. Contact support with issue reference\n\n## Prevention Best Practices\n\n1. **Monitoring**: Set up comprehensive monitoring and alerting\n2. **Testing**: Test configurations in development environment\n3. **Backups**: Regular backup of NSO data and configurations\n4. **Documentation**: Keep deployment documentation updated\n5. **Version Control**: Use GitOps for configuration management\n\nFor more information, see:\n- [Configuration Guide](configuration.md) for best practices\n- [Monitoring Guide](monitoring.md) for observability setup\n- [Operations Guide](../operations/) for production procedures