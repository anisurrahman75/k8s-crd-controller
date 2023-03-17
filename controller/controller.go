package controller

import (
	"context"
	"fmt"
	clientset "github.com/anisurrahman75/k8s-crd-controller/pkg/client/clientset/versioned"
	informer "github.com/anisurrahman75/k8s-crd-controller/pkg/client/informers/externalversions/mycrd.k8s/v1alpha1"
	lister "github.com/anisurrahman75/k8s-crd-controller/pkg/client/listers/mycrd.k8s/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workQueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// AppsCode resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workQueue.Forget(obj)
		log.Println("Successfully synced", "resourceName", key)
		return nil
	}(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the AppsCode resource
// with the current status of the resource.
// implement the business logic here.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the AppsCode Resources With this namespace/name
	appsCode, err := c.appsCodeLister.AppsCodes(namespace).Get(name)

	if err != nil {
		// The AppsCode resource may no longer exist, in which case we stop processing.
		if errors.IsNotFound(err) {
			// We choose to absorb the error here as the worker would requeue the
			// resource otherwise. Instead, the next time the resource is updated
			// the resource will be queued again.
			utilruntime.HandleError(fmt.Errorf("AppsCode '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	deploymentName := appsCode.Spec.Name
	log.Println("Deployments Name: ", deploymentName)
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s : deployment name must be specified", key))
		return nil
	}
	// Get the deployment with the name specified in appscode.spec
	deployment, err := c.deploymentsLister.Deployments(namespace).Get(deploymentName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		deployment, err = c.kubeclientset.AppsV1().Deployments(appsCode.Namespace).Create(context.TODO(), newDeployment(appsCode), metav1.CreateOptions{})
	}
	// If an error occurs during Get/Create, we'll requeue the item, so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	return nil
}
