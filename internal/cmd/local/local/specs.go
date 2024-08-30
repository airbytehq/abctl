package local

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ingress creates an ingress type for defining the webapp ingress rules.
func ingress(host string) *networkingv1.Ingress {
	var ingressClassName = "nginx"

	// Always add a localhost and host.docker.internal route.
	// This is necessary to ensure that this code can verify the Airbyte installation via `localhost`.
	// Additionally, make it easy for dockerized applications (Airflow) to talk with Airbyte over the docker host.
	rules := []networkingv1.IngressRule{ingressRule("localhost"), ingressRule("host.docker.internal")}
	// If a host that isn't `localhost` was provided, create a second rule for that host.
	// This is required to support non-local installation, such as running on an EC2 instance.
	if host != "localhost" {
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
