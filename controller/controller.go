package Controller

import (
	"fmt"
	clientset "github.com/anisurrahman75/k8s-crd-controller/pkg/client/clientset/versioned"
	informer "github.com/anisurrahman75/k8s-crd-controller/pkg/client/informers/externalversions/mycrd.k8s/v1alpha1"
	lister "github.com/anisurrahman75/k8s-crd-controller/pkg/client/listers/mycrd.k8s/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log"
	"time"
)

type Controller struct { // For controller, at least we need an informer, clientset, lister, queue

	// kubeclientset is a standard kubernetes clientset
	deploymentsLister appslisters.DeploymentLister
	kubeclientset     kubernetes.Interface
	deploymentsSynced cache.InformerSynced

	// sampleclientset is a clientset for our own API group
	sampleclientset clientset.Interface
	AppsCodeLister  lister.AppsCodeLister
	AppsCodeSynced  cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workQueue workqueue.RateLimitingInterface
}

func NewController(
	kubeclientset kubernetes.Interface,
	sampleclientset clientset.Interface,
	deploymentInformer appsinformer.DeploymentInformer,
	appCodeInformer informer.AppsCodeInformer,
) *Controller {

	ctrl := &Controller{
		kubeclientset:     kubeclientset,
		sampleclientset:   sampleclientset,
		deploymentsLister: deploymentInformer.Lister(),
		AppsCodeLister:    appCodeInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		AppsCodeSynced:    appCodeInformer.Informer().HasSynced,
	}
	log.Println("Setting up event handlers")
	// Set Up an Event handler when AppsCode resources Change...
	appCodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ctrl.enqueAppsCode(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			ctrl.enqueAppsCode(oldObj)
		},
		DeleteFunc: func(obj interface{}) {
			ctrl.enqueAppsCode(obj)
		},
	})

	return ctrl

}

// enqueueAppsCode takes an AppsCode resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than AppsCode.
func (c *Controller) enqueAppsCode(obj interface{}) {
	log.Println("Enqueueing AppsCode...")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workQueue.AddRateLimited(key)
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shut down the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Println("Starting AppsCode Controller")

	// Wait for the caches to be synced before starting workers
	log.Println("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.AppsCodeSynced); !ok {
		return fmt.Errorf("failed to wait for cache to sync")
	}

	log.Println("Starting workers")

	// Launch two workers to process AppsCode resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Println("Worker Started")
	<-stopCh
	log.Println("Shutting down workers")

	return nil
}
