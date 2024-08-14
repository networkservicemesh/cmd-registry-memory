# cmd-registry-memory

# Usage

## Environment config

* `NSM_LISTEN_ON`                - url to listen on. (default: "unix:///listen.on.socket")
* `NSM_MAX_TOKEN_LIFETIME`       - maximum lifetime of tokens (default: "10m")
* `NSM_REGISTRY_SERVER_POLICIES` - paths to files and directories that contain registry server policies (default: "etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/server/.*.rego")
* `NSM_REGISTRY_CLIENT_POLICIES` - paths to files and directories that contain registry client policies (default: "etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/client/.*.rego")
* `NSM_PROXY_REGISTRY_URL`       - url to the proxy registry that handles this domain
* `NSM_EXPIRE_PERIOD`            - period to check expired NSEs (default: "1s")
* `NSM_LOG_LEVEL`                - Log level (default: "INFO")
* `NSM_OPEN_TELEMETRY_ENDPOINT`  - OpenTelemetry Collector Endpoint (default: "otel-collector.observability.svc.cluster.local:4317")
* `NSM_METRICS_EXPORT_INTERVAL`  - interval between mertics exports (default: "10s")
* `NSM_PPROF_ENABLED`            - is pprof enabled (default: "false")
* `NSM_PPROF_LISTEN_ON`          - pprof URL to ListenAndServe (default: "localhost:6060")
