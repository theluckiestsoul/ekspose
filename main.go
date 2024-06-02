package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	homeDir, _ := os.UserHomeDir()
	path.Join()
	defaultKubeConfigPath := filepath.Join(homeDir, ".kube", "config")
	cfgFile := flag.String("kubeconfig", defaultKubeConfigPath, "kubeconfig file")
	cfg, err := clientcmd.BuildConfigFromFlags("", *cfgFile)
	if err != nil {
		fmt.Println("Using in-cluster config")
		cfg, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("failed to build in-cluster config: %v", err)
		}
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create clientset: %v", err)
	}

	ch := make(chan struct{})
	infrmrs := informers.NewSharedInformerFactory(cs, 10*time.Minute)
	c := newController(cs, infrmrs.Apps().V1().Deployments())
	infrmrs.Start(ch)
	c.run(ch)
	fmt.Println(infrmrs)
}
