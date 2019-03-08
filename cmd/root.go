package cmd

import (
	"fmt"
	"os"

	"github.com/balchua/humpback/controller"
	"github.com/balchua/humpback/handler"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var clientset *kubernetes.Clientset

var application string
var command string
var namespace string
var kubeConfig string

var rootCmd = &cobra.Command{
	Use:   "humpback",
	Short: "Humpback is a helper command to deploy a pod and monitor till its completion.",
	Long:  `Humpback is a helper command to deploy a pod and monitor till its completion.  This is useful to integrate with external job orchestration tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Info("Calling getClient()")
		clientset, _ = getClient()
		jobHandler := handler.Init(application, namespace, command, clientset)
		jobHandler.Schedule()

		if jobHandler.PodScheduled {

			listOptions := metav1.ListOptions{
				LabelSelector: jobHandler.Selector,
			}
			controller.Start(clientset, namespace, listOptions)
		} else {
			logrus.Errorf("Unable to find app")
			os.Exit(1)
		}
	},
}

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true})

	// Output to stdout instead of the default stderr, could also be a file.
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logrus.SetLevel(logrus.DebugLevel)

	rootCmd.PersistentFlags().StringVarP(&application, "application", "a", "", "The application to run.")
	rootCmd.PersistentFlags().StringVarP(&command, "command", "c", "", "The Job to run.")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Deploy Pod to which namespace.")
	rootCmd.PersistentFlags().StringVarP(&kubeConfig, "kube-config", "k", "", "Kubernetes Configuration to use, when running outside the cluster.")

}

func getClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if kubeConfig == "" {
		logrus.Debug("Using in cluster config")
		config, err = rest.InClusterConfig()
		// in cluster access
	} else {
		logrus.Debug("Using out of cluster config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

//Execute this marks the entry point of execution
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
