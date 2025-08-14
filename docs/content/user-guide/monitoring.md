# Monitoring and Observability

This guide covers how to monitor NSO Operator, NSO instances, and PackageBundles using Prometheus, Grafana, and other observability tools.

## Overview

The NSO Operator provides comprehensive monitoring capabilities:

- **Operator metrics**: Performance and health of the operator itself
- **NSO instance metrics**: NSO performance, transactions, device status
- **PackageBundle metrics**: Package operations and status
- **Custom metrics**: Application-specific monitoring
- **Logs aggregation**: Centralized logging with structured output
- **Distributed tracing**: Request tracing across components

## Quick Setup

### Prerequisites

- Prometheus installed in cluster
- Grafana for visualization (optional but recommended)
- Service monitors enabled

### Enable Monitoring

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: monitored-nso
spec:
  image: "cisco/nso:6.3"
  
  # Enable monitoring
  monitoring:
    enabled: true
    prometheus:
      enabled: true
      port: 9090
      serviceMonitor:
        enabled: true
```

## Operator Metrics

### Available Metrics

The NSO Operator exposes these metrics:

| Metric Name | Type | Description |
|-------------|------|--------------|
| `nso_operator_reconcile_total` | Counter | Total reconciliations |
| `nso_operator_reconcile_duration_seconds` | Histogram | Reconciliation duration |
| `nso_operator_reconcile_errors_total` | Counter | Reconciliation errors |
| `nso_operator_nso_instances_total` | Gauge | Total NSO instances |
| `nso_operator_packagebundle_operations_total` | Counter | PackageBundle operations |
| `nso_operator_leader_election_status` | Gauge | Leader election status |

### ServiceMonitor Configuration

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nso-operator
  namespace: nso-operator-system
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: nso-operator
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
```

## NSO Instance Metrics

### Built-in NSO Metrics

NSO provides various metrics through its management interface:

```yaml
spec:
  monitoring:
    nso:
      # Enable NSO's built-in metrics
      enabled: true
      
      # Metrics endpoint configuration
      endpoint:
        port: 9001
        path: "/metrics"
        
      # Metrics to collect
      metrics:
        - "nso_transactions_total"
        - "nso_device_connections"
        - "nso_commit_queue_length"
        - "nso_memory_usage"
        - "nso_cpu_usage"
```

### Custom NSO Metrics Exporter

Deploy a custom exporter for detailed NSO metrics:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nso-metrics-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nso-metrics-exporter
  template:
    metadata:
      labels:
        app: nso-metrics-exporter
    spec:
      containers:
        - name: exporter
          image: your-registry/nso-metrics-exporter:latest
          ports:
            - containerPort: 9090
              name: metrics
          env:
            - name: NSO_URL
              value: "http://my-nso:8080"
            - name: NSO_USERNAME
              valueFrom:
                secretKeyRef:
                  name: nso-admin-secret
                  key: username
            - name: NSO_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: nso-admin-secret
                  key: password
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nso-metrics-exporter
spec:
  selector:
    matchLabels:
      app: nso-metrics-exporter
  endpoints:
    - port: metrics
      interval: 15s
```

## PackageBundle Metrics

### Package Operation Metrics

```yaml
# Automatically exposed by the operator
nso_operator_packagebundle_downloads_total{status="success|failure"}
nso_operator_packagebundle_installations_total{status="success|failure"}
nso_operator_packagebundle_update_duration_seconds
nso_operator_packagebundle_size_bytes
```

### Package Status Monitoring

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: PackageBundle
metadata:
  name: monitored-packages
spec:
  source:
    git:
      url: "https://github.com/your-org/packages.git"
  
  # Monitoring configuration
  monitoring:
    enabled: true
    healthChecks:
      - name: package-integrity
        command: ["verify-packages"]
        interval: "5m"
      - name: dependency-check
        command: ["check-dependencies"]
        interval: "1h"
```

## Prometheus Configuration

### Prometheus Rules

Create alerting rules for NSO components:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: nso-operator-rules
spec:
  groups:
    - name: nso-operator.rules
      rules:
        # Operator health
        - alert: NSOOperatorDown
          expr: up{job="nso-operator"} == 0
          for: 1m
          labels:
            severity: critical
          annotations:
            summary: "NSO Operator is down"
            description: "NSO Operator has been down for more than 1 minute"
        
        # Reconciliation errors
        - alert: NSOOperatorHighErrorRate
          expr: rate(nso_operator_reconcile_errors_total[5m]) > 0.1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High reconciliation error rate"
            description: "NSO Operator error rate is {{ $value }} errors/sec"
        
        # NSO instance health
        - alert: NSOInstanceDown
          expr: nso_operator_nso_ready == 0
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "NSO instance {{ $labels.name }} is down"
            description: "NSO instance has been unavailable for 2 minutes"
        
        # Package operations
        - alert: PackageBundleFailure
          expr: increase(nso_operator_packagebundle_operations_total{status="failure"}[5m]) > 0
          labels:
            severity: warning
          annotations:
            summary: "PackageBundle operation failed"
            description: "PackageBundle {{ $labels.name }} operation failed"
```

### Recording Rules

Create recording rules for common queries:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: nso-recording-rules
spec:
  groups:
    - name: nso-recording.rules
      interval: 30s
      rules:
        - record: nso:reconcile_rate
          expr: rate(nso_operator_reconcile_total[5m])
        
        - record: nso:error_rate
          expr: rate(nso_operator_reconcile_errors_total[5m])
        
        - record: nso:success_rate
          expr: |
            (
              rate(nso_operator_reconcile_total[5m]) -
              rate(nso_operator_reconcile_errors_total[5m])
            ) / rate(nso_operator_reconcile_total[5m])
```

## Grafana Dashboards

### NSO Operator Dashboard

Create a comprehensive Grafana dashboard:

```json
{
  "dashboard": {
    "title": "NSO Operator Overview",
    "panels": [
      {
        "title": "Reconciliation Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(rate(nso_operator_reconcile_total[5m]))"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(rate(nso_operator_reconcile_errors_total[5m]))"
          }
        ]
      },
      {
        "title": "NSO Instances",
        "type": "table",
        "targets": [
          {
            "expr": "nso_operator_nso_instances_total by (namespace, name, status)"
          }
        ]
      }
    ]
  }
}
```

### Dashboard ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nso-grafana-dashboard
  labels:
    grafana_dashboard: "1"
data:
  nso-operator.json: |
    # Dashboard JSON content here
```

## Logging

### Structured Logging Configuration

```yaml
apiVersion: orchestration.cisco.com/v1alpha1
kind: NSO
metadata:
  name: logged-nso
spec:
  logging:
    # Log level
    level: "info"
    
    # Structured JSON logging
    format: "json"
    
    # Log outputs
    outputs:
      - type: "stdout"
      - type: "file"
        path: "/var/log/nso/nso.log"
        rotation:
          maxSize: "100Mi"
          maxFiles: 10
    
    # Additional log fields
    fields:
      service: "nso"
      cluster: "production"
      version: "6.3.1"
```

### Log Aggregation with Fluentd

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-nso-config
data:
  fluent.conf: |
    <source>
      @type tail
      path /var/log/containers/*nso*.log
      pos_file /var/log/fluentd-nso.log.pos
      tag kubernetes.nso
      format json
      time_key time
      time_format %Y-%m-%dT%H:%M:%S.%NZ
    </source>
    
    <filter kubernetes.nso>
      @type kubernetes_metadata
      ca_file /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      verify_ssl true
      bearer_token_file /var/run/secrets/kubernetes.io/serviceaccount/token
    </filter>
    
    <match kubernetes.nso>
      @type elasticsearch
      host elasticsearch.logging.svc.cluster.local
      port 9200
      index_name nso-logs
      type_name _doc
    </match>
```

## Distributed Tracing

### Jaeger Integration

```yaml
spec:
  tracing:
    enabled: true
    jaeger:
      endpoint: "http://jaeger-collector.tracing:14268/api/traces"
      
    # Sampling configuration
    sampling:
      type: "probabilistic"
      param: 0.1  # 10% of requests
    
    # Service name for traces
    serviceName: "nso-operator"
```

### OpenTelemetry Configuration

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: nso-otel-collector
spec:
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    
    processors:
      batch:
    
    exporters:
      jaeger:
        endpoint: jaeger-collector:14250
        tls:
          insecure: true
    
    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [batch]
          exporters: [jaeger]
```

## Health Checks and Probes

### Readiness Probes

```yaml
spec:
  probes:
    readiness:
      httpGet:
        path: /health/ready
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
      timeoutSeconds: 5
      failureThreshold: 3
```

### Liveness Probes

```yaml
spec:
  probes:
    liveness:
      httpGet:
        path: /health/live
        port: 8080
      initialDelaySeconds: 60
      periodSeconds: 30
      timeoutSeconds: 10
      failureThreshold: 3
```

### Custom Health Checks

```yaml
spec:
  healthChecks:
    - name: database-connection
      command: ["ncs_cmd", "-c", "show packages"]
      interval: "60s"
      timeout: "10s"
      
    - name: device-connectivity
      command: ["python3", "/scripts/check_devices.py"]
      interval: "300s"
      timeout: "30s"
```

## Alerting

### Alertmanager Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: alertmanager-config
type: Opaque
stringData:
  alertmanager.yml: |
    global:
      smtp_smarthost: 'smtp.example.com:587'
      smtp_from: 'alerts@example.com'
    
    route:
      group_by: ['alertname', 'cluster']
      group_wait: 10s
      group_interval: 10s
      repeat_interval: 1h
      receiver: 'nso-team'
    
    receivers:
      - name: 'nso-team'
        email_configs:
          - to: 'nso-team@example.com'
            subject: '{{ .GroupLabels.alertname }} - {{ .Status }}'
            body: |
              {{ range .Alerts }}
              Alert: {{ .Annotations.summary }}
              Description: {{ .Annotations.description }}
              Labels: {{ range .Labels.SortedPairs }}{{ .Name }}={{ .Value }} {{ end }}
              {{ end }}
        
        slack_configs:
          - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
            channel: '#nso-alerts'
            title: 'NSO Alert'
            text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

## Performance Monitoring

### Resource Usage Tracking

```yaml
# ServiceMonitor for resource metrics
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nso-resources
spec:
  selector:
    matchLabels:
      app: nso
  endpoints:
    - port: metrics
      interval: 15s
      path: /metrics/resources
```

### Custom Performance Metrics

```python
# Example Python exporter for NSO performance
import time
import requests
from prometheus_client import start_http_server, Gauge, Counter

# Metrics definitions
device_count = Gauge('nso_devices_total', 'Total number of devices')
transaction_duration = Gauge('nso_transaction_duration_seconds', 'Transaction duration')
commit_queue_size = Gauge('nso_commit_queue_size', 'Commit queue size')

def collect_nso_metrics():
    """Collect metrics from NSO RESTCONF API"""
    try:
        # Get device count
        response = requests.get('http://nso:8080/restconf/data/tailf-ncs:devices')
        devices = response.json()
        device_count.set(len(devices.get('devices', {}).get('device', [])))
        
        # Get commit queue info
        response = requests.get('http://nso:8080/restconf/data/tailf-ncs:commit-queue')
        queue_info = response.json()
        commit_queue_size.set(queue_info.get('commit-queue', {}).get('queue-length', 0))
        
    except Exception as e:
        print(f"Error collecting metrics: {e}")

if __name__ == '__main__':
    start_http_server(8000)
    while True:
        collect_nso_metrics()
        time.sleep(30)
```

## Troubleshooting Monitoring

### Common Issues

**Metrics Not Appearing**
1. Check ServiceMonitor selectors
2. Verify Prometheus discovery
3. Check firewall/network policies
4. Review metric exposition format

**High Cardinality Metrics**
1. Limit label values
2. Use recording rules
3. Set metric retention policies
4. Monitor storage usage

**Missing Alerts**
1. Verify PrometheusRule syntax
2. Check alert evaluation
3. Review Alertmanager routing
4. Test notification channels

### Debug Commands

```bash
# Check Prometheus targets
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Visit http://localhost:9090/targets

# Check ServiceMonitor
kubectl get servicemonitor -A

# View operator metrics
curl http://localhost:8080/metrics

# Check alert rules
kubectl get prometheusrule -o yaml
```

## Next Steps

- Set up [Operations procedures](../operations/) based on monitoring
- Create custom dashboards for your specific use cases  
- Implement automated remediation based on alerts
- Review [Troubleshooting Guide](troubleshooting.md) for monitoring issues