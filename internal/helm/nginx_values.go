package helm

import (
	"bytes"
	"fmt"
	"text/template"
)

var nginxValuesTpl = template.Must(template.New("nginx-values").Parse(`
controller:
  hostPort:
    enabled: true
    ports:
      http: 8080
      https: 8443
  service:
    type: NodePort
    ports:
      http: {{ .Port }}
    httpsPort:
      enable: false
  config:
    proxy-body-size: 10m
    proxy-read-timeout: "600"
    proxy-send-timeout: "600"
    # Use non-privileged ports for nginx
    http-port: 8080
    https-port: 8443
  # Rootless container compatibility settings
  containerSecurityContext:
    allowPrivilegeEscalation: false
    runAsNonRoot: true
    runAsUser: 101
    capabilities:
      drop:
        - ALL
      add:
        - NET_BIND_SERVICE
  # Use non-privileged ports for health checks
  containerPort:
    http: 8080
    https: 8443
    healthz: 10254
  # Configure health checks for non-privileged setup
  livenessProbe:
    httpGet:
      path: /healthz
      port: 10254
      scheme: HTTP
    initialDelaySeconds: 30
    periodSeconds: 10
    timeoutSeconds: 5
    failureThreshold: 10
  readinessProbe:
    httpGet:
      path: /healthz
      port: 10254
      scheme: HTTP
    initialDelaySeconds: 30
    periodSeconds: 10
    timeoutSeconds: 5
    failureThreshold: 10
`))

func BuildNginxValues(port int) (string, error) {
	var buf bytes.Buffer
	err := nginxValuesTpl.Execute(&buf, map[string]any{"Port": port})
	if err != nil {
		return "", fmt.Errorf("failed to build nginx values yaml: %w", err)
	}
	return buf.String(), nil
}
