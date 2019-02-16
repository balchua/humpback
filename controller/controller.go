package controller

import (
	"fmt"
	"os"
	"os/signal"

	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const maxRetries = 5

// Controller object
type Controller struct {
	clientset kubernetes.Interface
	queue     workqueue.RateLimitingInterface
	informer  cache.SharedIndexInformer
}

/*
Start function starts setting up the informer.
This method also find pods with label appType=installer
*/
func Start(kubeClient *kubernetes.Clientset, namespace string, listOptions meta_v1.ListOptions) {

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().Pods(namespace).List(listOptions)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().Pods(namespace).Watch(listOptions)
			},
		},
		&api_v1.Pod{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, informer)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}

// Only act on when Pod is updated.
func newResourceController(client kubernetes.Interface, informer cache.SharedIndexInformer) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{

		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(old)
			if err == nil {
				logrus.Info("Adding to Queue")
				queue.Add(key)
			}
		},
	})

	return &Controller{
		clientset: client,
		informer:  informer,
		queue:     queue,
	}
}

// Run starts the podwatcher controller
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	logrus.Info("Starting pod-watcher controller")

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	logrus.Info("pod-watcher controller synced and ready")

	wait.Until(c.runWorker, time.Second, stopCh)
}

// HasSynced is required for the cache.Controller interface.
func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

// LastSyncResourceVersion is required for the cache.Controller interface.
func (c *Controller) LastSyncResourceVersion() string {
	return c.informer.LastSyncResourceVersion()
}

func (c *Controller) runWorker() {
	for c.processNextItem() {

	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()

	if quit {
		return false
	}

	defer c.queue.Done(key)
	isDone := c.processItem(key.(string))

	if isDone {
		// No error, reset the ratelimit counters
		c.queue.Forget(key)
	} else if c.queue.NumRequeues(key) < maxRetries {
		c.queue.AddRateLimited(key)
	}

	return isDone
}

func (c *Controller) processItem(key string) bool {
	obj, _, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		logrus.Errorf("Error fetching object with key %s from store: %v", key, err)
		return true
	}

	if obj != nil {
		pod := obj.(*api_v1.Pod)
		logrus.Infof("Running the pod %s with status %s", pod.ObjectMeta.Name, pod.Status.Phase)
		if pod.Status.Phase == api_v1.PodFailed {
			logrus.Infof("Pod (%s) on namespace (%s) status is %s", pod.ObjectMeta.Name,
				pod.ObjectMeta.Namespace, pod.Status.Phase)

			exitWithError()
		} else if pod.Status.Phase == api_v1.PodSucceeded {

			logrus.Infof("Pod (%s) on namespace (%s) status is %s", pod.ObjectMeta.Name,
				pod.ObjectMeta.Namespace, pod.Status.Phase)
			exitNoError()
		} else {
			return false
		}
	}

	return false
}

func exitWithError() {
	os.Exit(1)
}

func exitNoError() {
	os.Exit(0)
}
