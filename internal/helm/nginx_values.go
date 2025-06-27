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
    proxy-body-size: 10m
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
