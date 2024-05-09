package main

import (
	"context"
	"fmt"
	v12 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type controller struct {
	clientset      kubernetes.Interface
	depLister      appslisters.DeploymentLister
	depCacheSynced cache.InformerSynced
	queue          workqueue.RateLimitingInterface
}

func newController(clientset *kubernetes.Clientset, deploymentsInformer v1.DeploymentInformer) *controller {
	c := &controller{
		clientset:      clientset,
		depLister:      deploymentsInformer.Lister(),
		depCacheSynced: deploymentsInformer.Informer().HasSynced,
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ekspose"),
	}

	deploymentsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleAdd,
		DeleteFunc: c.handleDel,
	})

	return c
}

func (c controller) handleDel(obj interface{}) {
	fmt.Println("delete was called")
	c.queue.Add(obj)
}
func (c controller) handleAdd(obj interface{}) {
	fmt.Println("add was called")
	c.queue.Add(obj)
}

func (c controller) run(ch chan struct{}) {
	fmt.Println("starting controller")
	if !cache.WaitForCacheSync(ch, c.depCacheSynced) {
		fmt.Print("waiting for cache to be synced\n")
	}

	go wait.Until(c.worker, 1*time.Second, ch)

	<-ch
}

func (c controller) worker() {
	for c.processItem() {

	}
}

func (c controller) processItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Forget(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		fmt.Printf("getting key from cahce %s\n", err.Error())
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Printf("splitting key into namespace and name %s\n", err.Error())
		return false
	}

	// check if the object has been deleted from k8s cluster
	ctx := context.Background()
	_, err = c.clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		fmt.Printf("handle delete event for dep %s\n", name)
		// delete service
		err := c.clientset.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("deleting service %s, error %s\n", name, err.Error())
			return false
		}

		//err = c.clientset.NetworkingV1().Ingresses(ns).Delete(ctx, name, metav1.DeleteOptions{})
		//if err != nil {
		//	fmt.Printf("deleting ingrss %s, error %s\n", name, err.Error())
		//	return false
		//}

		return true
	}

	err = c.syncDeployment(ns, name)
	if err != nil {
		// re-try
		fmt.Printf("syncing deployment %s\n", err.Error())
		return false
	}
	return true
}

func (c controller) syncDeployment(ns string, name string) error {
	ctx := context.Background()

	dep, err := c.depLister.Deployments(ns).Get(name)
	if err != nil {
		fmt.Printf("getting deployment from lister %s\n", err.Error())
	}
	// create service
	// we have to modify this, to figure out the port
	// our deployment's container is listening on
	fmt.Println("Creating service")
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: depLabels(*dep),
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	_, err = c.clientset.CoreV1().Services(ns).Create(ctx, &svc, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("creating service %s\n", err.Error())
	}
	// create ingress
	return nil
}

func depLabels(deployment v12.Deployment) map[string]string {
	return deployment.Spec.Template.ObjectMeta.Labels
}
