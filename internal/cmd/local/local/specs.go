package local

import (
	"fmt"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ingress creates an ingress type for defining the webapp ingress rules.
func ingress(host string) *networkingv1.Ingress {
	var pathType = networkingv1.PathType("Prefix")
	var ingressClassName = "nginx"

	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      airbyteIngress,
			Namespace: airbyteNamespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/custom-headers": "airbyte-abctl-custom-headers",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []networkingv1.IngressRule{
				{
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
				},
			},
		},
	}
}
