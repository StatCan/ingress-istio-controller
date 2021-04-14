package controller

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"istio.io/api/networking/v1beta1"
	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	v1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
)

var (
	IngressClassAnnotation = "kubernetes.io/ingress.class"
	IgnoreAnnotation       = "ingress.statcan.gc.ca/ignore"
	GatewaysAnnotation     = "ingress.statcan.gc.ca/gateways"
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

	// Check for conditions which cause us to ignore the Ingress
	ignore := false

	// If we don't have an ingress class, then let's ignore it
	if val, ok := ingress.Annotations[IngressClassAnnotation]; !ok || (c.ingressClass != "" && val != c.ingressClass) {
		klog.Infof("ingress class not set or does not match %s: %s/%s", c.ingressClass, ingress.Namespace, ingress.Name)
		ignore = true
	}

	// Explicit ignore annotation
	if val, ok := ingress.Annotations[IgnoreAnnotation]; ok {
		bval, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing %s (%t): %v", IgnoreAnnotation, bval, err)
		}
		ignore = ignore || bval
	}

	if ignore {
		// A VirtualService already exists, so let's delete it
		if vs != nil {
			klog.Infof("removing owned virtualservice: %s/%s", vs.Namespace, vs.Name)
			err := c.istioclientset.NetworkingV1beta1().VirtualServices(vs.Namespace).Delete(vs.Name, &metav1.DeleteOptions{})
			return err
		}

		klog.Infof("skipping ingress: %s/%s", ingress.Namespace, ingress.Name)
		return nil
	}

	// Identify the gateway to attach the ingress to
	gateways := []string{c.defaultGateway}

	if val, ok := ingress.Annotations[GatewaysAnnotation]; ok {
		gateways = strings.Split(val, ",")
		klog.Infof("using override gateways for %s/%s: %s", ingress.Namespace, ingress.Name, gateways)
	}

	nvs, err := c.generateVirtualService(ingress, gateways)
	if err != nil {
		return err
	}

	// If we don't have virtual service, then let's make one
	if vs == nil {
		_, err = c.istioclientset.NetworkingV1beta1().VirtualServices(ingress.Namespace).Create(nvs)
		if err != nil {
			return err
		}
	} else if !reflect.DeepEqual(vs.Spec, nvs.Spec) {
		klog.Infof("updated virtual service %s/%s", vs.Namespace, vs.Name)

		// Copy the new spec
		vs.Spec = nvs.Spec

		_, err = c.istioclientset.NetworkingV1beta1().VirtualServices(ingress.Namespace).Update(vs)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) generateVirtualService(ingress *networkingv1beta1.Ingress, gateways []string) (*istionetworkingv1beta1.VirtualService, error) {
	vs := &istionetworkingv1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingress.Name,
			Namespace: ingress.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ingress, ingress.GroupVersionKind()),
			},
			Labels: ingress.Labels,
		},
		Spec: v1beta1.VirtualService{
			Gateways: gateways,
			Hosts:    []string{},
			Http:     []*v1beta1.HTTPRoute{},
		},
	}

	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			return nil, fmt.Errorf("invalid ingress rule: %s/%s - no http definition", ingress.Namespace, ingress.Name)
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
				return nil, err
			}

			var authorityMatchType v1beta1.StringMatch

			if strings.Contains(host, "*") {
				authorityMatchType = v1beta1.StringMatch{
					MatchType: &v1beta1.StringMatch_Regex{
						Regex: strings.ReplaceAll(strings.ReplaceAll(host, ".", "\\."), "*", ".*"),
					},
				}
			} else {
				authorityMatchType = v1beta1.StringMatch{
					MatchType: &v1beta1.StringMatch_Exact{
						Exact: host,
					},
				}
			}

			route := &v1beta1.HTTPRoute{
				Match: []*v1beta1.HTTPMatchRequest{
					{
						Authority: &authorityMatchType,
					},
				},
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
				route.Match[0].Uri = createStringMatch(path.Path)
			}

			vs.Spec.Http = append(vs.Spec.Http, route)
		}
	}

	return vs, nil
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
