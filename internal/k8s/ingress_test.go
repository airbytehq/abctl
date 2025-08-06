package k8s

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	networkingv1 "k8s.io/api/networking/v1"
)

func TestIngress(t *testing.T) {
	tests := []struct {
		name         string
		chartVersion string
		hosts        []string
		expHosts     []string
	}{
		{
			name:         "nil hosts v1",
			chartVersion: "1.9.9",
			hosts:        nil,
			expHosts:     []string{""},
		},
		{
			name:         "empty hosts v1",
			chartVersion: "1.9.9",
			hosts:        []string{},
			expHosts:     []string{""},
		},
		{
			name:         "single new host v1",
			chartVersion: "1.9.9",
			hosts:        []string{"example.test"},
			expHosts:     []string{"example.test", "localhost", "host.docker.internal"},
		},
		{
			name:         "localhost v1",
			chartVersion: "1.9.9",
			hosts:        []string{"localhost"},
			expHosts:     []string{"localhost", "host.docker.internal"},
		},
		{
			name:         "host.docker.internal v1",
			chartVersion: "1.9.9",
			hosts:        []string{"host.docker.internal"},
			expHosts:     []string{"localhost", "host.docker.internal"},
		},
		{
			name:         "multiple new hosts v1",
			chartVersion: "1.9.9",
			hosts:        []string{"abc.test", "xyz.test"},
			expHosts:     []string{"abc.test", "localhost", "host.docker.internal", "xyz.test"},
		},
		{
			name:         "nil hosts v2",
			chartVersion: "2.0.0",
			hosts:        nil,
			expHosts:     []string{""},
		},
		{
			name:         "empty hosts v2",
			chartVersion: "2.0.0",
			hosts:        []string{},
			expHosts:     []string{""},
		},
		{
			name:         "single new host v2",
			chartVersion: "2.0.0",
			hosts:        []string{"example.test"},
			expHosts:     []string{"example.test", "localhost", "host.docker.internal"},
		},
		{
			name:         "localhost v2",
			chartVersion: "2.0.0",
			hosts:        []string{"localhost"},
			expHosts:     []string{"localhost", "host.docker.internal"},
		},
		{
			name:         "host.docker.internal v2",
			chartVersion: "2.0.0",
			hosts:        []string{"host.docker.internal"},
			expHosts:     []string{"localhost", "host.docker.internal"},
		},
		{
			name:         "multiple new hosts v2",
			chartVersion: "2.0.0",
			hosts:        []string{"abc.test", "xyz.test"},
			expHosts:     []string{"abc.test", "localhost", "host.docker.internal", "xyz.test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actHosts := extractHosts(Ingress(tt.chartVersion, tt.hosts))
			sort.Strings(actHosts)
			sort.Strings(tt.expHosts)
			if d := cmp.Diff(tt.expHosts, actHosts); d != "" {
				t.Errorf("unexpected hosts (-want, +got):\n%v", d)
			}
		})
	}
}

// extractHosts returns the host fields from the ingress rules.
func extractHosts(ingress *networkingv1.Ingress) []string {
	var hosts []string
	for _, h := range ingress.Spec.Rules {
		hosts = append(hosts, h.Host)
	}
	return hosts
}

func TestIngressRouting(t *testing.T) {
	tests := []struct {
		name         string
		chartVersion string
		wantService  string
	}{
		{
			name:         "v1 routes to webapp",
			chartVersion: "1.9.9",
			wantService:  "airbyte-abctl-airbyte-webapp-svc",
		},
		{
			name:         "v2 routes to server",
			chartVersion: "2.0.0",
			wantService:  "airbyte-abctl-airbyte-server-svc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ingress := Ingress(tt.chartVersion, []string{"localhost"})

			// Get the default route (path: "/")
			var defaultRoute *networkingv1.HTTPIngressPath
			for _, rule := range ingress.Spec.Rules {
				for _, path := range rule.HTTP.Paths {
					if path.Path == "/" {
						defaultRoute = &path
						break
					}
				}
				if defaultRoute != nil {
					break
				}
			}

			if defaultRoute == nil {
				t.Fatal("default route (/) not found")
			}

			gotService := defaultRoute.Backend.Service.Name
			if gotService != tt.wantService {
				t.Errorf("wrong service for default route: got %s, want %s", gotService, tt.wantService)
			}
		})
	}
}
