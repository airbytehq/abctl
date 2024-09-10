package local

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ingress creates an ingress type for defining the webapp ingress rules.
func ingress(hosts []string) *networkingv1.Ingress {
	var ingressClassName = "nginx"

	// if no host is defined, default to an empty host
	if len(hosts) == 0 {
		hosts = append(hosts, "")
	} else {
		// If a host that isn't `localhost` was provided, create a second rule for localhost.
		// This is required to ensure we can talk to Airbyte via localhost
		if !contains(hosts, "localhost") {
			hosts = append(hosts, "localhost")
		}
		// If a host that isn't `host.docker.internal` was provided, create a second rule for localhost.
		// This is required to ensure we can talk to other containers.
		if !contains(hosts, "host.docker.internal") {
			hosts = append(hosts, "host.docker.internal")
		}
	}

	var rules []networkingv1.IngressRule
	for _, host := range hosts {
		rules = append(rules, ingressRule(host))
	}

	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      airbyteIngress,
			Namespace: airbyteNamespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules:            rules,
		},
	}
}

// ingressRule creates a rule for the host to the webapp service.
func ingressRule(host string) networkingv1.IngressRule {
	var pathType = networkingv1.PathType("Prefix")

	return networkingv1.IngressRule{
		Host: host,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     "/",
						PathType: &pathType,
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: fmt.Sprintf("%s-airbyte-webapp-svc", airbyteChartRelease),
								Port: networkingv1.ServiceBackendPort{
									Name: "http",
								},
							},
						},
					},
				},
			},
		},
	}
}

// contains returns true if the slice contains the item, otherwise false.
func contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}

	return false
}
