package k8s

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	networkingv1 "k8s.io/api/networking/v1"
)

func TestIngress(t *testing.T) {
	tests := []struct {
		name     string
		hosts    []string
		expHosts []string
	}{
		{
			name:     "nil hosts",
			hosts:    nil,
			expHosts: []string{""},
		},
		{
			name:     "empty hosts",
			hosts:    []string{},
			expHosts: []string{""},
		},
		{
			name:     "single new host",
			hosts:    []string{"example.test"},
			expHosts: []string{"example.test", "localhost", "host.docker.internal"},
		},
		{
			name:     "localhost",
			hosts:    []string{"localhost"},
			expHosts: []string{"localhost", "host.docker.internal"},
		},
		{
			name:     "host.docker.internal",
			hosts:    []string{"host.docker.internal"},
			expHosts: []string{"localhost", "host.docker.internal"},
		},
		{
			name:     "multiple new hosts",
			hosts:    []string{"abc.test", "xyz.test"},
			expHosts: []string{"abc.test", "localhost", "host.docker.internal", "xyz.test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actHosts := extractHosts(Ingress(tt.hosts))
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
