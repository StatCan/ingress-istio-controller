package controller

import (
	// "log"

	"reflect"
	"testing"
	"time"

	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	istio "istio.io/client-go/pkg/clientset/versioned"
	istiofake "istio.io/client-go/pkg/clientset/versioned/fake"
	istioscheme "istio.io/client-go/pkg/clientset/versioned/scheme"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	istionetworkinginformers "istio.io/client-go/pkg/informers/externalversions/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	corev1informers "k8s.io/client-go/informers/core/v1"
	networkinginformers "k8s.io/client-go/informers/networking/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

type system struct {
	kubeclient  kubernetes.Interface
	istioclient istio.Interface

	ingressesInformer       networkinginformers.IngressInformer
	servicesInformer        corev1informers.ServiceInformer
	virtualServicesInformer istionetworkinginformers.VirtualServiceInformer

	controller *Controller
}

func setup(kubeObjects, istioObjects []runtime.Object) *system {
	kubeclient := fake.NewSimpleClientset(kubeObjects...)
	istioclient := istiofake.NewSimpleClientset(istioObjects...)

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclient, time.Duration(0))
	istioInformerFactory := istioinformers.NewSharedInformerFactory(istioclient, time.Duration(0))

	ingressesInformer := kubeInformerFactory.Networking().V1beta1().Ingresses()
	servicesInformer := kubeInformerFactory.Core().V1().Services()
	virtualServicesInformer := istioInformerFactory.Networking().V1beta1().VirtualServices()

	controller := NewController(
		kubeclient, nil,
		"cluster.local",
		"default-gateway", "istio", 100,
		ingressesInformer,
		servicesInformer,
		virtualServicesInformer)

	return &system{
		kubeclient:              kubeclient,
		istioclient:             istioclient,
		ingressesInformer:       ingressesInformer,
		servicesInformer:        servicesInformer,
		controller:              controller,
		virtualServicesInformer: virtualServicesInformer,
	}
}

func TestGenerateVirtualServiceBasic(t *testing.T) {
	// Setup test
	var ingress string = `
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: test-ing
  namespace: test-ns
spec:
  rules:
  - host: example.ca
    http:
      paths:
        - backend:
            serviceName: test-svc
            servicePort: 80
`

	var vs string = `
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: test-ing
  namespace: test-ns
spec:
  gateways:
    - istio-system/ingress
  hosts:
    - example.ca
  http:
    - match:
        - authority:
            exact: example.ca
      route:
        - destination:
            host: test-svc.test-ns.svc.cluster.local
            port:
              number: 80
          weight: 100
`

	// Convert to objects
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(ingress), nil, nil)

	if err != nil {
		t.Fatalf("failed creating test object: %v", err)
	}

	test := obj.(*networkingv1beta1.Ingress)

	decode = istioscheme.Codecs.UniversalDeserializer().Decode
	obj, _, err = decode([]byte(vs), nil, nil)

	if err != nil {
		t.Fatalf("failed creating expected object: %v", err)
	}

	expected := obj.(*istionetworkingv1beta1.VirtualService)

	// Generate VirtualService from Ingress
	system := setup(nil, nil)
	result, err := system.controller.generateVirtualService(test, []string{"istio-system/ingress"})
	if err != nil {
		t.Fatalf("got unexpected error converting ingress: %v", err)
	}

	// Compare generated VirtualService against expected
	if result.Name != expected.Name {
		t.Errorf("got %q for VirtualService.Name, but expected %q", result.Name, expected.Name)
	}

	if result.Namespace != expected.Namespace {
		t.Errorf("got %q for VirtualService.Namespace, but expected %q", result.Namespace, expected.Namespace)
	}

	if !metav1.IsControlledBy(result, test) {
		t.Errorf("VirtualService was not owned by the Ingress")
	}

	if !reflect.DeepEqual(result.Spec, expected.Spec) {
		t.Errorf("got %v but expected %v", result.Spec, expected.Spec)
	}
}

func TestGenerateVirtualServiceServicePortName(t *testing.T) {
	// Setup test
	var service string = `
apiVersion: v1
kind: Service
metadata:
  name: test-svc
  namespace: test-ns
spec:
  ports:
  - name: http
    port: 80
`
	var ingress string = `
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: test-ing
  namespace: test-ns
spec:
  rules:
  - host: example.ca
    http:
      paths:
        - backend:
            serviceName: test-svc
            servicePort: http
`

	var vs string = `
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: test-ing
  namespace: test-ns
spec:
  gateways:
    - istio-system/ingress
  hosts:
    - example.ca
  http:
    - match:
        - authority:
            exact: example.ca
      route:
        - destination:
            host: test-svc.test-ns.svc.cluster.local
            port:
              number: 80
          weight: 100
`
	// Convert to objects
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, _, err := decode([]byte(service), nil, nil)
	if err != nil {
		t.Fatalf("failed creating service object: %v", err)
	}
	svc := obj.(*corev1.Service)

	obj, _, err = decode([]byte(ingress), nil, nil)
	if err != nil {
		t.Fatalf("failed creating test object: %v", err)
	}
	test := obj.(*networkingv1beta1.Ingress)

	decode = istioscheme.Codecs.UniversalDeserializer().Decode
	obj, _, err = decode([]byte(vs), nil, nil)

	if err != nil {
		t.Fatalf("failed creating expected object: %v", err)
	}

	expected := obj.(*istionetworkingv1beta1.VirtualService)

	// Register service
	system := setup([]runtime.Object{svc}, nil)
	system.servicesInformer.Informer().GetIndexer().Add(svc)

	// Generate VirtualService from Ingress
	result, err := system.controller.generateVirtualService(test, []string{"istio-system/ingress"})
	if err != nil {
		t.Fatalf("got unexpected error converting ingress: %v", err)
	}

	// Compare generated VirtualService against expected
	if !reflect.DeepEqual(result.Spec, expected.Spec) {
		t.Errorf("got %v but expected %v", result.Spec, expected.Spec)
	}
}
