package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	clientSet      kubernetes.Interface
	depLister      applisters.DeploymentLister
	depCacheSynced cache.InformerSynced
	queue          workqueue.RateLimitingInterface
}

func newController(cs kubernetes.Interface, deplInformer appsinformers.DeploymentInformer) *controller {
	informer := deplInformer.Informer()
	c := &controller{
		clientSet:      cs,
		depLister:      deplInformer.Lister(),
		depCacheSynced: informer.HasSynced,
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ekspose"),
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAdd,
		UpdateFunc: c.handleUpdate,
		DeleteFunc: c.deleteFunc,
	})

	return c
}

func (c *controller) run(stopCh <-chan struct{}) {

	if !cache.WaitForCacheSync(stopCh, c.depCacheSynced) {
		return
	}

	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *controller) runWorker() {
	for c.processItem() {
	}
}

func (c *controller) processItem() bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	defer c.queue.Forget(obj)

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		fmt.Printf("Error fetching key: %v", err)
		return true
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Printf("Error splitting key: %v", err)
		return false
	}

	err = c.syncDeployment(ns, name)
	if err != nil {
		fmt.Printf("Error syncing deployment: %v", err)
		return false
	}

	fmt.Println("Item processed")

	return true
}

func (c *controller) syncDeployment(ns, name string) error {
	ctx := context.Background()

	dep, err := c.depLister.Deployments(ns).Get(name)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %v", err)
	}
	fmt.Println("Deployment found")

	labels := depLabels(dep)

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: dep.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port: 80,
					Name: "http",
				},
			},
		},
	}
	_, err = c.clientSet.CoreV1().Services(ns).Create(ctx, &svc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}
	fmt.Println("Service created")
	//create service
	//create ingress
	return nil
}

func (c *controller) handleAdd(obj interface{}) {
	// Add your logic here
	fmt.Println("Add")
	c.queue.Add(obj)
}

func (c *controller) handleUpdate(oldObj, newObj interface{}) {
	// Add your logic here
	fmt.Println("Update")
}

func (c *controller) deleteFunc(obj interface{}) {
	// Add your logic here
	fmt.Println("Delete")
	c.queue.Add(obj)
}

func depLabels(dep *appsv1.Deployment) map[string]string {
	return dep.Spec.Template.Labels
}
