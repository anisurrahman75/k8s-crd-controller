package controller

import (
	"fmt"
	clientset "github.com/anisurrahman75/k8s-crd-controller/pkg/client/clientset/versioned"
	informer "github.com/anisurrahman75/k8s-crd-controller/pkg/client/informers/externalversions/mycrd.k8s/v1alpha1"
	lister "github.com/anisurrahman75/k8s-crd-controller/pkg/client/listers/mycrd.k8s/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"log"
	"time"

	appsinformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller is the controller implementation for AppsCode resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	sampleclientset clientset.Interface

	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	appsCodeLister    lister.AppsCodeLister
	appsCodeSynced    cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workQueue workqueue.RateLimitingInterface
}

// Controller is the controller implementation for AppsCode resources

func NewController(
	kubeclientset kubernetes.Interface,
	sampleclientset clientset.Interface,
	deploymentInformer appsinformer.DeploymentInformer,
	appsCodeInformer informer.AppsCodeInformer) *Controller {

	ctrl := &Controller{
		kubeclientset:     kubeclientset,
		sampleclientset:   sampleclientset,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		appsCodeLister:    appsCodeInformer.Lister(),
		appsCodeSynced:    appsCodeInformer.Informer().HasSynced,
		workQueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "AppsCodes"),
	}
	log.Println("Setting up event Handlers")
	// Set up an event handler for when AppsCode resources change
	appsCodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ctrl.enqueueAppsCode,
		UpdateFunc: func(oldObj, newObj interface{}) {
			ctrl.enqueueAppsCode(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			ctrl.enqueueAppsCode(obj)
		},
	})
	return ctrl
}

// enqueueAppsCode takes an AppsCode resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than AppsCode.
func (c *Controller) enqueueAppsCode(obj interface{}) {
	log.Println("Enqueueing AppsCode...")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	// Adding all of changes to the workQueue
	c.workQueue.AddRateLimited(key)
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shut down the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Println("Starting AppsCode Controller")

	// Wait for the caches to be synced before starting workers
	log.Println("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.appsCodeSynced); !ok {
		return fmt.Errorf("failed to wait for cache to sync")
	}

	log.Println("Starting workers")

	// Launch  workers to process AppsCode resources
	log.Println("Worker Started")
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
	log.Println("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the workqueue.
func (c *Controller) runWorker() {
	for c.ProcessNextItem() {
	}
}
func (c *Controller) ProcessNextItem() bool {
	obj, shutdown := c.workQueue.Get()

	if shutdown {
		return false
	}

	log.Println(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}
