package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
)

// handleIngressStatus will synchronize the status of the Load Balancer
// of the Service the Gateway is associated with.
func (c *Controller) handleIngressStatus(ingress *networkingv1.Ingress, vs *istionetworkingv1beta1.VirtualService) (*networkingv1.Ingress, error) {
	ctx := context.Background()

	gateways, err := c.getGatewaysForVirtualService(vs)
	if err != nil {
		return ingress, err
	}

	loadBalancerStatus := corev1.LoadBalancerStatus{}

	for _, gateway := range gateways {
		services, err := c.getServicesForGateway(gateway)
		if err != nil {
			return ingress, err
		}

		for _, service := range services {
			loadBalancerStatus.Ingress = append(loadBalancerStatus.Ingress, service.Status.LoadBalancer.Ingress...)
		}
	}

	// Compare the current status to the newly generated status
	// and if they differ, apply the change.
	if !reflect.DeepEqual(ingress.Status.LoadBalancer, loadBalancerStatus) {
		klog.Infof("updating ingress status for \"%s/%s\"", ingress.Namespace, ingress.Name)
		updatedIngress := ingress.DeepCopy()
		updatedIngress.Status.LoadBalancer = loadBalancerStatus

		ingress, err = c.kubeclientset.NetworkingV1().Ingresses(updatedIngress.Namespace).UpdateStatus(ctx, updatedIngress, metav1.UpdateOptions{})
		if err != nil {
			return ingress, err
		}
	}

	return ingress, nil
}

// getGatewaysForVirtualService will get the gateways associated with the Virtual Service.
func (c *Controller) getGatewaysForVirtualService(vs *istionetworkingv1beta1.VirtualService) ([]*istionetworkingv1beta1.Gateway, error) {
	return c.getGatewaysByName(vs.Spec.Gateways, vs.Namespace)
}

// Return the Gateways based on their names. The names are in the form of "namespace/name".
// If there is no namespace prepended, the current namespace is used as the originating source.
func (c *Controller) getGatewaysByName(gatewayNames []string, currentNamespace string) ([]*istionetworkingv1beta1.Gateway, error) {
	gateways := []*istionetworkingv1beta1.Gateway{}

	for _, gatewayId := range gatewayNames {
		var gateway *istionetworkingv1beta1.Gateway
		var err error

		// Split the gatewayId into [namespace, name]
		idParts := strings.Split(gatewayId, "/")

		switch len(idParts) {
		case 1:
			gateway, err = c.gatewaysListers.Gateways(currentNamespace).Get(idParts[0])
		case 2:
			gateway, err = c.gatewaysListers.Gateways(idParts[0]).Get(idParts[1])
		default:
			return nil, fmt.Errorf("unexpected number of parts in Gateway identifier %q: %d", gatewayId, len(idParts))
		}

		// If the Gateway is not found, then ignore the error.
		// Otherwise, this is an unexpected error and return it.
		if err != nil && errors.IsNotFound(err) {
			klog.Errorf("failed to load gateway %q: %v", gatewayId, err)
			continue
		} else if err != nil {
			return nil, err
		}

		gateways = append(gateways, gateway)
	}

	return gateways, nil
}

// getServicesForGateway returns Services associated with the given Gateway.
func (c *Controller) getServicesForGateway(gateway *istionetworkingv1beta1.Gateway) ([]*corev1.Service, error) {
	selector := labels.SelectorFromSet(gateway.Spec.Selector)

	if c.scopedGateways {
		return c.servicesLister.Services(gateway.Namespace).List(selector)
	}

	return c.servicesLister.List(selector)
}
