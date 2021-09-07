package controller

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"istio.io/api/networking/v1beta1"
	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

var (
	// Controller.ingressClass is expected as the value
	IngressClassAnnotation = "kubernetes.io/ingress.class"
	// boolean is expected for the value
	IgnoreAnnotation = "ingress.statcan.gc.ca/ignore"
	// Comma seperated list of Gateways in <namespace>/<name> format
	GatewaysAnnotation = "ingress.statcan.gc.ca/gateways"
	// The value verified in IngressClass.spec.controller
	IngressIstioController = "ingress.statcan.gc.ca/ingress-istio-controller"
)

func (c *Controller) handleVirtualService(ingress *networkingv1.Ingress) error {

	ctx := context.Background()

	// Find an existing virtual service of the same name
	vs, err := c.virtualServicesListers.VirtualServices(ingress.Namespace).Get(ingress.Name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// Check that we own this VirtualService
	if vs != nil {
		if !metav1.IsControlledBy(vs, ingress) {
			msg := fmt.Sprintf("VirtualService \"%s/%s\" already exists and is not managed by Ingress", vs.Namespace, vs.Name)
			c.recorder.Event(ingress, v1.EventTypeWarning, "ErrResourceExists", msg)
			return fmt.Errorf("%s", msg)
		}
	}

	// Check for conditions which cause us to handle the Ingress
	handle := false
	// Determines if the IngressClassAnnotation is set - to preserve backwards compatibility.
	hasIngressClassAnnotation := false
	var ingressClassAnnotationValue string

	// If the IngressClassAnnotation is set, handle. This takes precedence over the IngressClass.
	if ingressClassAnnotationValue, hasIngressClassAnnotation = ingress.Annotations[IngressClassAnnotation]; hasIngressClassAnnotation && c.ingressClass != "" && ingressClassAnnotationValue == c.ingressClass {
		klog.Infof("deprecated annotation \"%s=%s\" set and takes precedence over ingressClassName for Ingress: \"%s/%s\"", IngressClassAnnotation, c.ingressClass, ingress.Namespace, ingress.Name)
		handle = true
	}

	// Ensure that if it has an IngressClassAnnotation, it doesn't handle via the
	// ingressClassName so that previous behaviour is maintained.
	if !hasIngressClassAnnotation && ingress.Spec.IngressClassName != nil {
		ingressClass, err := c.ingressClassesLister.Get(*ingress.Spec.IngressClassName)
		if err != nil {
			klog.Error("error getting IngressClass %q", *ingress.Spec.IngressClassName)
			return err
		}

		if ingressClass.Spec.Controller == IngressIstioController {
			klog.Infof("IngressClass set to \"%s\" - handling Ingress", IngressIstioController)
			handle = true
		}
	}

	// Explicit ignore annotation
	if val, ok := ingress.Annotations[IgnoreAnnotation]; ok {
		bval, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing %s (%t): %v", IgnoreAnnotation, bval, err)
		}
		handle = handle && !bval
	}

	if !handle {
		// A VirtualService already exists, so let's delete it
		if vs != nil {
			klog.Infof("removing owned virtualservice: \"%s/%s\"", vs.Namespace, vs.Name)
			err := c.istioclientset.NetworkingV1beta1().VirtualServices(vs.Namespace).Delete(ctx, vs.Name, metav1.DeleteOptions{})
			return err
		}

		klog.Infof("skipping ingress: \"%s/%s\"", ingress.Namespace, ingress.Name)
		return nil
	}

	// Identify the gateway to attach the ingress to
	gateways := []string{c.defaultGateway}

	if val, ok := ingress.Annotations[GatewaysAnnotation]; ok {
		gateways = strings.Split(val, ",")
		klog.Infof("using override gateways for \"%s/%s\": %s", ingress.Namespace, ingress.Name, gateways)
	}

	nvs, err := c.generateVirtualService(ingress, vs, gateways)
	if err != nil {
		return err
	}

	// If we don't have virtual service, then let's make one
	if vs == nil {
		_, err = c.istioclientset.NetworkingV1beta1().VirtualServices(ingress.Namespace).Create(ctx, nvs, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if !reflect.DeepEqual(vs.ObjectMeta.Labels, nvs.ObjectMeta.Labels) || !reflect.DeepEqual(vs.ObjectMeta.Annotations, nvs.ObjectMeta.Annotations) || !reflect.DeepEqual(vs.Spec, nvs.Spec) {
		klog.Infof("updating virtual service \"%s/%s\"", vs.Namespace, vs.Name)

		uvs := vs.DeepCopy()

		// Copy the new spec
		uvs.ObjectMeta.Annotations = nvs.ObjectMeta.Annotations
		uvs.ObjectMeta.Labels = nvs.ObjectMeta.Labels
		uvs.Spec = nvs.Spec

		_, err = c.istioclientset.NetworkingV1beta1().VirtualServices(ingress.Namespace).Update(ctx, uvs, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func generateObjectMetadata(ingress *networkingv1.Ingress, existingVirtualService *istionetworkingv1beta1.VirtualService) (labels map[string]string, annotations map[string]string) {
	if existingVirtualService != nil {
		labels = ingress.DeepCopy().Labels
		annotations = ingress.DeepCopy().Annotations
	}

	if labels == nil {
		labels = make(map[string]string)
	}

	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Overwrite with metadata from ingress
	for k, v := range ingress.Labels {
		labels[k] = v
	}

	for k, v := range ingress.Annotations {
		annotations[k] = v
	}

	// Overwrite metadata with controller information
	labels["app.kubernetes.io/managed-by"] = controllerAgentName
	labels["app.kubernetes.io/created-by"] = controllerAgentName
	annotations["meta.statcan.gc.ca/version"] = controllerAgentVersion

	return
}

func (c *Controller) generateVirtualService(ingress *networkingv1.Ingress, existing *istionetworkingv1beta1.VirtualService, gateways []string) (*istionetworkingv1beta1.VirtualService, error) {
	labels, annotations := generateObjectMetadata(ingress, existing)

	vs := &istionetworkingv1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingress.Name,
			Namespace: ingress.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ingress, networkingv1.SchemeGroupVersion.WithKind("Ingress")),
			},
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1beta1.VirtualService{
			Gateways: gateways,
			Hosts:    []string{},
			Http:     []*v1beta1.HTTPRoute{},
		},
	}

	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			return nil, fmt.Errorf("invalid ingress rule: \"%s/%s\" - no http definition", ingress.Namespace, ingress.Name)
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
						// Convert to Regex which is required by Envoy.
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
							Host: fmt.Sprintf("%s.%s.svc.%s", path.Backend.Service.Name, ingress.Namespace, c.clusterDomain),
							Port: &v1beta1.PortSelector{
								Number: servicePort,
							},
						},
						Weight: int32(c.defaultWeight),
					},
				},
			}

			httpMatch := route.Match[0]
			httpMatch.Uri = createStringMatch(path)

			vs.Spec.Http = append(vs.Spec.Http, route)
		}
	}

	return vs, nil
}

func (c *Controller) getServicePort(namespace string, backend networkingv1.IngressBackend) (uint32, error) {
	if backend.Service.Port.Number > 0 {
		return uint32(backend.Service.Port.Number), nil
	} else if backend.Service.Port.Name != "" {
		// Find the service and conver the service name to a port
		service, err := c.servicesLister.Services(namespace).Get(backend.Service.Name)
		if err != nil {
			return 0, err
		}

		for _, port := range service.Spec.Ports {
			if port.Name == backend.Service.Port.Name {
				return uint32(port.Port), nil
			}
		}
	}

	return 0, fmt.Errorf("unknown backend service port type")
}
