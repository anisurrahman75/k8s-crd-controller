package Controller

import (
	"context"
	"fmt"
	controllerv1 "github.com/anisurrahman75/k8s-crd-controller/pkg/apis/mycrd.k8s/v1alpha1"
	clientset "github.com/anisurrahman75/k8s-crd-controller/pkg/client/clientset/versioned"
	informer "github.com/anisurrahman75/k8s-crd-controller/pkg/client/informers/externalversions/mycrd.k8s/v1alpha1"
	lister "github.com/anisurrahman75/k8s-crd-controller/pkg/client/listers/mycrd.k8s/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	deploymentInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: ctrl.enqueAppscode,
			UpdateFunc: func(oldObj, newObj interface{}) {
				ctrl.enqueAppscode(newObj)
			},
			DeleteFunc: func(obj interface{}) {
				ctrl.enqueAppscode(obj)
			},
		})
	return ctrl
}

// enqueueAppsCode takes an AppsCode resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than AppsCode.
func (c *Controller) enqueAppscode(obj interface{}) {
	log.Println("Enqueueing AppsCode...")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	fmt.Println("From enqueAppscode Function: ", key)
	c.workQueue.AddRateLimited(key)
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shut down the workqueue and wait for
// workers to finish processing their current work items.

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.workQueue.ShutDown()

	// Start the informersFactories to begin populating the informer caches
	log.Println("Starting AppsCode Controller")

	// wait fot the cache to be sunced before starting workers
	log.Println("Waiting to infomer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.AppsCodeSynced); !ok {
		log.Printf("Failed to wait for cache to syn\n")
	}
	log.Println("Starting Workers")
	// Launch Worker to process AppsCode Resources
	go wait.Until(c.runWorker, time.Second, stopCh) // TODO
	<-stopCh
	log.Println("Shutting Down Workers")
}

// runWorker is a long-running function that will continually call the
// procesNextWorkItem function in order to read and process a messege on the workque
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {

	}

}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workQueue.Get()
	if shutdown {
		return false
	}
	// we wrap this block in a func, so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off period.
		defer c.workQueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// AppsCode  resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		// Finally, if no error occurs we Forget this item, so it does not
		// get queued again until another change happens.
		c.workQueue.Forget(obj)
		log.Printf("successfully synced '%s'\n", key)
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
	fmt.Println("\tTesting")
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	// Get the AppsCode resource with this namespace/name
	AppsCode, err := c.AppsCodeLister.AppsCodes(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// We choose to absorb the error here as the worker would requeue the
			// resource otherwise. Instead, the next time the resource is updated
			// the resource will be queued again.
			utilruntime.HandleError(fmt.Errorf("AppsCode '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	deploymentName := AppsCode.Spec.Name
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s : deployment name must be specified", key))
		return nil
	}
	// Get the deployment with the name specified in Appscode.Spec
	deployment, err := c.deploymentsLister.Deployments(namespace).Get(deploymentName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		deployment, err = c.kubeclientset.AppsV1().Deployments(AppsCode.Namespace).Create(context.TODO(), newDeployment(AppsCode), metav1.CreateOptions{})
	}
	// If an error occurs during Get/Create, we'll requeue the item, so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}
	// If this number of the replicas on the Ishtiaq resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
	if AppsCode.Spec.Replicas != nil && *AppsCode.Spec.Replicas != *deployment.Spec.Replicas {

		log.Printf("Ishtiaq %s replicas: %d, deployment replicas: %d\n", name, *AppsCode.Spec.Replicas, *deployment.Spec.Replicas)

		deployment, err = c.kubeclientset.AppsV1().Deployments(namespace).Update(context.TODO(), newDeployment(AppsCode), metav1.UpdateOptions{})
		// If an error occurs during Update, we'll requeue the item, so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		if err != nil {
			return err
		}
		// Finally, we update the status block of the Ishtiaq resource to reflect the
		// current state of the world
		err = c.updateAppsCodeStatus(AppsCode, deployment)
		if err != nil {
			return err
		}

	}

	return nil
}
func (c *Controller) updateAppsCodeStatus(appscode *controllerv1.AppsCode, deployment *appsv1.Deployment) error {

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	appscodeCopy := appscode.DeepCopy()
	appscodeCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas

	// If the CustomResource Subresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Foo resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.sampleclientset.MycrdV1alpha1().AppsCodes(appscode.Namespace).Update(context.TODO(), appscodeCopy, metav1.UpdateOptions{})
	return err
}

// newDeployment Creates a new deploment for a AppsCode Resources. It also sets
// the appropiate OwnerReferences on their resources so handleObject can discover
// the AppsCode resources that 'owns' it.
func newDeployment(appscode *controllerv1.AppsCode) *appsv1.Deployment {

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appscode.Spec.Name,
			Namespace: appscode.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: appscode.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "my-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "my-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-pod",
							Image: appscode.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: appscode.Spec.Port,
								},
							},
						},
					},
				},
			},
		},
	}

}
