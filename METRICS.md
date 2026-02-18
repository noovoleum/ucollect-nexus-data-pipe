# Metrics and Monitoring

This document describes the metrics and monitoring capabilities of the data-pipe application.

## Overview

data-pipe provides industry-standard Prometheus metrics and health check endpoints to help you monitor your data pipelines in production. These metrics can be scraped by Prometheus and visualized using tools like Grafana.

## Configuration

To enable metrics, add the `metrics` section to your pipeline configuration:

```json
{
  "pipeline": {
    "name": "my-pipeline",
    "metrics": {
      "enabled": true,
      "port": 2112
    }
  },
  "source": { ... },
  "sink": { ... }
}
```

### Configuration Options

- `enabled` (boolean): Enable or disable metrics collection and HTTP endpoints
- `port` (integer): Port number for the metrics HTTP server (default: 2112)

## Endpoints

When metrics are enabled, the following HTTP endpoints are available:

### `/metrics` - Prometheus Metrics

Returns metrics in Prometheus exposition format for scraping.

**Example:**
```bash
curl http://localhost:2112/metrics
```

### `/health` - Health Check

Returns detailed health status in JSON format.

**Example Response:**
```json
{
  "healthy": true,
  "pipeline_running": true,
  "source_connected": true,
  "sink_connected": true,
  "last_event_time": "2024-01-15T10:30:00Z",
  "uptime_seconds": 3600
}
```

**Status Codes:**
- `200 OK`: Pipeline is healthy
- `503 Service Unavailable`: Pipeline is unhealthy

### `/ready` - Readiness Probe

Returns readiness status for Kubernetes readiness probes.

**Example Response:**
```
ready
```

**Status Codes:**
- `200 OK`: Pipeline is ready (source and sink connected)
- `503 Service Unavailable`: Pipeline is not ready

### `/` - Root

Provides a simple HTML page with links to all available endpoints.

## Available Metrics

### Event Processing Metrics

#### `datapipe_events_processed_total`

Counter of successfully processed events, labeled by pipeline name and operation type.

**Labels:**
- `pipeline`: Name of the pipeline
- `operation`: Type of operation (`insert`, `update`, `delete`, `replace`)

**Example:**
```
datapipe_events_processed_total{pipeline="my-pipeline",operation="insert"} 1523
datapipe_events_processed_total{pipeline="my-pipeline",operation="update"} 847
datapipe_events_processed_total{pipeline="my-pipeline",operation="delete"} 42
```

#### `datapipe_events_errored_total`

Counter of events that failed processing, labeled by pipeline, component, and error type.

**Labels:**
- `pipeline`: Name of the pipeline
- `component`: Component where error occurred (`source`, `sink`, `transformer`)
- `error_type`: Type of error (`connection_error`, `read_error`, `write_error`, `transform_error`)

**Example:**
```
datapipe_events_errored_total{pipeline="my-pipeline",component="sink",error_type="write_error"} 3
```

#### `datapipe_event_processing_duration_seconds`

Histogram of event processing duration in seconds.

**Labels:**
- `pipeline`: Name of the pipeline
- `component`: Component being measured (`source_connect`, `sink_connect`, `transform`)

**Buckets:** Default Prometheus buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)

**Example:**
```
datapipe_event_processing_duration_seconds_bucket{pipeline="my-pipeline",component="transform",le="0.005"} 1234
datapipe_event_processing_duration_seconds_sum{pipeline="my-pipeline",component="transform"} 12.34
datapipe_event_processing_duration_seconds_count{pipeline="my-pipeline",component="transform"} 1500
```

### Status Metrics

#### `datapipe_pipeline_status`

Gauge indicating pipeline status (1 = running, 0 = stopped).

**Labels:**
- `pipeline`: Name of the pipeline (as constant label)

**Example:**
```
datapipe_pipeline_status{pipeline="my-pipeline"} 1
```

#### `datapipe_source_connected`

Gauge indicating source connection status (1 = connected, 0 = disconnected).

**Labels:**
- `pipeline`: Name of the pipeline (as constant label)

**Example:**
```
datapipe_source_connected{pipeline="my-pipeline"} 1
```

#### `datapipe_sink_connected`

Gauge indicating sink connection status (1 = connected, 0 = disconnected).

**Labels:**
- `pipeline`: Name of the pipeline (as constant label)

**Example:**
```
datapipe_sink_connected{pipeline="my-pipeline"} 1
```

## Prometheus Configuration

To scrape metrics from data-pipe, add a job to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'data-pipe'
    static_configs:
      - targets: ['localhost:2112']
    scrape_interval: 15s
```

## Kubernetes Integration

### Deployment with Metrics

Example Kubernetes deployment with metrics enabled:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: data-pipe
spec:
  replicas: 1
  selector:
    matchLabels:
      app: data-pipe
  template:
    metadata:
      labels:
        app: data-pipe
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "2112"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: data-pipe
        image: data-pipe:latest
        ports:
        - containerPort: 2112
          name: metrics
        livenessProbe:
          httpGet:
            path: /health
            port: 2112
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 2112
          initialDelaySeconds: 5
          periodSeconds: 10
```

### Service for Metrics

```yaml
apiVersion: v1
kind: Service
metadata:
  name: data-pipe-metrics
  labels:
    app: data-pipe
spec:
  ports:
  - port: 2112
    name: metrics
  selector:
    app: data-pipe
```

## Grafana Dashboards

### Example Queries

Here are some useful Prometheus queries for visualizing data-pipe metrics in Grafana:

#### Event Processing Rate

```promql
rate(datapipe_events_processed_total[5m])
```

#### Event Processing Rate by Operation

```promql
sum by (operation) (rate(datapipe_events_processed_total[5m]))
```

#### Error Rate

```promql
rate(datapipe_events_errored_total[5m])
```

#### Error Rate by Component

```promql
sum by (component) (rate(datapipe_events_errored_total[5m]))
```

#### Success Rate Percentage

```promql
(
  sum(rate(datapipe_events_processed_total[5m]))
  /
  (sum(rate(datapipe_events_processed_total[5m])) + sum(rate(datapipe_events_errored_total[5m])))
) * 100
```

#### Average Processing Duration

```promql
rate(datapipe_event_processing_duration_seconds_sum[5m]) 
/ 
rate(datapipe_event_processing_duration_seconds_count[5m])
```

#### 95th Percentile Processing Duration

```promql
histogram_quantile(0.95, rate(datapipe_event_processing_duration_seconds_bucket[5m]))
```

#### Pipeline Uptime

```promql
datapipe_pipeline_status
```

#### Connection Status

```promql
datapipe_source_connected + datapipe_sink_connected
```

## Alerting

### Example Prometheus Alert Rules

```yaml
groups:
  - name: data-pipe
    interval: 30s
    rules:
      - alert: DataPipelineDown
        expr: datapipe_pipeline_status == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Data pipeline {{ $labels.pipeline }} is down"
          description: "The data pipeline has been down for more than 1 minute"

      - alert: DataPipelineSourceDisconnected
        expr: datapipe_source_connected == 0
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Data pipeline {{ $labels.pipeline }} source disconnected"
          description: "The source connection has been lost for more than 2 minutes"

      - alert: DataPipelineSinkDisconnected
        expr: datapipe_sink_connected == 0
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Data pipeline {{ $labels.pipeline }} sink disconnected"
          description: "The sink connection has been lost for more than 2 minutes"

      - alert: DataPipelineHighErrorRate
        expr: |
          (
            sum(rate(datapipe_events_errored_total[5m]))
            /
            (sum(rate(datapipe_events_processed_total[5m])) + sum(rate(datapipe_events_errored_total[5m])))
          ) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Data pipeline {{ $labels.pipeline }} has high error rate"
          description: "Error rate is {{ $value | humanizePercentage }} over the last 5 minutes"

      - alert: DataPipelineNoEvents
        expr: rate(datapipe_events_processed_total[10m]) == 0
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Data pipeline {{ $labels.pipeline }} is not processing events"
          description: "No events have been processed in the last 10 minutes"

      - alert: DataPipelineSlowProcessing
        expr: |
          histogram_quantile(0.95, 
            rate(datapipe_event_processing_duration_seconds_bucket[5m])
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Data pipeline {{ $labels.pipeline }} is processing slowly"
          description: "95th percentile processing time is {{ $value }}s"
```

## Best Practices

1. **Monitor Event Processing Rate**: Track the rate of events processed to understand throughput and detect processing issues.

2. **Track Error Rates**: Monitor error rates to detect issues early. Set up alerts for abnormal error rates.

3. **Use Health Checks**: Configure Kubernetes liveness and readiness probes using the `/health` and `/ready` endpoints.

4. **Set Up Alerts**: Create alerts for:
   - Pipeline down
   - Connection failures
   - High error rates
   - No events processed
   - Slow processing

5. **Monitor Resource Usage**: Combine data-pipe metrics with system metrics (CPU, memory, network) for complete observability.

6. **Use Grafana Dashboards**: Create dashboards to visualize:
   - Event throughput
   - Error rates
   - Processing latency
   - Connection status
   - Pipeline uptime

7. **Regular Review**: Regularly review metrics to understand normal operating patterns and identify anomalies.

## Troubleshooting

### Metrics Server Not Starting

If the metrics server fails to start, check:

1. Port availability: Ensure port 2112 (or your configured port) is not already in use
2. Configuration: Verify `metrics.enabled` is set to `true`
3. Logs: Check application logs for error messages

### Missing Metrics

If metrics are not appearing in Prometheus:

1. Verify Prometheus can reach the metrics endpoint: `curl http://localhost:2112/metrics`
2. Check Prometheus scrape configuration
3. Verify network connectivity between Prometheus and data-pipe
4. Check Prometheus logs for scrape errors

### Inaccurate Metrics

If metrics seem inaccurate:

1. Verify the pipeline is processing events
2. Check for connection issues with source or sink
3. Review error logs for processing failures
4. Ensure metrics are being scraped at appropriate intervals

## Security Considerations

1. **Network Security**: The metrics endpoint is unauthenticated. Ensure it's not exposed to untrusted networks.

2. **Firewall Rules**: Restrict access to the metrics port to only trusted monitoring systems.

3. **Kubernetes Network Policies**: Use network policies to restrict access to the metrics port.

4. **Sensitive Data**: Metrics labels should not contain sensitive information. The current implementation only uses pipeline names and operation types.

## Example: Complete Monitoring Stack

Here's an example docker-compose setup with data-pipe, Prometheus, and Grafana:

```yaml
version: '3.8'

services:
  data-pipe:
    build: .
    volumes:
      - ./config-with-metrics.json:/root/config.json
    ports:
      - "2112:2112"
    depends_on:
      - mongodb
      - postgresql

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    depends_on:
      - prometheus

  mongodb:
    image: mongo:6
    command: mongod --replSet rs0
    ports:
      - "27017:27017"

  postgresql:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
```

With corresponding `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'data-pipe'
    static_configs:
      - targets: ['data-pipe:2112']
```

## Conclusion

The metrics and monitoring features in data-pipe provide comprehensive observability for your data pipelines. By integrating with industry-standard tools like Prometheus and Grafana, you can effectively monitor, alert, and troubleshoot your data pipelines in production environments.
