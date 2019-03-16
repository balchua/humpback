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

var appParams handler.AppParameters

var rootCmd = &cobra.Command{
	Use:   "humpback",
	Short: "Humpback is a helper command to deploy a pod and monitor till its completion.",
	Long:  `Humpback is a helper command to deploy a pod and monitor till its completion.  This is useful to integrate with external job orchestration tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Info("Calling getClient()")
		clientset, _ = getClient()
		jobHandler := handler.Init(appParams, clientset)
		jobHandler.Schedule()

		if jobHandler.PodScheduled {

			listOptions := metav1.ListOptions{
				LabelSelector: jobHandler.Selector,
			}
			controller.Start(clientset, appParams.Namespace, listOptions)
		} else {
			logrus.Errorf("Unable to find app")
			os.Exit(1)
		}
	},
}

func init() {
	appParams = handler.AppParameters{}
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true})

	// Output to stdout instead of the default stderr, could also be a file.
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logrus.SetLevel(logrus.DebugLevel)

	rootCmd.PersistentFlags().StringVarP(&appParams.Application, "application", "a", "", "The application to run.")
	rootCmd.MarkPersistentFlagRequired("application")
	rootCmd.PersistentFlags().StringVarP(&appParams.Command, "command", "c", "", "The Job to run.")
	rootCmd.MarkPersistentFlagRequired("command")
	rootCmd.PersistentFlags().StringVarP(&appParams.Namespace, "namespace", "n", "", "Deploy Pod to which namespace.")
	rootCmd.MarkPersistentFlagRequired("namespace")
	rootCmd.PersistentFlags().StringVarP(&appParams.KubeConfig, "kubeconfig", "k", "", "Kubernetes Configuration to use, when running outside the cluster.")

	rootCmd.PersistentFlags().StringVarP(&appParams.ConfigPath, "appconfig-path", "p", "", "The path to the application config, i.e humpback.yaml.  Do not include the filename, just the directory.")

}

func getClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if appParams.KubeConfig == "" {
		logrus.Debug("Using in cluster config")
		config, err = rest.InClusterConfig()
		// in cluster access
	} else {
		logrus.Debug("Using out of cluster config")
		config, err = clientcmd.BuildConfigFromFlags("", appParams.KubeConfig)
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
