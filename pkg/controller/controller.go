package controller

import (
	"context"
	"fmt"
	"time"

	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	istio "istio.io/client-go/pkg/clientset/versioned"
	istionetworkinginformers "istio.io/client-go/pkg/informers/externalversions/networking/v1beta1"
	istionetworkinglisters "istio.io/client-go/pkg/listers/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1informers "k8s.io/client-go/informers/core/v1"
	networkinginformers "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	networkinglisters "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

var controllerAgentName = "ingress-istio-controller"
var controllerAgentVersion = "development"

// Controller responds to new resources and applies the necessary configuration
type Controller struct {
	kubeclientset  kubernetes.Interface
	istioclientset istio.Interface

	clusterDomain  string
	defaultGateway string
	scopedGateways bool
	ingressClass   string
	defaultWeight  int

	ingressesLister  networkinglisters.IngressLister
	ingressesSynched cache.InformerSynced

	ingressClassesLister  networkinglisters.IngressClassLister
	ingressClassesSynched cache.InformerSynced

	servicesLister  corev1listers.ServiceLister
	servicesSynched cache.InformerSynced

	virtualServicesListers istionetworkinglisters.VirtualServiceLister
	virtualServicesSynched cache.InformerSynced

	gatewaysListers istionetworkinglisters.GatewayLister
	gatewaysSynched cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

// NewController creates a new Controller object.
func NewController(
	kubeclientset kubernetes.Interface,
	istioclientset istio.Interface,
	clusterDomain string,
	defaultGateway string,
	scopedGateways bool,
	ingressClass string,
	defaultWeight int,
	ingressesInformer networkinginformers.IngressInformer,
	ingressClassesInformer networkinginformers.IngressClassInformer,
	servicesInformer corev1informers.ServiceInformer,
	virtualServicesInformer istionetworkinginformers.VirtualServiceInformer,
	gatewaysInformer istionetworkinginformers.GatewayInformer) *Controller {
	klog.Infof("setting up controller %s: %s", controllerAgentName, controllerAgentVersion)

	// Create event broadcaster
	klog.V(4).Info("creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:          kubeclientset,
		istioclientset:         istioclientset,
		clusterDomain:          clusterDomain,
		defaultGateway:         defaultGateway,
		ingressClass:           ingressClass,
		scopedGateways:         scopedGateways,
		defaultWeight:          defaultWeight,
		ingressesLister:        ingressesInformer.Lister(),
		ingressesSynched:       ingressesInformer.Informer().HasSynced,
		ingressClassesLister:   ingressClassesInformer.Lister(),
		ingressClassesSynched:  ingressClassesInformer.Informer().HasSynced,
		servicesLister:         servicesInformer.Lister(),
		servicesSynched:        servicesInformer.Informer().HasSynced,
		virtualServicesListers: virtualServicesInformer.Lister(),
		virtualServicesSynched: virtualServicesInformer.Informer().HasSynced,
		gatewaysListers:        gatewaysInformer.Lister(),
		gatewaysSynched:        gatewaysInformer.Informer().HasSynced,
		workqueue:              workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "IngressIstio"),
		recorder:               recorder,
	}

	klog.Info("setting up event handlers")
	ingressesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueIngress,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueIngress(new)
		},
	})

	virtualServicesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			nvs := new.(*istionetworkingv1beta1.VirtualService)
			ovs := old.(*istionetworkingv1beta1.VirtualService)
			if nvs.ResourceVersion == ovs.ResourceVersion {
				// Periodic resync will send update events for all known VirtualService.
				// Two different versions of the same VirtualService will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run runs the controller.
func (c *Controller) Run(threadiness int, ctx context.Context) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("starting controller")

	klog.Info("waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.ingressesSynched, c.ingressClassesSynched, c.servicesSynched, c.virtualServicesSynched, c.gatewaysSynched); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}

	klog.Info("started workers")
	<-ctx.Done()
	klog.Info("shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error synching %q: %v, requeing", key, err)
		}

		c.workqueue.Forget(obj)
		klog.Infof("successfully synched %q", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the ingress object
	ingress, err := c.ingressesLister.Ingresses(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("ingress %q in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// Handle the VirtualService
	vs, err := c.handleVirtualServiceForIngress(ingress)
	if err != nil {
		klog.Errorf("failed to handle virtual service: %v", err)
		return err
	}

	// If the Ingress was handled, update its status.
	if vs != nil {
		_, err = c.handleIngressStatus(ingress, vs)
		if err != nil {
			klog.Errorf("failed to handle Ingress status: %v", err)
			return err
		}
	}

	return nil
}

func (c *Controller) enqueueIngress(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}

func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by an Ingress, we should not do anything more
		// with it.
		if ownerRef.Kind != "Ingress" {
			return
		}

		ingress, err := c.ingressesLister.Ingresses(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of ingress '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueIngress(ingress)
		return
	}
}
