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
  service:
    type: NodePort
    ports:
      http: {{ .Port }}
    httpsPort:
      enable: false
  config:
    # Discover results and connection catalogs for wide schemas can be over 10m.
    # Match the platform API's own 50MB body limit (airbyte-platform-internal#20068).
    proxy-body-size: 50m
    proxy-read-timeout: "600"
    proxy-send-timeout: "600"
`))

func BuildNginxValues(port int) (string, error) {
	var buf bytes.Buffer
	err := nginxValuesTpl.Execute(&buf, map[string]any{"Port": port})
	if err != nil {
		return "", fmt.Errorf("failed to build nginx values yaml: %w", err)
	}
	return buf.String(), nil
}
