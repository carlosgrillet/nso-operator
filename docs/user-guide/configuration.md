# Configuration Guide

This guide covers advanced configuration options for the NSO Operator, NSO instances, and PackageBundles.

## Operator Configuration

### Environment Variables

Configure the NSO Operator behavior using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_ADDR` | `:8080` | Metrics server bind address |
| `ENABLE_LEADER_ELECTION` | `false` | Enable leader election for HA |
| `HEALTH_PROBE_ADDR` | `:8081` | Health probe server address |
| `RECONCILE_TIMEOUT` | `10m` | Maximum reconciliation time |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `NAMESPACE` | `` | Operator watch namespace (empty = all) |
| `NSO_IMAGE_REGISTRY` | `` | Default registry for NSO images |
| `PACKAGE_DOWNLOAD_TIMEOUT` | `5m` | Package download timeout |

### Operator Deployment Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nso-operator-controller-manager
  namespace: nso-operator-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    spec:
      containers:
      - name: manager
        image: your-registry/nso-operator:latest
        env:
        - name: LOG_LEVEL
          value: \"debug\"
        - name: RECONCILE_TIMEOUT
          value: \"15m\"
        - name: NSO_IMAGE_REGISTRY
          value: \"your-registry.com\"
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
```

## NSO Configuration

### Resource Management

Configure CPU and memory resources:

```yaml
spec:
  resources:
    limits:
      memory: \"4Gi\"
      cpu: \"2000m\"
      # Ephemeral storage limit
      ephemeral-storage: \"10Gi\"
    requests:
      memory: \"2Gi\"
      cpu: \"1000m\"
      ephemeral-storage: \"5Gi\"
```

### Java Virtual Machine Configuration

Tune JVM settings for NSO:

```yaml
spec:
  config:
    # Java heap size
    javaHeapSize: \"2G\"
    
    # Additional JVM options
    javaOpts: \"-XX:+UseG1GC -XX:MaxGCPauseMillis=200\"
    
    # Java system properties
    javaProperties:
      \"java.awt.headless\": \"true\"
      \"file.encoding\": \"UTF-8\"
```

### NSO-specific Configuration

```yaml
spec:
  config:
    # NSO runtime configuration
    nsoConfig: |
      <config xmlns=\"http://tail-f.com/ns/config/1.0\">
        <!-- Enable Web UI -->
        <webui xmlns=\"http://tail-f.com/ns/webui\">
          <enabled>true</enabled>
          <transport>
            <tcp>
              <enabled>true</enabled>
              <ip>0.0.0.0</ip>
              <port>8080</port>
            </tcp>
          </transport>
        </webui>
        
        <!-- RESTCONF configuration -->
        <restconf xmlns=\"http://tail-f.com/ns/restconf\">
          <enabled>true</enabled>
        </restconf>
        
        <!-- NETCONF configuration -->
        <netconf-north-bound xmlns=\"http://tail-f.com/ns/netconf-nb\">
          <enabled>true</enabled>
          <transport>
            <ssh>
              <enabled>true</enabled>
              <ip>0.0.0.0</ip>
              <port>2022</port>
            </ssh>
          </transport>
        </netconf-north-bound>
      </config>
    
    # Log configuration
    logConfig: |
      <config xmlns=\"http://tail-f.com/ns/config/1.0\">
        <logs xmlns=\"http://tail-f.com/ns/logs\">
          <syslog-config>
            <facility>local7</facility>
          </syslog-config>
          <audit-log>
            <enabled>true</enabled>
          </audit-log>
        </logs>
      </config>
```

### Environment Variables

Pass environment variables to NSO containers:

```yaml\nspec:\n  env:\n    - name: NSO_LOG_LEVEL\n      value: \"info\"\n    - name: CUSTOM_SETTING\n      value: \"production\"\n    - name: DATABASE_URL\n      valueFrom:\n        secretKeyRef:\n          name: database-credentials\n          key: url\n    - name: NODE_NAME\n      valueFrom:\n        fieldRef:\n          fieldPath: spec.nodeName\n```

### Volume Mounts

Mount additional volumes for configuration or data:

```yaml\nspec:\n  volumeMounts:\n    - name: custom-config\n      mountPath: /etc/nso/custom\n      readOnly: true\n    - name: certificates\n      mountPath: /etc/ssl/nso\n      readOnly: true\n  \n  volumes:\n    - name: custom-config\n      configMap:\n        name: nso-custom-config\n    - name: certificates\n      secret:\n        secretName: nso-certificates\n```

### Init Containers

Run initialization tasks before NSO starts:

```yaml\nspec:\n  initContainers:\n    - name: setup-database\n      image: postgres:13\n      command:\n        - sh\n        - -c\n        - |\n          echo \"Setting up database...\"\n          # Database initialization logic\n      env:\n        - name: PGHOST\n          value: postgres-service\n    \n    - name: download-config\n      image: curlimages/curl:latest\n      command:\n        - sh\n        - -c\n        - |\n          curl -o /shared/nso.conf https://config.example.com/nso.conf\n      volumeMounts:\n        - name: shared-config\n          mountPath: /shared\n```

## Storage Configuration\n\n### Persistent Volume Claims\n\n```yaml\nspec:\n  storage:\n    size: \"100Gi\"\n    storageClassName: \"fast-ssd\"\n    accessModes:\n      - ReadWriteOnce\n    \n    # Volume selector (optional)\n    selector:\n      matchLabels:\n        type: nso-storage\n    \n    # Data source (for cloning)\n    dataSource:\n      name: nso-snapshot\n      kind: VolumeSnapshot\n      apiGroup: snapshot.storage.k8s.io\n```\n\n### Multiple Volumes\n\nConfigure multiple persistent volumes:\n\n```yaml\nspec:\n  storage:\n    volumes:\n      - name: data\n        size: \"50Gi\"\n        storageClassName: \"fast-ssd\"\n        mountPath: \"/nso/data\"\n      - name: logs\n        size: \"20Gi\"\n        storageClassName: \"standard\"\n        mountPath: \"/nso/logs\"\n      - name: backups\n        size: \"100Gi\"\n        storageClassName: \"backup-storage\"\n        mountPath: \"/nso/backups\"\n```

## Networking Configuration\n\n### Service Types\n\n```yaml\nspec:\n  service:\n    # LoadBalancer for external access\n    type: LoadBalancer\n    \n    # Load balancer specific configuration\n    loadBalancerIP: \"203.0.113.10\"\n    loadBalancerSourceRanges:\n      - \"10.0.0.0/8\"\n      - \"192.168.0.0/16\"\n    \n    # Additional annotations\n    annotations:\n      service.beta.kubernetes.io/aws-load-balancer-type: \"nlb\"\n      cloud.google.com/neg: '{\"ingress\": true}'\n```\n\n### Ingress Configuration\n\n```yaml\napiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: nso-ingress\n  annotations:\n    nginx.ingress.kubernetes.io/rewrite-target: /\n    nginx.ingress.kubernetes.io/ssl-redirect: \"true\"\nspec:\n  tls:\n    - hosts:\n        - nso.example.com\n      secretName: nso-tls\n  rules:\n    - host: nso.example.com\n      http:\n        paths:\n          - path: /\n            pathType: Prefix\n            backend:\n              service:\n                name: my-nso\n                port:\n                  number: 8080\n```\n\n### Network Policies\n\n```yaml\napiVersion: networking.k8s.io/v1\nkind: NetworkPolicy\nmetadata:\n  name: nso-network-policy\nspec:\n  podSelector:\n    matchLabels:\n      app: nso\n  policyTypes:\n    - Ingress\n    - Egress\n  ingress:\n    - from:\n        - podSelector:\n            matchLabels:\n              app: nso-client\n      ports:\n        - protocol: TCP\n          port: 8080\n        - protocol: TCP\n          port: 2022\n  egress:\n    - to: []\n      ports:\n        - protocol: TCP\n          port: 22  # SSH to network devices\n        - protocol: TCP\n          port: 443 # HTTPS\n```\n\n## Security Configuration\n\n### Pod Security Context\n\n```yaml\nspec:\n  securityContext:\n    # Run as non-root user\n    runAsNonRoot: true\n    runAsUser: 1000\n    runAsGroup: 1000\n    fsGroup: 1000\n    \n    # Security capabilities\n    seccompProfile:\n      type: RuntimeDefault\n    \n    # Supplemental groups\n    supplementalGroups:\n      - 2000\n```\n\n### Container Security Context\n\n```yaml\nspec:\n  containerSecurityContext:\n    # Capabilities\n    capabilities:\n      drop:\n        - ALL\n      add:\n        - NET_BIND_SERVICE\n    \n    # Privilege escalation\n    allowPrivilegeEscalation: false\n    \n    # Read-only root filesystem\n    readOnlyRootFilesystem: true\n    \n    # User/group override\n    runAsUser: 1001\n```\n\n### TLS Configuration\n\nConfigure TLS for NSO services:\n\n```yaml\nspec:\n  tls:\n    # Enable TLS\n    enabled: true\n    \n    # Certificate source\n    certificate:\n      secretName: nso-tls-cert\n      # or auto-generate\n      autoGenerate: true\n      \n    # TLS configuration\n    config:\n      minVersion: \"1.2\"\n      cipherSuites:\n        - \"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384\"\n        - \"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256\"\n```\n\n## High Availability Configuration\n\n### Multi-replica Setup\n\n```yaml\nspec:\n  replicas: 3\n  \n  # Anti-affinity for spreading replicas\n  affinity:\n    podAntiAffinity:\n      preferredDuringSchedulingIgnoredDuringExecution:\n        - weight: 100\n          podAffinityTerm:\n            labelSelector:\n              matchLabels:\n                app: nso\n            topologyKey: kubernetes.io/hostname\n  \n  # Shared storage for HA\n  storage:\n    size: \"100Gi\"\n    storageClassName: \"shared-storage\"\n    accessModes:\n      - ReadWriteMany\n```\n\n### Readiness and Liveness Probes\n\n```yaml\nspec:\n  probes:\n    readiness:\n      httpGet:\n        path: /health\n        port: 8080\n        scheme: HTTP\n      initialDelaySeconds: 30\n      periodSeconds: 10\n      timeoutSeconds: 5\n      successThreshold: 1\n      failureThreshold: 3\n    \n    liveness:\n      tcpSocket:\n        port: 2022\n      initialDelaySeconds: 60\n      periodSeconds: 30\n      timeoutSeconds: 10\n      failureThreshold: 3\n    \n    startup:\n      httpGet:\n        path: /health\n        port: 8080\n      initialDelaySeconds: 10\n      periodSeconds: 10\n      timeoutSeconds: 5\n      failureThreshold: 30\n```\n\n## PackageBundle Configuration\n\n### Advanced Source Configuration\n\n```yaml\napiVersion: orchestration.cisco.com/v1alpha1\nkind: PackageBundle\nmetadata:\n  name: advanced-packages\nspec:\n  source:\n    git:\n      url: \"https://github.com/your-org/packages.git\"\n      ref: \"main\"\n      depth: 1\n      \n      # Submodules\n      submodules: true\n      \n      # LFS support\n      lfs: true\n      \n      # Custom Git configuration\n      config:\n        \"http.sslverify\": \"false\"\n        \"core.longpaths\": \"true\"\n  \n  # Package processing\n  processing:\n    # Compression format\n    compression: \"gzip\"\n    \n    # Package validation\n    validation:\n      enabled: true\n      schema: \"nso-package-v2\"\n    \n    # Custom transformations\n    transforms:\n      - name: \"update-versions\"\n        script: |\n          #!/bin/bash\n          sed -i 's/VERSION_PLACEHOLDER/v2.1.0/g' */src/Makefile\n```\n\n### Conditional Updates\n\n```yaml\nspec:\n  updatePolicy:\n    conditions:\n      - type: \"TimeWindow\"\n        window:\n          start: \"02:00\"\n          end: \"04:00\"\n          timezone: \"UTC\"\n          days: [\"monday\", \"wednesday\", \"friday\"]\n      \n      - type: \"HealthCheck\"\n        healthCheck:\n          endpoint: \"http://my-nso:8080/health\"\n          timeout: \"30s\"\n      \n      - type: \"ManualApproval\"\n        approval:\n          required: true\n          approvers:\n            - \"team-lead@company.com\"\n```\n\n## Monitoring and Observability\n\n### Metrics Configuration\n\n```yaml\nspec:\n  monitoring:\n    # Enable Prometheus metrics\n    prometheus:\n      enabled: true\n      port: 9090\n      path: \"/metrics\"\n      \n      # Service monitor\n      serviceMonitor:\n        enabled: true\n        interval: \"30s\"\n        labels:\n          team: \"network-ops\"\n    \n    # Custom metrics\n    customMetrics:\n      - name: \"nso_transactions_total\"\n        type: \"counter\"\n        help: \"Total number of NSO transactions\"\n      \n      - name: \"nso_device_status\"\n        type: \"gauge\"\n        help: \"NSO device connection status\"\n```\n\n### Logging Configuration\n\n```yaml\nspec:\n  logging:\n    # Log level\n    level: \"info\"\n    \n    # Log format (json, text)\n    format: \"json\"\n    \n    # Log outputs\n    outputs:\n      - type: \"stdout\"\n      - type: \"file\"\n        path: \"/var/log/nso/nso.log\"\n        maxSize: \"100Mi\"\n        maxFiles: 5\n      - type: \"syslog\"\n        facility: \"local0\"\n        tag: \"nso\"\n    \n    # Structured logging fields\n    fields:\n      service: \"nso\"\n      version: \"6.3.1\"\n```\n\n## Best Practices\n\n### Resource Planning\n\n1. **CPU**: Start with 1 core per NSO instance, scale based on device count\n2. **Memory**: 2GB minimum, add 100MB per 100 devices\n3. **Storage**: 10GB minimum, plan for logs and backups\n4. **Network**: Consider bandwidth for device management\n\n### Security Hardening\n\n1. Use non-root containers\n2. Enable read-only root filesystem\n3. Implement network policies\n4. Use TLS for all communications\n5. Regular security updates\n\n### Performance Tuning\n\n1. Tune Java heap size based on workload\n2. Use SSD storage for databases\n3. Configure appropriate readiness/liveness probes\n4. Monitor resource usage and adjust limits\n\n### Backup Strategy\n\n1. Regular database backups\n2. Configuration backups\n3. Package archive backups\n4. Test restore procedures\n\n## Troubleshooting Configuration\n\n### Common Issues\n\n**Resource Limits Too Low**\n```yaml\n# Symptoms: OOMKilled, CPU throttling\n# Solution: Increase resource limits\nspec:\n  resources:\n    limits:\n      memory: \"4Gi\"  # Increase from 2Gi\n      cpu: \"2000m\"   # Increase from 1000m\n```\n\n**Storage Issues**\n```yaml\n# Symptoms: Pods stuck pending, storage errors\n# Solution: Check StorageClass and PVC\nspec:\n  storage:\n    storageClassName: \"fast-ssd\"  # Verify exists\n    size: \"50Gi\"                  # Check available capacity\n```\n\n**Networking Problems**\n```yaml\n# Symptoms: Connection timeouts, DNS issues\n# Solution: Review service and network policy\nspec:\n  service:\n    type: ClusterIP  # Try different service type\n  # Check network policies allow required traffic\n```\n\n## Next Steps\n\n- Review [Monitoring Guide](monitoring.md) for observability setup\n- Check [Operations Guide](../operations/) for production deployment\n- Explore [Examples](../examples/) for specific use cases