package controller

import (
	"bufio"
	"fmt"

	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Tail object
type Tail struct {
	clientset      kubernetes.Interface
	podName        string
	podNamespace   string
	hasTailStarted bool
	TailClosed     chan int
}

// DoTail - Start tailing the pod logs.
func (t *Tail) DoTail() {
	logrus.Debug("starting tail logs.....")
	if t.hasTailStarted {
		logrus.Debug("Tail has already started.....")
		return
	}
	t.hasTailStarted = true
	podClient := t.clientset.CoreV1().Pods(t.podNamespace)
	podLogOptions := api_v1.PodLogOptions{Follow: true}

	req := podClient.GetLogs(t.podName, &podLogOptions)
	logrus.Debug("getting logs.....")
	stream, err := req.Stream()
	logrus.Debug("getting logs.....")
	if err != nil {
		logrus.Errorf("Error opening stream to %s/%s: \n", t.podNamespace, t.podName)
		t.hasTailStarted = false
		return
	}
	defer stream.Close()

	go func() {

		<-t.TailClosed
		logrus.Debug("Log Stream Closing.....")
		stream.Close()
	}()

	reader := bufio.NewReader(stream)

	for {

		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}

		str := string(line)

		fmt.Print(str)
	}

}

// CloseTail - signal to close the tail channel
func (t *Tail) CloseTail() {
	if t != nil {
		t.TailClosed <- 1
	}
}
