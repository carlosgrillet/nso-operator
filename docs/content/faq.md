# Frequently Asked Questions (FAQ)

This document answers common questions about the NSO Operator.

## General Questions

### Q: What is the NSO Operator?

**A:** The NSO Operator is a Kubernetes operator that manages the lifecycle of Cisco NSO (Network Services Orchestrator) instances in Kubernetes clusters. It automates deployment, scaling, updates, and management of NSO instances and their associated package bundles.

### Q: Why use the NSO Operator instead of deploying NSO manually?

**A:** The NSO Operator provides several advantages:

- **Automation**: Automates NSO deployment, scaling, and updates
- **Best Practices**: Implements Kubernetes and NSO best practices
- **Package Management**: Simplifies NSO package installation and updates
- **Monitoring**: Built-in health checks and metrics
- **Consistency**: Ensures consistent deployments across environments
- **Self-healing**: Automatically recovers from failures

### Q: What versions of NSO are supported?

**A:** The NSO Operator supports NSO 6.0 and later versions. Check the compatibility matrix in our documentation for specific version support.

### Q: Can I run multiple NSO instances in the same cluster?

**A:** Yes, you can run multiple NSO instances in the same cluster. Each NSO resource creates an independent NSO deployment. You can deploy them in the same namespace or separate namespaces for better isolation.

## Installation and Setup

### Q: How do I install the NSO Operator?

**A:** You can install the NSO Operator using several methods:

1. **kubectl**: Apply the installation YAML directly
2. **Helm**: Use the provided Helm chart
3. **From source**: Build and deploy from the GitHub repository

See our [Installation Guide](getting-started/installation.md) for detailed instructions.

### Q: What are the minimum system requirements?

**A:** Minimum requirements:
- Kubernetes 1.20+
- 2 CPU cores and 4GB RAM per NSO instance
- Persistent storage support
- LoadBalancer or Ingress controller (for external access)

See [Prerequisites](getting-started/prerequisites.md) for complete requirements.

### Q: Do I need NSO licenses to use the operator?

**A:** Yes, you need valid Cisco NSO licenses to run NSO instances. The operator manages the deployment, but NSO itself requires proper licensing.

### Q: Can I use my existing NSO container images?

**A:** Yes, you can use any NSO container image that follows standard conventions. Specify your image in the NSO resource spec:

```yaml
spec:
  image: "your-registry/nso:your-tag"
```

## Configuration and Deployment

### Q: How do I configure NSO with custom settings?

**A:** Use ConfigMaps to provide NSO configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nso-config
data:
  nso.conf: |
    # Your NSO configuration
```

Then reference it in your NSO resource:

```yaml
spec:
  nsoConfigRef: "nso-config"
```

### Q: How do I access the NSO Web UI?

**A:** You can access the NSO Web UI by:

1. **Port forwarding** (development):
   ```bash
   kubectl port-forward svc/my-nso-service 8080:8080
   ```

2. **LoadBalancer service** (production):
   ```yaml
   spec:
     service:
       type: LoadBalancer
   ```

3. **Ingress controller** (production):
   Create an Ingress resource pointing to the NSO service.

### Q: How do I scale NSO instances?

**A:** Update the `replicas` field in your NSO resource:

```bash
kubectl patch nso my-nso -p '{"spec":{"replicas":3}}'
```

**Note**: NSO clustering requires additional configuration for shared storage and coordination.

### Q: Can I use persistent storage with NSO?

**A:** Yes, configure persistent volumes in your NSO resource:

```yaml
spec:
  volumes:
    - name: nso-data
      persistentVolumeClaim:
        claimName: nso-data-pvc
  volumeMounts:
    - name: nso-data
      mountPath: /nso/data
```

## Package Management

### Q: How do I install NSO packages?

**A:** Use PackageBundle resources to manage NSO packages:

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: my-packages
spec:
  targetName: "my-nso"
  origin: "SCM"
  source:
    url: "https://github.com/my-org/nso-packages.git"
    branch: "main"
```

### Q: Can I use private Git repositories for packages?

**A:** Yes, configure authentication credentials:

```bash
# Create SSH key secret
kubectl create secret generic git-ssh-key \
  --from-file=ssh-privatekey=/path/to/key

# Reference in PackageBundle
spec:
  credentials:
    sshKeySecretRef: "git-ssh-key"
```

### Q: How do I update packages?

**A:** Update the PackageBundle resource with a new version:

```bash
kubectl patch packagebundle my-packages -p '{"spec":{"source":{"branch":"v2.1.0"}}}'
```

### Q: Can packages be downloaded from HTTP URLs?

**A:** Yes, use the URL origin type:

```yaml
spec:
  origin: "URL"
  source:
    url: "https://packages.example.com/nso-packages.tar.gz"
```

## Troubleshooting

### Q: My NSO pod is not starting. How do I debug?

**A:** Follow these steps:

1. **Check pod status**:
   ```bash
   kubectl describe pod <nso-pod-name>
   ```

2. **Check logs**:
   ```bash
   kubectl logs <nso-pod-name>
   ```

3. **Check operator logs**:
   ```bash
   kubectl logs -n nso-operator-system deployment/nso-operator-controller-manager
   ```

4. **Verify resources**:
   ```bash
   kubectl get nso,svc,pvc
   ```

### Q: PackageBundle is stuck in "Downloading" phase. What should I do?

**A:** Check the download job:

1. **Find the job**:
   ```bash
   kubectl get jobs -l packagebundle=<packagebundle-name>
   ```

2. **Check job logs**:
   ```bash
   kubectl logs job/<job-name>
   ```

3. **Common issues**:
   - Network connectivity problems
   - Authentication failures
   - Invalid source URLs

### Q: How do I check if NSO is ready?

**A:** Check the NSO resource status:

```bash
kubectl get nso my-nso -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
```

A result of "True" means NSO is ready.

### Q: NSO Web UI is not accessible. What could be wrong?

**A:** Check these items:

1. **Service exists**:
   ```bash
   kubectl get svc my-nso-service
   ```

2. **Endpoints are available**:
   ```bash
   kubectl get endpoints my-nso-service
   ```

3. **Port forwarding works**:
   ```bash
   kubectl port-forward svc/my-nso-service 8080:8080
   ```

4. **NSO is running**:
   ```bash
   kubectl logs <nso-pod> | grep -i "web ui"
   ```

## Performance and Scaling

### Q: How much resources does NSO need?

**A:** Resource requirements depend on your use case:

- **Development**: 1 CPU, 2GB RAM
- **Production**: 2+ CPUs, 4+ GB RAM
- **Large environments**: 4+ CPUs, 8+ GB RAM

Monitor actual usage and adjust accordingly.

### Q: Can I run NSO in high availability mode?

**A:** Yes, but it requires:

- Shared persistent storage (ReadWriteMany)
- NSO Enterprise license (for clustering)
- Proper NSO cluster configuration
- Load balancer configuration

### Q: How do I monitor NSO performance?

**A:** The operator provides Prometheus metrics. Enable monitoring:

```yaml
spec:
  monitoring:
    prometheus:
      enabled: true
      serviceMonitor:
        enabled: true
```

## Security

### Q: How do I secure NSO instances?

**A:** Implement these security practices:

1. **Use Secrets for sensitive data**
2. **Enable TLS for all connections**
3. **Implement network policies**
4. **Use non-root containers**
5. **Regular security updates**

See our [Security Guide](operations/security.md) for details.

### Q: Can I use LDAP/Active Directory for NSO authentication?

**A:** Yes, configure NSO's external authentication in your NSO configuration:

```yaml
data:
  nso.conf: |
    /ncs-config/aaa/authentication/external/enabled true
    /ncs-config/aaa/authentication/external/executable /path/to/ldap-auth.sh
```

### Q: How do I rotate NSO admin passwords?

**A:** Update the admin secret:

```bash
kubectl create secret generic new-admin-secret \
  --from-literal=password=new-password \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl patch nso my-nso -p '{"spec":{"adminCredentials":{"passwordSecretRef":"new-admin-secret"}}}'
```

## Integration and Automation

### Q: Can I use the NSO Operator with GitOps?

**A:** Yes, the operator works well with GitOps tools like ArgoCD and Flux. Store your NSO and PackageBundle resources in Git and let your GitOps tool manage deployments.

### Q: How do I integrate with CI/CD pipelines?

**A:** Use kubectl or Helm in your CI/CD pipelines to deploy NSO resources:

```bash
# In your CI/CD pipeline
kubectl apply -f nso-resources.yaml
kubectl wait --for=condition=ready nso/my-nso --timeout=300s
```

### Q: Can I backup NSO data automatically?

**A:** Yes, create CronJobs for automated backups:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nso-backup
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: nso-backup:latest
            # Backup commands
```

## Advanced Topics

### Q: Can I customize the NSO container?

**A:** Yes, you can:

1. **Build custom images** with additional packages
2. **Use init containers** for setup tasks
3. **Mount custom configurations** via ConfigMaps
4. **Add custom scripts** for automation

### Q: How do I migrate from manual NSO deployments to the operator?

**A:** Follow this migration process:

1. **Export current configuration** and data
2. **Create NSO and PackageBundle resources** matching your current setup
3. **Test in development environment** first
4. **Plan downtime** for production migration
5. **Deploy operator resources** and restore data

### Q: Can I extend the operator with custom functionality?

**A:** The operator is open source and can be extended:

1. **Fork the repository**
2. **Add custom controllers** or modify existing ones
3. **Submit pull requests** for community features
4. **Build custom operators** using the same patterns

### Q: How do I contribute to the NSO Operator?

**A:** See our [Contributing Guide](developer-guide/contributing.md) for:

- Development setup
- Code contribution guidelines
- Testing requirements
- Review process

## Support and Community

### Q: Where can I get help?

**A:** Several support options are available:

1. **Documentation**: Comprehensive guides and references
2. **GitHub Issues**: Bug reports and feature requests
3. **Community Forum**: Discussion and questions
4. **Cisco Support**: For Cisco customers with support contracts

### Q: How do I report bugs?

**A:** Report bugs on our [GitHub Issues](https://github.com/your-org/nso-operator/issues) page. Include:

- NSO Operator version
- Kubernetes version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs and configurations

### Q: How do I request new features?

**A:** Submit feature requests as GitHub issues with:

- Clear description of the feature
- Use case and business justification
- Proposed implementation (if any)
- Willingness to contribute

### Q: Is commercial support available?

**A:** Not yet.

## Still Have Questions?

If you can't find the answer to your question here:

1. **Search our documentation** for more detailed information
2. **Check GitHub Issues** for similar questions
3. **Join our community discussions**
4. **Contact support** if you have a support contract

**Helpful Resources:**
- [Getting Started Guide](getting-started/)
- [User Guide](user-guide/)
- [Troubleshooting Guide](user-guide/troubleshooting.md)
- [GitHub Repository](https://github.com/your-org/nso-operator)
