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

### Operator Statefulset Configuration

```yaml
//TODO
```

## NSO Configuration

### Resource Management

Configure CPU and memory resources:

```yaml
//TODO
```

### NSO-specific Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ncs-config
  namespace: default
data:
  ncs.conf: |
    <!-- -*- nxml -*- -->
    <!-- Example configuration file for ncs. -->

    <ncs-config xmlns="http://tail-f.com/yang/tailf-ncs-config">

      <!-- NCS can be configured to restrict access for incoming connections -->
      <!-- to the IPC listener sockets. The access check requires that -->
      <!-- connecting clients prove possession of a shared secret. -->
      <ncs-ipc-access-check>
        <enabled>false</enabled>
        <filename>${NCS_CONFIG_DIR}/ipc_access</filename>
      </ncs-ipc-access-check>

      <!-- Where to look for .fxs and snmp .bin files to load -->

      <load-path>
        <dir>${NCS_RUN_DIR}/packages</dir>
        <dir>${NCS_DIR}/etc/ncs</dir>

        <!-- To disable northbound snmp altogether -->
        <!-- comment out the path below -->
        <dir>${NCS_DIR}/etc/ncs/snmp</dir>
      </load-path>
      
      ... Rest of your config
```

### Environment Variables

Pass environment variables to NSO containers:

```yaml
//TODO
```

### Volume Mounts

Mount additional volumes for configuration or data:

```yaml
//TODO
```

### Init Containers

Run initialization tasks before NSO starts:

```yaml
//TODO
```
