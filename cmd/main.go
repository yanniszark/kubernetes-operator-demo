package cmd

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/yanniszark/kubernetes-operator-demo/pkg/controller"
)

func main() {

	// Set logging output to stdout
	log.SetOutput(os.Stdout)

	// Create a channel to receive OS signals
	sigs := make(chan os.Signal, 1)
	// Create a channel to receive stop signal
	stop := make(chan struct{})

	// Register the sigs channel to receive SIGTERM
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Goroutines need to add themselves to this to be waited on when they finish
	wg := &sync.WaitGroup{}

	// Register CMD flag indicating if the controller is running outside of Kubernetes
	runOutsideCluster := flag.Bool("run-outside-cluster", false, "Set this flag when running outside of the cluster.")
	flag.Parse()

	// Create clientset for interacting with the kubernetes cluster
	clientset, err := newClientSet(*runOutsideCluster)

	// Initialize controller

	if err != nil {
		panic(err.Error())
	}

	controller.NewNamespaceController(clientset).Run(stop, wg)

	// Wait for signals (this hangs until a signal arrives)
	<-sigs
	log.Printf("Shutting down...")
}

func newClientSet(runOutsideCluster bool) (*kubernetes.Clientset, error) {
	kubeConfigLocation := ""

	if runOutsideCluster == true {
		homeDir := os.Getenv("HOME")
		kubeConfigLocation = filepath.Join(homeDir, ".kube", "config")
	}

	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigLocation)

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)

}
