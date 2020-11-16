package controller

import (
	"fmt"

	"gopkg.in/yaml.v2"
	"istio.io/api/networking/v1beta1"
	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	v1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
)

func (c *Controller) handleVirtualService(ingress *networkingv1beta1.Ingress) error {
	// Find an existing virtual service of the same name
	vs, err := c.virtualServicesListers.VirtualServices(ingress.Namespace).Get(ingress.Name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// Check that we own this VirtualService
	if vs != nil {
		if !metav1.IsControlledBy(vs, ingress) {
			msg := fmt.Sprintf("VirtualService %q already exists and is not managed by Ingress", vs.Name)
			c.recorder.Event(ingress, v1.EventTypeWarning, "ErrResourceExists", msg)
			return fmt.Errorf("%s", msg)
		}
	}

	// If we don't have virtual service, then let's make one
	if vs == nil {
		vs = &istionetworkingv1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: ingress.GenerateName,
				Namespace:    ingress.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(ingress, schema.GroupVersionKind{
						Version: "networking.k8s.io",
						Kind:    "Ingress",
					}),
				},
				Labels: ingress.Labels,
			},
			Spec: v1beta1.VirtualService{
				Gateways: []string{c.defaultGateway},
				Hosts:    []string{},
				Http:     []*v1beta1.HTTPRoute{},
			},
		}

		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				return fmt.Errorf("invalid ingress rule: %s/%s - no http definition", ingress.Namespace, ingress.Name)
			}

			// Add the host
			host := rule.Host
			if host == "" {
				host = "*"
			}
			if !stringInArray(host, vs.Spec.Hosts) {
				vs.Spec.Hosts = append(vs.Spec.Hosts, host)
			}

			// Add the path
			for _, path := range rule.HTTP.Paths {
				servicePort, err := c.getServicePort(ingress.Namespace, path.Backend)
				if err != nil {
					return err
				}

				route := &v1beta1.HTTPRoute{
					Route: []*v1beta1.HTTPRouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: fmt.Sprintf("%s.%s.svc.%s", path.Backend.ServiceName, ingress.Namespace, c.clusterDomain),
								Port: &v1beta1.PortSelector{
									Number: servicePort,
								},
							},
							Weight: int32(c.defaultWeight),
						},
					},
				}

				if path.Path != "" {
					route.Match = []*v1beta1.HTTPMatchRequest{
						{
							Uri: createStringMatch(path.Path),
						},
					}
				}

				vs.Spec.Http = append(vs.Spec.Http, route)
			}
		}
	}

	b, err := yaml.Marshal(vs)
	if err != nil {
		return err
	}

	klog.Infof("vs: %s", string(b))
	return fmt.Errorf("handleVirtualService: not implemented")
}

func (c *Controller) getServicePort(namespace string, backend networkingv1beta1.IngressBackend) (uint32, error) {
	switch backend.ServicePort.Type {
	case intstr.Int:
		return uint32(backend.ServicePort.IntValue()), nil
	case intstr.String:
		// Find the service and conver the service name to a port
		service, err := c.servicesLister.Services(namespace).Get(backend.ServiceName)
		if err != nil {
			return 0, err
		}

		for _, port := range service.Spec.Ports {
			if port.Name == backend.ServicePort.String() {
				return uint32(port.Port), nil
			}
		}

		return 0, fmt.Errorf("cannot find port named %q in service %s/%s", backend.ServicePort.String(), namespace, service.Name)
	default:
		return 0, fmt.Errorf("unknown backend service port type: %d", backend.ServicePort.Type)
	}
}
