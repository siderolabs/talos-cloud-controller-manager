# Metrics documentation

This document is a reflection of the current state of the exposed metrics of the Talos CCM.

## Gather metrics from talos-cloud-controller-manager

By default, the Talos CCM exposes metrics on the `https://localhost:50258/metrics` endpoint.

Enabling the metrics is done by setting the `--secure-port` and the `--authorization-always-allow-paths` flag to allow access to the `/metrics` endpoint.

```yaml
talos-cloud-controller-manager
  --authorization-always-allow-paths="/metrics"
  --secure-port=50258
```

### Helm chart values

The following values can be set in the Helm chart to expose the metrics of the Talos CCM.

```yaml
service:
  containerPort: 50258
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/scheme: "https"
    prometheus.io/port: "50258"
```

## Metrics exposed by the CCM

### Talos API calls

|Metric name|Metric type|Labels/tags|
|-----------|-----------|-----------|
|talosccm_api_request_duration_seconds|Histogram|`request`=<api_request>|
|talosccm_api_request_errors_total|Counter|`request`=<api_request>|

Example output:

```txt
talosccm_api_request_duration_seconds_bucket{request="addresses",le="0.1"} 10
talosccm_api_request_duration_seconds_bucket{request="addresses",le="0.25"} 16
talosccm_api_request_duration_seconds_bucket{request="addresses",le="0.5"} 16
talosccm_api_request_duration_seconds_bucket{request="addresses",le="1"} 16
talosccm_api_request_duration_seconds_bucket{request="addresses",le="2.5"} 16
talosccm_api_request_duration_seconds_bucket{request="addresses",le="5"} 16
talosccm_api_request_duration_seconds_bucket{request="addresses",le="10"} 16
talosccm_api_request_duration_seconds_bucket{request="addresses",le="30"} 16
talosccm_api_request_duration_seconds_bucket{request="addresses",le="+Inf"} 16
talosccm_api_request_duration_seconds_sum{request="addresses"} 1.369387789
talosccm_api_request_duration_seconds_count{request="addresses"} 16
talosccm_api_request_duration_seconds_bucket{request="platformmetadata",le="0.1"} 14
talosccm_api_request_duration_seconds_bucket{request="platformmetadata",le="0.25"} 16
talosccm_api_request_duration_seconds_bucket{request="platformmetadata",le="0.5"} 16
talosccm_api_request_duration_seconds_bucket{request="platformmetadata",le="1"} 16
talosccm_api_request_duration_seconds_bucket{request="platformmetadata",le="2.5"} 16
talosccm_api_request_duration_seconds_bucket{request="platformmetadata",le="5"} 16
talosccm_api_request_duration_seconds_bucket{request="platformmetadata",le="10"} 16
talosccm_api_request_duration_seconds_bucket{request="platformmetadata",le="30"} 16
talosccm_api_request_duration_seconds_bucket{request="platformmetadata",le="+Inf"} 16
talosccm_api_request_duration_seconds_sum{request="platformmetadata"} 1.2046141220000002
talosccm_api_request_duration_seconds_count{request="platformmetadata"} 16
```

### Certificate signing requests (CSR) approval calls

|Metric name|Metric type|Labels/tags|
|-----------|-----------|-----------|
|talosccm_csr_approval_count|Counter|`status`=<approve|deny>|

Example output:

```txt
talosccm_csr_approval_count{status="approve"} 2
```

### Transformer rules calls

|Metric name|Metric type|Labels/tags|
|-----------|-----------|-----------|
|talosccm_transformer_duration_seconds|Histogram|`type`=<type_transformation>|
|talosccm_transformer_errors_total|Counter|`type`=<type_transformation>|

Example output:

```txt
talosccm_transformer_duration_seconds_bucket{type="metadata",le="0.001"} 16
talosccm_transformer_duration_seconds_bucket{type="metadata",le="0.01"} 16
talosccm_transformer_duration_seconds_bucket{type="metadata",le="0.05"} 16
talosccm_transformer_duration_seconds_bucket{type="metadata",le="0.1"} 16
talosccm_transformer_duration_seconds_bucket{type="metadata",le="+Inf"} 16
talosccm_transformer_duration_seconds_sum{type="metadata"} 0.0012434149999999999
talosccm_transformer_duration_seconds_count{type="metadata"} 16
talosccm_transformer_errors_total{type="metadata"} 6
```
