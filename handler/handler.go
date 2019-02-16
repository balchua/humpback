package handler

import (
	"bytes"
	"log"

	"github.com/balchua/pod-runner/config"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"html/template"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var clientset *kubernetes.Clientset

//Handler - this holds the information to help render the template and schedule the pod to the cluster.
type Handler struct {
	clientset             *kubernetes.Clientset
	applicationToSchedule string
	namespace             string
	command               string
	appConfig             config.ApplicationConfiguration
	renderedYaml          string
	Selector              string
	PodScheduled          bool
}

//Init - Initializes the Handler with necessary information
func Init(applicationToSchedule string, namespace string, command string,
	clientset *kubernetes.Clientset) (h *Handler) {

	handler := &Handler{}
	handler.clientset = clientset
	handler.namespace = namespace
	handler.command = command
	handler.applicationToSchedule = applicationToSchedule
	handler.getAppConfig()

	return handler

}

//Schedule -  Renders the template and Schedule the Pod to the namespace
func (h *Handler) Schedule() {
	h.PodScheduled = false
	if h.renderTemplate() {
		h.schedulePod()
		h.PodScheduled = true
		h.Selector = "appUnique=" + h.appConfig.Name + "-" + h.appConfig.UniqueId
	}

}

func (h *Handler) schedulePod() {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, _, err := decode([]byte(h.renderedYaml), nil, nil)
	if err != nil {
		logrus.Errorf("%#v", err)
	}

	pod := obj.(*corev1.Pod)

	pod.Spec.RestartPolicy = "Never"

	logrus.Debugf("%#v\n", pod)
	podClient := h.clientset.CoreV1().Pods(h.namespace)

	podClient.Create(pod)
}

func (h *Handler) renderTemplate() bool {
	if h.appConfig.Name != "" {
		t, _ := template.ParseFiles(h.appConfig.Template)
		var tpl bytes.Buffer
		t.Execute(&tpl, h.appConfig)
		h.renderedYaml = tpl.String()
		logrus.Debugf("template content: %s", h.renderedYaml)
		return true
	}
	return false
}

func (h *Handler) getAppConfig() {
	viper.SetConfigName("pod-runner")
	viper.AddConfigPath(".")
	var configuration config.Configuration

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&configuration)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}
	for _, app := range configuration.Applications {
		logrus.Infof("Config is %s", app.Name)
		logrus.Infof("Config is %s", app.Container.Image)
		logrus.Infof("Template is %s", app.Template)
		logrus.Infof("Config is %s", app.Container.ResourceRequest.Memory)

		if h.applicationToSchedule == app.Name {
			id := xid.New()
			app.UniqueId = id.String()
			app.Container.Arguments = h.command
			h.appConfig = app
		}

	}

}
