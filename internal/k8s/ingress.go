package k8s

import (
	"fmt"
	"slices"

	"github.com/airbytehq/abctl/internal/common"
	"github.com/airbytehq/abctl/internal/helm"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ingress creates an ingress type for defining the webapp ingress rules.
func Ingress(chartVersion string, hosts []string) *networkingv1.Ingress {
	var ingressClassName = "nginx"

	// if no host is defined, default to an empty host
	if len(hosts) == 0 {
		hosts = append(hosts, "")
	} else {
		// If a host that isn't `localhost` was provided, create a second rule for localhost.
		// This is required to ensure we can talk to Airbyte via localhost
		if !slices.Contains(hosts, "localhost") {
			hosts = append(hosts, "localhost")
		}
		// If a host that isn't `host.docker.internal` was provided, create a second rule for localhost.
		// This is required to ensure we can talk to other containers.
		if !slices.Contains(hosts, "host.docker.internal") {
			hosts = append(hosts, "host.docker.internal")
		}
	}

	var rules []networkingv1.IngressRule
	for _, host := range hosts {
		rules = append(rules, ingressRules(chartVersion, host))
	}

	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.AirbyteIngress,
			Namespace: common.AirbyteNamespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules:            rules,
		},
	}
}

// ingressRule creates a rule for the host with proper API routing.
func ingressRules(chartVersion string, host string) networkingv1.IngressRule {
	rules := ingressRulesForV1()
	if helm.ChartIsV2Plus(chartVersion) {
		rules = ingressRulesForV2()
	}

	return networkingv1.IngressRule{
		Host:             host,
		IngressRuleValue: rules,
	}
}

func ingressRulesForV1() networkingv1.IngressRuleValue {
	var pathType = networkingv1.PathType("Prefix")

	return networkingv1.IngressRuleValue{
		HTTP: &networkingv1.HTTPIngressRuleValue{
			Paths: []networkingv1.HTTPIngressPath{
				// Route connector builder API to connector-builder-server
				{
					Path:     "/api/v1/connector_builder",
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: fmt.Sprintf("%s-airbyte-connector-builder-server-svc", common.AirbyteChartRelease),
							Port: networkingv1.ServiceBackendPort{
								Name: "http",
							},
						},
					},
				},
				// Default route for everything else to webapp
				{
					Path:     "/",
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: fmt.Sprintf("%s-airbyte-webapp-svc", common.AirbyteChartRelease),
							Port: networkingv1.ServiceBackendPort{
								Name: "http",
							},
						},
					},
				},
			},
		},
	}
}

func ingressRulesForV2() networkingv1.IngressRuleValue {
	var pathType = networkingv1.PathType("Prefix")

	return networkingv1.IngressRuleValue{
		HTTP: &networkingv1.HTTPIngressRuleValue{
			Paths: []networkingv1.HTTPIngressPath{
				// Route connector builder API to connector-builder-server
				{
					Path:     "/api/v1/connector_builder",
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: fmt.Sprintf("%s-airbyte-connector-builder-server-svc", common.AirbyteChartRelease),
							Port: networkingv1.ServiceBackendPort{
								Name: "http",
							},
						},
					},
				},
				// Default route for everything else to the server
				{
					Path:     "/",
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: fmt.Sprintf("%s-airbyte-server-svc", common.AirbyteChartRelease),
							Port: networkingv1.ServiceBackendPort{
								Name: "http",
							},
						},
					},
				},
			},
		},
	}
}
