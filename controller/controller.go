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
	pod       *api_v1.Pod
	tail      *Tail
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
	signal.Notify(sigterm,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go shutdownHook(c, sigterm)
	exitChan := make(chan int)

	<-exitChan
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
		c.pod = pod
		c.initTail()
		logrus.Infof("Running the pod %s with status %s", pod.ObjectMeta.Name, pod.Status.Phase)
		if pod.Status.Phase == api_v1.PodFailed {
			logrus.Errorf("Pod (%s) on namespace (%s) status is %s", pod.ObjectMeta.Name,
				pod.ObjectMeta.Namespace, pod.Status.Phase)

			c.exitWithError()
		} else if pod.Status.Phase == api_v1.PodSucceeded {

			logrus.Infof("Pod (%s) on namespace (%s) status is %s", pod.ObjectMeta.Name,
				pod.ObjectMeta.Namespace, pod.Status.Phase)
			c.exitNoError()
		} else if pod.Status.Phase == api_v1.PodPending {
			logrus.Infof("Pod (%s) on namespace (%s) status is %s", pod.ObjectMeta.Name,
				pod.ObjectMeta.Namespace, pod.Status.Phase)
			if c.hasErrorWhenStartingContainer(pod) {
				c.exitWithError()
			}
		} else if pod.Status.Phase == api_v1.PodRunning {

			logrus.Infof("Pod Job Started (%s) on namespace (%s) status is %s", pod.ObjectMeta.Name,
				pod.ObjectMeta.Namespace, pod.Status.Phase)
			go c.tail.DoTail()

		} else {
			logrus.Infof("Pod (%s) on namespace (%s) status is %s", pod.ObjectMeta.Name,
				pod.ObjectMeta.Namespace, pod.Status.Phase)
			return false
		}
	}

	return false
}

func (c *Controller) hasErrorWhenStartingContainer(pod *api_v1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		logrus.Infof("Container (%s) status is %s", containerStatus.Name,
			containerStatus.State.Waiting)

		if containerStatus.State.Waiting.Reason != "ContainerCreating" {
			logrus.Errorf("Unable to pull image - [%s]", containerStatus.State.Waiting.Reason)
			return true
		}

	}
	return false

}

func (c *Controller) cleanup() {
	c.tail.CloseTail()
	if c.pod != nil {
		podClient := c.clientset.CoreV1().Pods(c.pod.Namespace)
		logrus.Infof("Cleaning up pod %s on namespace %s", c.GetPodName(), c.GetPodNamespace())
		if err := podClient.Delete(c.pod.Name, &meta_v1.DeleteOptions{}); err != nil {
			logrus.Errorf("Unable to clean up pod %s on namespace %s", c.GetPodName(), c.GetPodNamespace())
			panic(err)
		}
		logrus.Infof("Pod %s on namespace %s is deleted.", c.GetPodName(), c.GetPodNamespace())
	}

}

func (c *Controller) exitWithError() {

	c.cleanup()
	os.Exit(1)
}

func (c *Controller) exitNoError() {
	c.cleanup()
	os.Exit(0)
}

func shutdownHook(c *Controller, sigterm <-chan os.Signal) {
	for {
		logrus.Infof("shutdown hook called.")

		s := <-sigterm
		switch s {
		// kill -SIGHUP XXXX
		case syscall.SIGHUP:
			logrus.Errorf("Terminating pod %s on namespace %s", c.GetPodName(), c.GetPodNamespace())
			c.exitWithError()

		// kill -SIGINT XXXX or Ctrl+c
		case syscall.SIGINT:
			logrus.Errorf("Terminating pod %s on namespace %s", c.GetPodName(), c.GetPodNamespace())
			c.exitWithError()

		// kill -SIGTERM XXXX
		case syscall.SIGTERM:
			logrus.Errorf("Terminating pod %s on namespace %s", c.GetPodName(), c.GetPodNamespace())
			c.exitWithError()

		// kill -SIGQUIT XXXX
		case syscall.SIGQUIT:
			logrus.Errorf("Terminating pod %s on namespace %s", c.GetPodName(), c.GetPodNamespace())
			c.exitWithError()

		default:
			logrus.Errorf("Unknown signal.")
		}
	}
}

// GetPodName - Retrieve the Pod name associated with this controller
func (c *Controller) GetPodName() string {
	var name string
	if c.pod != nil {
		name = c.pod.Name
	}
	return name
}

// GetPodNamespace - Retrieves the Namespace associated with this controller
func (c *Controller) GetPodNamespace() string {
	var namespace string
	if c.pod != nil {
		namespace = c.pod.Namespace
	}
	return namespace
}

func (c *Controller) initTail() {
	if c.tail == nil {
		c.tail = &Tail{
			TailClosed:     make(chan int, 1),
			clientset:      c.clientset,
			podName:        c.GetPodName(),
			podNamespace:   c.GetPodNamespace(),
			hasTailStarted: false,
		}
	}
}
